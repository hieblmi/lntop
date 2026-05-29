package loop

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/lightninglabs/loop/looprpc"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network/models"
)

// Backend talks to a local loopd over gRPC. It is created lazily by
// network.New when the user enables [loop] in their config. Ping() is called
// at startup but does not block UI startup — if loopd is not reachable the
// backend keeps its handle and individual calls will surface the error.
type Backend struct {
	cfg    *config.Loop
	logger logging.Logger

	mu     sync.Mutex // guards conn/client during lazy redial.
	conn   *grpc.ClientConn
	client looprpc.SwapClientClient
}

// New dials loopd. A failure to dial is non-fatal: it returns the Backend
// with a nil client, and callers can retry the dial later.
func New(c *config.Loop, logger logging.Logger) (*Backend, error) {
	b := &Backend{
		cfg:    c,
		logger: logger,
	}

	conn, err := newClientConn(c)
	if err != nil {
		logger.Info("loop: initial dial failed; will retry on demand",
			logging.Error(err))
		return b, nil
	}
	b.conn = conn
	b.client = looprpc.NewSwapClientClient(conn)
	return b, nil
}

// ensureClient returns the cached client, attempting one redial when nil.
// Safe to call concurrently — the Init() batch fires three loaders in
// parallel.
func (b *Backend) ensureClient() (looprpc.SwapClientClient, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.client != nil {
		return b.client, nil
	}
	conn, err := newClientConn(b.cfg)
	if err != nil {
		return nil, err
	}
	b.conn = conn
	b.client = looprpc.NewSwapClientClient(conn)
	return b.client, nil
}

// Ping issues a quick GetInfo so we can surface connectivity errors early.
func (b *Backend) Ping(ctx context.Context) error {
	clt, err := b.ensureClient()
	if err != nil {
		return err
	}
	_, err = clt.GetInfo(ctx, &looprpc.GetInfoRequest{})
	return err
}

func (b *Backend) Close() error {
	if b.conn == nil {
		return nil
	}
	return b.conn.Close()
}

// GetInfo returns daemon-level stats. The autoloop / static-address fields
// of LoopInfo are filled in by separate calls; this only populates the parts
// served by SwapClient.GetInfo.
func (b *Backend) GetInfo(ctx context.Context) (*models.LoopInfo, error) {
	clt, err := b.ensureClient()
	if err != nil {
		return nil, err
	}
	resp, err := clt.GetInfo(ctx, &looprpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	info := &models.LoopInfo{
		Version:    resp.Version,
		Network:    resp.Network,
		CommitHash: resp.CommitHash,
		RPCListen:  resp.RpcListen,
	}
	if resp.LoopOutStats != nil {
		info.OutPending = resp.LoopOutStats.PendingCount
		info.OutSuccess = resp.LoopOutStats.SuccessCount
		info.OutFail = resp.LoopOutStats.FailCount
		info.OutSumPend = resp.LoopOutStats.SumPendingAmt
		info.OutSumOk = resp.LoopOutStats.SumSucceededAmt
	}
	if resp.LoopInStats != nil {
		info.InPending = resp.LoopInStats.PendingCount
		info.InSuccess = resp.LoopInStats.SuccessCount
		info.InFail = resp.LoopInStats.FailCount
		info.InSumPend = resp.LoopInStats.SumPendingAmt
		info.InSumOk = resp.LoopInStats.SumSucceededAmt
	}
	return info, nil
}

// FillAutoloop merges autoloop status from GetLiquidityParams onto an
// existing LoopInfo. Errors are tolerated — the field is just left zero.
func (b *Backend) FillAutoloop(ctx context.Context, info *models.LoopInfo) {
	clt, err := b.ensureClient()
	if err != nil {
		return
	}
	params, err := clt.GetLiquidityParams(ctx, &looprpc.GetLiquidityParamsRequest{})
	if err != nil {
		b.logger.Debug("loop: GetLiquidityParams failed",
			logging.Error(err))
		return
	}
	info.AutoloopOn = params.Autoloop
	info.AutoloopBudget = params.AutoloopBudgetSat
}

// FillStaticSummary merges the static-address summary onto LoopInfo. Errors
// are tolerated — common when no static address has ever been created.
func (b *Backend) FillStaticSummary(ctx context.Context, info *models.LoopInfo) {
	clt, err := b.ensureClient()
	if err != nil {
		return
	}
	resp, err := clt.GetStaticAddressSummary(
		ctx, &looprpc.StaticAddressSummaryRequest{},
	)
	if err != nil {
		// A missing static address surfaces as a gRPC error. That's not
		// worth logging at warn level on every refresh.
		b.logger.Debug("loop: GetStaticAddressSummary failed",
			logging.Error(err))
		return
	}
	info.StaticAddress = resp.StaticAddress
	info.StaticDeposited = resp.ValueDepositedSatoshis
	info.StaticLoopedIn = resp.ValueLoopedInSatoshis
}

// ListSwaps returns the unified swap list, merging legacy ListSwaps,
// ListInstantOuts, and ListStaticAddressSwaps into one slice.
func (b *Backend) ListSwaps(ctx context.Context) ([]*models.LoopSwap, error) {
	clt, err := b.ensureClient()
	if err != nil {
		return nil, err
	}

	var out []*models.LoopSwap

	swaps, err := clt.ListSwaps(ctx, &looprpc.ListSwapsRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "loop: ListSwaps")
	}
	for _, s := range swaps.Swaps {
		out = append(out, swapStatusToLoopSwap(s))
	}

	// ListInstantOuts and ListStaticAddressSwaps may not be implemented on
	// older loopd versions — treat Unimplemented as empty rather than an
	// error so the rest of the Swaps tab still renders.
	instant, err := clt.ListInstantOuts(ctx, &looprpc.ListInstantOutsRequest{})
	switch {
	case err == nil:
		for _, s := range instant.Swaps {
			out = append(out, instantOutToLoopSwap(s))
		}
	case isUnimplemented(err):
		// older loopd: skip
	default:
		return nil, errors.Wrap(err, "loop: ListInstantOuts")
	}

	staticSwaps, err := clt.ListStaticAddressSwaps(
		ctx, &looprpc.ListStaticAddressSwapsRequest{},
	)
	switch {
	case err == nil:
		for _, s := range staticSwaps.Swaps {
			out = append(out, staticAddressSwapToLoopSwap(s))
		}
	case isUnimplemented(err):
		// older loopd: skip
	default:
		return nil, errors.Wrap(err, "loop: ListStaticAddressSwaps")
	}

	return out, nil
}

