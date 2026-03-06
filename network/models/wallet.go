package models

import "github.com/hieblmi/lntop/logging"

type WalletBalance struct {
	TotalBalance              int64
	ConfirmedBalance          int64
	UnconfirmedBalance        int64
	LockedBalance             int64
	ReservedBalanceAnchorChan int64
	AccountBalance            map[string]*WalletAccountBalance
}

type WalletAccountBalance struct {
	ConfirmedBalance   int64
	UnconfirmedBalance int64
}

func (m WalletBalance) MarshalLogObject(enc logging.ObjectEncoder) error {
	enc.AddInt64("total_balance", m.TotalBalance)
	enc.AddInt64("confirmed_balance", m.ConfirmedBalance)
	enc.AddInt64("unconfirmed_balance", m.UnconfirmedBalance)
	enc.AddInt64("locked_balance", m.LockedBalance)
	enc.AddInt64("reserved_balance_anchor_chan", m.ReservedBalanceAnchorChan)
	enc.AddInt("wallet_accounts", len(m.AccountBalance))

	return nil
}
