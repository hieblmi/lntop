package models

import "time"

// LoopSwapType discriminates the four kinds of loop activity surfaced by the
// LOOP view. They share a single rendered table because every row has an
// identifier, a state, an amount, and (where known) a timestamp.
type LoopSwapType int

const (
	LoopSwapTypeUnknown LoopSwapType = iota
	LoopSwapTypeOut                  // legacy off-chain → on-chain
	LoopSwapTypeIn                   // legacy on-chain → off-chain
	LoopSwapTypeInstantOut           // off-chain → on-chain via reservations
	LoopSwapTypeStaticIn             // static-address loop-in
)

func (t LoopSwapType) String() string {
	switch t {
	case LoopSwapTypeOut:
		return "OUT"
	case LoopSwapTypeIn:
		return "IN"
	case LoopSwapTypeInstantOut:
		return "INSTANT-OUT"
	case LoopSwapTypeStaticIn:
		return "STATIC-IN"
	default:
		return "?"
	}
}

// LoopSwap is the unified row type for the Swaps sub-tab. Fields that don't
// apply to a given swap type are zero/empty and rendered as "-".
type LoopSwap struct {
	Type           LoopSwapType
	ID             string // hex-encoded swap hash / id
	State          string
	FailureReason  string
	Amount         int64 // sats
	InitiationTime *time.Time
	LastUpdate     *time.Time
	CostServer     int64 // sats; legacy swaps only
	CostOnchain    int64 // sats; legacy swaps only
	CostOffchain   int64 // sats; legacy swaps only
	HtlcAddrP2WSH  string
	HtlcAddrP2TR   string
	Label          string
	LastHop        []byte
	OutgoingChans  []uint64
	// Type-specific extras.
	ReservationIDs   [][]byte // INSTANT-OUT
	SweepTxID        string   // INSTANT-OUT
	DepositOutpoints []string // STATIC-IN
	PaymentReqAmount int64    // STATIC-IN (invoiced amount)
}

// LoopDepositState mirrors looprpc.DepositState as plain strings so the
// network layer is the only place that needs to know about looprpc types.
type LoopDepositState int

const (
	LoopDepositStateUnknown LoopDepositState = iota
	LoopDepositStateDeposited
	LoopDepositStateWithdrawing
	LoopDepositStateWithdrawn
	LoopDepositStateLoopingIn
	LoopDepositStateLoopedIn
	LoopDepositStateSweepHTLCTimeout
	LoopDepositStateHTLCTimeoutSwept
	LoopDepositStatePublishExpired
	LoopDepositStateWaitForExpirySweep
	LoopDepositStateExpired
	LoopDepositStateOpeningChannel
	LoopDepositStateChannelPublished
)

func (s LoopDepositState) String() string {
	switch s {
	case LoopDepositStateDeposited:
		return "deposited"
	case LoopDepositStateWithdrawing:
		return "withdrawing"
	case LoopDepositStateWithdrawn:
		return "withdrawn"
	case LoopDepositStateLoopingIn:
		return "looping-in"
	case LoopDepositStateLoopedIn:
		return "looped-in"
	case LoopDepositStateSweepHTLCTimeout:
		return "sweep-htlc-to"
	case LoopDepositStateHTLCTimeoutSwept:
		return "htlc-to-swept"
	case LoopDepositStatePublishExpired:
		return "publ-expired"
	case LoopDepositStateWaitForExpirySweep:
		return "wait-exp-sweep"
	case LoopDepositStateExpired:
		return "expired"
	case LoopDepositStateOpeningChannel:
		return "opening-chan"
	case LoopDepositStateChannelPublished:
		return "chan-published"
	default:
		return "unknown"
	}
}

// LoopDeposit is the row type for the Deposits sub-tab.
type LoopDeposit struct {
	ID                 []byte
	State              LoopDepositState
	Outpoint           string
	Value              int64
	ConfirmationHeight int64
	BlocksUntilExpiry  int64
	SwapHash           []byte
}

// LoopInfo holds the data behind the Loop mini-header (daemon-level stats).
type LoopInfo struct {
	Version    string
	Network    string
	CommitHash string
	RPCListen  string

	OutPending uint64
	OutSuccess uint64
	OutFail    uint64
	OutSumPend int64
	OutSumOk   int64

	InPending uint64
	InSuccess uint64
	InFail    uint64
	InSumPend int64
	InSumOk   int64

	// AutoloopOn comes from GetLiquidityParams (separate RPC) and is filled in
	// by the model layer when available.
	AutoloopOn      bool
	AutoloopBudget  uint64
	StaticAddress   string
	StaticDeposited int64
	StaticLoopedIn  int64
}
