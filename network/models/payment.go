package models

import "github.com/hieblmi/lntop/logging"

type PaymentKind int

const (
	PaymentKindInvoice PaymentKind = iota
	PaymentKindKeysend
)

func (k PaymentKind) String() string {
	switch k {
	case PaymentKindKeysend:
		return "keysend"
	default:
		return "invoice"
	}
}

type PaymentStatus int

const (
	PaymentStatusUnknown PaymentStatus = iota
	PaymentStatusInFlight
	PaymentStatusSucceeded
	PaymentStatusFailed
	PaymentStatusInitiated
)

func (s PaymentStatus) String() string {
	switch s {
	case PaymentStatusInFlight:
		return "in-flight"
	case PaymentStatusSucceeded:
		return "succeeded"
	case PaymentStatusFailed:
		return "failed"
	case PaymentStatusInitiated:
		return "initiated"
	default:
		return "unknown"
	}
}

type PaymentAttemptStatus int

const (
	PaymentAttemptStatusInFlight PaymentAttemptStatus = iota
	PaymentAttemptStatusSucceeded
	PaymentAttemptStatusFailed
)

func (s PaymentAttemptStatus) String() string {
	switch s {
	case PaymentAttemptStatusSucceeded:
		return "succeeded"
	case PaymentAttemptStatusFailed:
		return "failed"
	default:
		return "in-flight"
	}
}

type PaymentAttempt struct {
	AttemptID          uint64
	Status             PaymentAttemptStatus
	AttemptTimeNs      int64
	ResolveTimeNs      int64
	FailureCode        string
	FailureSourceIndex uint32
	FailureChannelID   uint64
	FailureHTLCMsat    uint64
	FailureCLTVExpiry  uint32
	FailureHeight      uint32
	PaymentPreimage    string
	Route              *Route
}

type Payment struct {
	PaymentHash     string
	PaymentError    string
	PaymentPreimage string
	PaymentRequest  string
	Status          PaymentStatus
	FailureReason   string
	CreationDate    int64
	CreationTimeNs  int64
	ValueSat        int64
	ValueMsat       int64
	FeeSat          int64
	FeeMsat         int64
	PaymentIndex    uint64
	Attempts        int
	Kind            PaymentKind
	AttemptDetails  []*PaymentAttempt
	PayReq          *PayReq
	Route           *Route
}

func (p *Payment) CreatedAtNs() int64 {
	if p == nil {
		return 0
	}
	if p.CreationTimeNs > 0 {
		return p.CreationTimeNs
	}
	return p.CreationDate * 1_000_000_000
}

func (p Payment) MarshalLogObject(enc logging.ObjectEncoder) error {
	enc.AddString("payment_hash", p.PaymentHash)
	enc.AddString("status", p.Status.String())
	enc.AddString("failure_reason", p.FailureReason)
	enc.AddString("payment_error", p.PaymentError)
	enc.AddInt64("value_sat", p.ValueSat)
	enc.AddInt64("fee_sat", p.FeeSat)
	enc.AddUint64("payment_index", p.PaymentIndex)

	return nil
}
