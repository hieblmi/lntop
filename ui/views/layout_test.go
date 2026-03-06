package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	uimodels "github.com/hieblmi/lntop/ui/models"
)

func TestSelectedRowSingleLine(t *testing.T) {
	row := "on 🧑‍🚀 Space HODLer 1,000 1 1,000 1"
	got := selectedRow(row, 40)

	if strings.Contains(got, "\n") {
		t.Fatalf("selected row wrapped into multiple lines")
	}
	if w := lipgloss.Width(got); w > safeRowWidth(40) {
		t.Fatalf("selected row width %d exceeds safe width %d", w, safeRowWidth(40))
	}
}

func TestMenuRenderWidthBounded(t *testing.T) {
	m := NewMenu()
	width := 16
	height := 20

	out := m.Render(width, height)
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("line %d width %d exceeds %d", i+1, w, width)
		}
	}
}

func TestRenderHeaderCellWidthBounded(t *testing.T) {
	out := renderHeaderCell("LAST UPDATE", 15, DefaultColStyle)

	if strings.Contains(out, "\n") {
		t.Fatalf("header cell wrapped into multiple lines")
	}
	if w := lipgloss.Width(out); w > 15 {
		t.Fatalf("header cell width %d exceeds 15", w)
	}
}

func TestHeaderRenderWidthBounded(t *testing.T) {
	h := NewHeader(&uimodels.Info{
		Info: &netmodels.Info{
			Alias:       "alice",
			Version:     "0.20.0-beta",
			Chains:      []string{"bitcoin"},
			Network:     "regtest",
			Synced:      true,
			BlockHeight: 136,
			NumPeers:    16,
		},
	})

	out := h.Render(80)
	if strings.Contains(out, "\n") {
		t.Fatalf("header wrapped into multiple lines")
	}
	if w := lipgloss.Width(out); w > 80 {
		t.Fatalf("header width %d exceeds 80", w)
	}
}

func TestChannelAlertPulseWidthStable(t *testing.T) {
	channels := &Channels{}

	channels.SetPulseFrame(0)
	first := channels.renderAlertValue("    3", true)
	channels.SetPulseFrame(1)
	second := channels.renderAlertValue("    3", true)

	if w := lipgloss.Width(first); w != 5 {
		t.Fatalf("first pulse width %d, want 5", w)
	}
	if w := lipgloss.Width(second); w != 5 {
		t.Fatalf("second pulse width %d, want 5", w)
	}
}

func TestChannelExitBlinkTriggeredOnZeroTransition(t *testing.T) {
	channels := &Channels{
		prevHTLC:       map[string]int{"chan": 2},
		prevUnsettled:  map[string]int64{"chan": 50},
		prevSent:       map[string]int64{"chan": 10},
		prevReceived:   map[string]int64{"chan": 20},
		htlcBlink:      make(map[string]int),
		unsettledBlink: make(map[string]int),
		sentFlash:      make(map[string]int),
		receivedFlash:  make(map[string]int),
	}

	channels.syncAlertTransitions([]*netmodels.Channel{{
		ChannelPoint:        "chan",
		UnsettledBalance:    0,
		PendingHTLC:         nil,
		TotalAmountSent:     11,
		TotalAmountReceived: 21,
	}})

	if channels.htlcBlink["chan"] == 0 {
		t.Fatalf("expected HTLC blink to start on zero transition")
	}
	if channels.unsettledBlink["chan"] == 0 {
		t.Fatalf("expected unsettled blink to start on zero transition")
	}
	if channels.sentFlash["chan"] == 0 {
		t.Fatalf("expected sent flash to start on increase")
	}
	if channels.receivedFlash["chan"] == 0 {
		t.Fatalf("expected received flash to start on increase")
	}
}

