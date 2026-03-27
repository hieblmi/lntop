package models

import "github.com/hieblmi/lntop/logging"

type Route struct {
	// TimeLock: The cumulative (final) time lock across the entire route.
	// This is the CLTV value that should be extended to the first hop in the route.
	// All other hops will decrement the time-lock as advertised,
	// leaving enough time for all hops to wait for
	// or present the payment preimage to complete the payment.
	TimeLock uint32
	// Fee: The sum of the fees paid at each hop within the final route.
	// In the case of a one-hop payment, this value will be zero as we
	// don’t need to pay a fee it ourself.
	Fee int64
	// The total amount of funds required to complete a payment over this route.
	// This value includes the cumulative fees at each hop.
	// As a result, the HTLC extended to the first-hop in the route will need
	// to have at least this many satoshis, otherwise the route will fail
	// at an intermediate node due to an insufficient amount of fees.
	Amount int64
	// FeeMsat is the total route fee in millisatoshis.
	FeeMsat int64
	// AmountMsat is the total route amount in millisatoshis.
	AmountMsat int64
	// FirstHopAmountMsat is the amount sent over the first hop in millisatoshis.
	FirstHopAmountMsat int64

	Hops []*Hop
}

func (r Route) MarshalLogObject(enc logging.ObjectEncoder) error {
	enc.AddUint32("time_lock", r.TimeLock)
	enc.AddInt64("fee", r.Fee)
	enc.AddInt64("Amount", r.Amount)
	enc.AddInt64("fee_msat", r.FeeMsat)
	enc.AddInt64("amount_msat", r.AmountMsat)

	return nil
}

type Hop struct {
	// ChanID: The unique channel ID for the channel.
	// The first 3 bytes are the block height,
	// the next 3 the index within the block,
	// and the last 2 bytes are the output index for the channel.
	ChanID       uint64
	ChanCapacity int64
	Amount       int64
	AmountMsat   int64
	Fee          int64
	FeeMsat      int64
	Expiry       uint32
	PubKey       string
	Alias        string
}
