package views

import (
	"testing"

	"github.com/hieblmi/lntop/config"
	uimodels "github.com/hieblmi/lntop/ui/models"
)

func TestChannelsAllColumnsSortable(t *testing.T) {
	view := NewChannels(&config.View{Columns: []string{
		"STATUS", "ALIAS", "GAUGE", "LOCAL", "REMOTE", "CAP",
		"SENT", "RECEIVED", "HTLC", "UNSETTLED", "CFEE",
		"LAST UPDATE", "PRIVATE", "ID", "SCID", "NUPD",
		"BASE_OUT", "RATE_OUT", "BASE_IN", "RATE_IN",
		"INBOUND_BASE", "INBOUND_RATE", "AGE",
	}}, uimodels.NewChannels())

	for i, col := range view.columns {
		if col.sort == nil {
			t.Fatalf("channels column %q at index %d is not sortable", col.name, i)
		}
	}
}

func TestTransactionsAllColumnsSortable(t *testing.T) {
	view := NewTransactions(&config.View{Columns: []string{
		"DATE", "HEIGHT", "CONFIR", "AMOUNT", "FEE", "ADDRESSES", "TXHASH", "BLOCKHASH",
	}}, &uimodels.Transactions{})

	for i, col := range view.columns {
		if col.sort == nil {
			t.Fatalf("transactions column %q at index %d is not sortable", col.name, i)
		}
	}
}

func TestRoutingAllColumnsSortable(t *testing.T) {
	view := NewRouting(&config.View{Columns: []string{
		"DIR", "STATUS", "IN_ALIAS", "IN_CHANNEL", "IN_SCID", "IN_TIMELOCK", "IN_HTLC",
		"OUT_ALIAS", "OUT_CHANNEL", "OUT_SCID", "OUT_TIMELOCK", "OUT_HTLC",
		"AMOUNT", "FEE", "LAST UPDATE", "INBOUND_BASE_IN", "INBOUND_RATE_IN", "DETAIL",
	}}, &uimodels.RoutingLog{}, uimodels.NewChannels())

	for i, col := range view.columns {
		if col.sort == nil {
			t.Fatalf("routing column %q at index %d is not sortable", col.name, i)
		}
	}
}

func TestFwdingHistoryAllColumnsSortable(t *testing.T) {
	view := NewFwdingHist(&config.View{Columns: []string{
		"ALIAS_IN", "ALIAS_OUT", "AMT_IN", "AMT_OUT", "FEE",
		"TIMESTAMP_NS", "CHAN_ID_IN", "CHAN_ID_OUT", "INBOUND_BASE_IN", "INBOUND_RATE_IN",
	}}, &uimodels.FwdingHist{}, uimodels.NewChannels())

	for i, col := range view.columns {
		if col.sort == nil {
			t.Fatalf("forwarding history column %q at index %d is not sortable", col.name, i)
		}
	}
}

func TestReceivedAllColumnsSortable(t *testing.T) {
	view := NewReceived(&config.View{Columns: []string{
		"TYPE", "TIME", "AMOUNT", "MEMO", "R_HASH",
	}}, &uimodels.Received{})

	for i, col := range view.columns {
		if col.sort == nil {
			t.Fatalf("received column %q at index %d is not sortable", col.name, i)
		}
	}
}