func TestSummaryRenderWidthBoundedWithAccounting(t *testing.T) {
	channels := uimodels.NewChannels()
	channels.Add(&netmodels.Channel{Capacity: 10_000})

	fwdingHist := &uimodels.FwdingHist{StartTime: "-1d"}
	fwdingHist.Update([]*netmodels.ForwardingEvent{{
		AmtOut: 8_000,
		Fee:    80,
	}})

	summary := NewSummary(
		&uimodels.Info{Info: &netmodels.Info{
			NumActiveChannels:   1,
			NumPendingChannels:  0,
			NumInactiveChannels: 0,
		}},
		&uimodels.ChannelsBalance{ChannelsBalance: &netmodels.ChannelsBalance{
			Balance: 4_000,
		}},
		&uimodels.WalletBalance{WalletBalance: &netmodels.WalletBalance{
			TotalBalance:              5_000,
			ConfirmedBalance:          4_000,
			UnconfirmedBalance:        1_000,
			LockedBalance:             200,
			ReservedBalanceAnchorChan: 300,
			AccountBalance: map[string]*netmodels.WalletAccountBalance{
				"default": {ConfirmedBalance: 4_000, UnconfirmedBalance: 1_000},
			},
		}},
		channels,
		fwdingHist,
	)
	summary.SetForwardingState(true, true, "-1w", "")
	summary.SetPulseFrame(1)

	out := summary.Render(100)
	if !strings.Contains(out, "Accounting") {
		t.Fatalf("summary should include accounting panel")
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 100 {
			t.Fatalf("line %d width %d exceeds 100", i+1, w)
		}
	}
}

func TestSummaryRenderFillsWideLayoutWidth(t *testing.T) {
	channels := uimodels.NewChannels()
	channels.Add(&netmodels.Channel{Capacity: 10_000})

	fwdingHist := &uimodels.FwdingHist{StartTime: "-1d"}
	fwdingHist.Update([]*netmodels.ForwardingEvent{{
		AmtOut:  8_000,
		FeeMsat: 80_000,
	}})

	summary := NewSummary(
		&uimodels.Info{Info: &netmodels.Info{
			NumActiveChannels:   1,
			NumPendingChannels:  0,
			NumInactiveChannels: 0,
		}},
		&uimodels.ChannelsBalance{ChannelsBalance: &netmodels.ChannelsBalance{
			Balance: 4_000,
		}},
		&uimodels.WalletBalance{WalletBalance: &netmodels.WalletBalance{
			TotalBalance:              5_000,
			ConfirmedBalance:          4_000,
			UnconfirmedBalance:        1_000,
			LockedBalance:             200,
			ReservedBalanceAnchorChan: 300,
			AccountBalance:            map[string]*netmodels.WalletAccountBalance{},
		}},
		channels,
		fwdingHist,
	)

	out := summary.Render(180)
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w != 180 {
			t.Fatalf("line %d width %d, want 180", i+1, w)
		}
	}
}

func TestChannelsRenderSlidesVisibleColumnsWithColumnCursor(t *testing.T) {
	chModel := uimodels.NewChannels()
	chModel.Add(&netmodels.Channel{
		ChannelPoint:  "abcdef:2",
		Node:          &netmodels.Node{Alias: "alice"},
		ID:            123,
		Capacity:      1000,
		LocalBalance:  500,
		RemoteBalance: 500,
	})

	view := NewChannels(&config.View{Columns: []string{
		"STATUS", "ALIAS", "LOCAL", "REMOTE", "ID", "CHANNEL_POINT",
	}}, chModel)

	initial := view.Render(60, 6)
	if strings.Contains(initial, "CHANNEL_POINT") {
		t.Fatalf("initial render should not show far-right column header")
	}

	for view.ColCursor < len(view.columns)-1 {
		view.ColumnRight()
	}
	scrolled := view.Render(60, 6)
	if !strings.Contains(scrolled, "CHANNEL_POINT") {
		t.Fatalf("render should show far-right column header when cursor moves right")
	}
}