// ListDeposits returns every known static-address deposit (no state filter).
func (b *Backend) ListDeposits(ctx context.Context) ([]*models.LoopDeposit, error) {
	clt, err := b.ensureClient()
	if err != nil {
		return nil, err
	}
	resp, err := clt.ListStaticAddressDeposits(
		ctx, &looprpc.ListStaticAddressDepositsRequest{},
	)
	switch {
	case err == nil:
		// fall through to convert.
	case isUnimplemented(err):
		return nil, nil
	default:
		return nil, errors.Wrap(err, "loop: ListStaticAddressDeposits")
	}

	deposits := make([]*models.LoopDeposit, 0, len(resp.FilteredDeposits))
	for _, d := range resp.FilteredDeposits {
		deposits = append(deposits, depositToLoopDeposit(d))
	}
	return deposits, nil
}

// SubscribeSwaps streams swap state changes from loopd's Monitor RPC. The
// goroutine returns when ctx is canceled or the stream errors; restart logic
// lives at the pubsub layer.
func (b *Backend) SubscribeSwaps(
	ctx context.Context, ch chan<- *models.LoopSwap,
) error {
	clt, err := b.ensureClient()
	if err != nil {
		return err
	}
	stream, err := clt.Monitor(ctx, &looprpc.MonitorRequest{})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			st, ok := status.FromError(err)
			if ok && (st.Code() == codes.Canceled ||
				st.Code() == codes.Unavailable) {
				return nil
			}
			return err
		}
		select {
		case ch <- swapStatusToLoopSwap(msg):
		case <-ctx.Done():
			return nil
		}
	}
}

// swapStatusToLoopSwap converts a legacy looprpc.SwapStatus row (used by
// ListSwaps, SwapInfo, and the Monitor stream) to our unified model.
func swapStatusToLoopSwap(s *looprpc.SwapStatus) *models.LoopSwap {
	swap := &models.LoopSwap{
		ID:            hex.EncodeToString(s.IdBytes),
		State:         s.State.String(),
		FailureReason: failureReasonToString(s.FailureReason),
		Amount:        s.Amt,
		CostServer:    s.CostServer,
		CostOnchain:   s.CostOnchain,
		CostOffchain:  s.CostOffchain,
		HtlcAddrP2WSH: s.HtlcAddressP2Wsh,
		HtlcAddrP2TR:  s.HtlcAddressP2Tr,
		Label:         s.Label,
		LastHop:       s.LastHop,
		OutgoingChans: s.OutgoingChanSet,
	}
	switch s.Type {
	case looprpc.SwapType_LOOP_OUT:
		swap.Type = models.LoopSwapTypeOut
	case looprpc.SwapType_LOOP_IN:
		swap.Type = models.LoopSwapTypeIn
	default:
		swap.Type = models.LoopSwapTypeUnknown
	}
	if s.InitiationTime != 0 {
		t := time.Unix(0, s.InitiationTime)
		swap.InitiationTime = &t
	}
	if s.LastUpdateTime != 0 {
		t := time.Unix(0, s.LastUpdateTime)
		swap.LastUpdate = &t
	}
	return swap
}

func instantOutToLoopSwap(s *looprpc.InstantOut) *models.LoopSwap {
	return &models.LoopSwap{
		Type:           models.LoopSwapTypeInstantOut,
		ID:             hex.EncodeToString(s.SwapHash),
		State:          s.State,
		Amount:         int64(s.Amount),
		ReservationIDs: s.ReservationIds,
		SweepTxID:      s.SweepTxId,
	}
}

func staticAddressSwapToLoopSwap(s *looprpc.StaticAddressLoopInSwap) *models.LoopSwap {
	return &models.LoopSwap{
		Type:             models.LoopSwapTypeStaticIn,
		ID:               hex.EncodeToString(s.SwapHash),
		State:            s.State.String(),
		Amount:           s.SwapAmountSatoshis,
		PaymentReqAmount: s.PaymentRequestAmountSatoshis,
		DepositOutpoints: s.DepositOutpoints,
	}
}

func depositToLoopDeposit(d *looprpc.Deposit) *models.LoopDeposit {
	return &models.LoopDeposit{
		ID:                 d.Id,
		State:              depositStateToModel(d.State),
		Outpoint:           d.Outpoint,
		Value:              d.Value,
		ConfirmationHeight: d.ConfirmationHeight,
		BlocksUntilExpiry:  d.BlocksUntilExpiry,
		SwapHash:           d.SwapHash,
	}
}

func depositStateToModel(s looprpc.DepositState) models.LoopDepositState {
	switch s {
	case looprpc.DepositState_DEPOSITED:
		return models.LoopDepositStateDeposited
	case looprpc.DepositState_WITHDRAWING:
		return models.LoopDepositStateWithdrawing
	case looprpc.DepositState_WITHDRAWN:
		return models.LoopDepositStateWithdrawn
	case looprpc.DepositState_LOOPING_IN:
		return models.LoopDepositStateLoopingIn
	case looprpc.DepositState_LOOPED_IN:
		return models.LoopDepositStateLoopedIn
	case looprpc.DepositState_SWEEP_HTLC_TIMEOUT:
		return models.LoopDepositStateSweepHTLCTimeout
	case looprpc.DepositState_HTLC_TIMEOUT_SWEPT:
		return models.LoopDepositStateHTLCTimeoutSwept
	case looprpc.DepositState_PUBLISH_EXPIRED:
		return models.LoopDepositStatePublishExpired
	case looprpc.DepositState_WAIT_FOR_EXPIRY_SWEEP:
		return models.LoopDepositStateWaitForExpirySweep
	case looprpc.DepositState_EXPIRED:
		return models.LoopDepositStateExpired
	default:
		return models.LoopDepositStateUnknown
	}
}

func failureReasonToString(r looprpc.FailureReason) string {
	switch r {
	case looprpc.FailureReason_FAILURE_REASON_NONE:
		return ""
	case looprpc.FailureReason_FAILURE_REASON_OFFCHAIN:
		return "offchain"
	case looprpc.FailureReason_FAILURE_REASON_TIMEOUT:
		return "timeout"
	case looprpc.FailureReason_FAILURE_REASON_SWEEP_TIMEOUT:
		return "sweep-timeout"
	case looprpc.FailureReason_FAILURE_REASON_INSUFFICIENT_VALUE:
		return "insufficient-value"
	case looprpc.FailureReason_FAILURE_REASON_TEMPORARY:
		return "temporary"
	case looprpc.FailureReason_FAILURE_REASON_INCORRECT_AMOUNT:
		return "incorrect-amount"
	case looprpc.FailureReason_FAILURE_REASON_ABANDONED:
		return "abandoned"
	case looprpc.FailureReason_FAILURE_REASON_INSUFFICIENT_CONFIRMED_BALANCE:
		return "insufficient-confirmed-balance"
	case looprpc.FailureReason_FAILURE_REASON_INCORRECT_HTLC_AMT_SWEPT:
		return "incorrect-htlc-amt-swept"
	default:
		return fmt.Sprintf("unknown(%d)", r)
	}
}

// isUnimplemented returns true for the "Unimplemented" gRPC error. Some
// loopd builds (older releases, or feature-flag-disabled ones) refuse to
// handle newer RPCs and we don't want that to kill the whole swap-list
// load.
func isUnimplemented(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.Unimplemented
}
