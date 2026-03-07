package ui

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hieblmi/lntop/app"
	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/events"
	"github.com/hieblmi/lntop/logging"
	netmodels "github.com/hieblmi/lntop/network/models"
	uimodels "github.com/hieblmi/lntop/ui/models"
	"github.com/hieblmi/lntop/ui/views"
)

func TestCurrentTableViewFollowsMenuSelection(t *testing.T) {
	menu := views.NewMenu()
	menu.SetCurrent(views.CHANNELS)

	m := &model{
		activeView: views.CHANNELS,
		menuOpen:   true,
		views: &views.Views{
			Menu: menu,
		},
	}

	m.views.Menu.CursorDown()

	if got := m.currentTableView(); got != views.TRANSACTIONS {
		t.Fatalf("currentTableView() = %q, want %q", got, views.TRANSACTIONS)
	}
	if m.activeView != views.CHANNELS {
		t.Fatalf("activeView changed during menu preview: got %q", m.activeView)
	}
}

func TestHandleMenuEnterCommitsSelection(t *testing.T) {
	menu := views.NewMenu()
	menu.SetCurrent(views.CHANNELS)
	menu.CursorDown()

	m := &model{
		activeView: views.CHANNELS,
		menuOpen:   true,
		views: &views.Views{
			Menu: menu,
		},
	}

	_, _ = m.handleMenuKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.activeView != views.TRANSACTIONS {
		t.Fatalf("activeView = %q, want %q", m.activeView, views.TRANSACTIONS)
	}
	if m.menuOpen {
		t.Fatalf("menu should close after enter")
	}
}

func TestHandleKeyClosingMenuCommitsPreviewSelection(t *testing.T) {
	menu := views.NewMenu()
	menu.SetCurrent(views.CHANNELS)
	menu.CursorDown()

	m := &model{
		activeView: views.CHANNELS,
		menuOpen:   true,
		views: &views.Views{
			Menu: menu,
		},
	}

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyF2})

	if m.activeView != views.TRANSACTIONS {
		t.Fatalf("activeView = %q, want %q", m.activeView, views.TRANSACTIONS)
	}
	if m.menuOpen {
		t.Fatalf("menu should close after F2")
	}
}

func TestPulseTickAdvancesFrame(t *testing.T) {
	channelModel := uimodels.NewChannels()
	m := &model{
		activeView: views.CHANNELS,
		views: &views.Views{
			Channels: views.NewChannels(nil, channelModel),
		},
	}

	_, cmd := m.Update(pulseTickMsg{})

	if m.pulseFrame != 1 {
		t.Fatalf("pulseFrame = %d, want 1", m.pulseFrame)
	}
	if cmd != nil {
		t.Fatalf("expected no follow-up pulse tick when no animation is active")
	}
}

func TestEnsurePulseTickStartsWhenChannelAlertsActive(t *testing.T) {
	channelModel := uimodels.NewChannels()
	channelModel.Add(&netmodels.Channel{
		ChannelPoint:     "chan-1",
		UnsettledBalance: 1,
	})

	m := &model{
		activeView: views.CHANNELS,
		views: &views.Views{
			Channels: views.NewChannels(nil, channelModel),
		},
	}

	cmd := m.ensurePulseTick()
	if cmd == nil {
		t.Fatalf("expected pulse tick cmd when channel alerts are active")
	}
	if !m.pulseActive {
		t.Fatalf("expected pulseActive to be set")
	}
}

func TestStartupStaysActiveWhileInfoSchedulesChannelReload(t *testing.T) {
	channelModel := uimodels.NewChannels()
	channelModel.Add(&netmodels.Channel{
		ChannelPoint: "chan-1",
		ID:           1 << 40,
		Age:          0,
	})

	m := &model{
		app:        &app.App{},
		activeView: views.CHANNELS,
		models: &uimodels.Models{
			Info:     &uimodels.Info{},
			Channels: channelModel,
		},
		views: &views.Views{
			Channels: views.NewChannels(nil, channelModel),
		},
		startupActive: true,
		startupTasks: map[string]bool{
			"info": true,
		},
		infoLoading: true,
	}

	_, cmd := m.Update(infoLoadedMsg{
		info: &netmodels.Info{BlockHeight: 100},
	})

	if !m.startupActive {
		t.Fatalf("startup should remain active while follow-up channel load is pending")
	}
	if !m.channelsLoading {
		t.Fatalf("expected follow-up channels load to start")
	}
	if cmd == nil {
		t.Fatalf("expected follow-up command batch")
	}
}

func TestStartupWaitsForCompletionTickBeforeEnteringApp(t *testing.T) {
	m := &model{
		startupActive:    true,
		startupFinishing: true,
		startupTasks:     map[string]bool{},
		models: &uimodels.Models{
			WalletBalance: &uimodels.WalletBalance{},
		},
	}

	_, cmd := m.Update(walletBalanceLoadedMsg{
		balance: &netmodels.WalletBalance{},
	})

	if !m.startupActive {
		t.Fatalf("startup should remain visible until completion tick")
	}
	if cmd == nil {
		t.Fatalf("expected completion command")
	}

	_, _ = m.Update(startupCompleteMsg{})

	if m.startupActive {
		t.Fatalf("startup should finish after completion tick")
	}
	if m.startupFinishing {
		t.Fatalf("startup finishing flag should be cleared")
	}
}

func TestStartupTaskErrorRetriesInsteadOfCompleting(t *testing.T) {
	logger, err := logging.NewNopLogger()
	if err != nil {
		t.Fatalf("NewNopLogger() error = %v", err)
	}

	m := &model{
		startupActive: true,
		startupTasks: map[string]bool{
			"wallet": true,
		},
		walletBalanceLoading: true,
		logger:               logger,
		models: &uimodels.Models{
			WalletBalance: &uimodels.WalletBalance{},
		},
	}

	_, cmd := m.Update(walletBalanceLoadedMsg{
		err: errors.New("timeout"),
	})

	if !m.startupActive {
		t.Fatalf("startup should stay active after a failed startup task")
	}
	if !m.startupTasks["wallet"] {
		t.Fatalf("wallet task should remain pending after failure")
	}
	if cmd == nil {
		t.Fatalf("expected retry command after startup task failure")
	}
}

func TestApplySettingsEditUpdatesConfiguredQueries(t *testing.T) {
	m := &model{
		app:        &app.App{},
		activeView: views.FWDINGHIST,
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1d", MaxNumEvents: 50},
			Received:   &uimodels.Received{},
		},
	}

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	if !m.settingsOpen {
		t.Fatalf("expected settings popup to open")
	}
	if m.forwardingWindowInput != "-1d" {
		t.Fatalf("forwardingWindowInput = %q, want %q", m.forwardingWindowInput, "-1d")
	}
	if m.forwardingMaxEventsInput != "50" {
		t.Fatalf("forwardingMaxEventsInput = %q, want %q", m.forwardingMaxEventsInput, "50")
	}

	m.forwardingWindowInput = "-1w"
	m.forwardingMaxEventsInput = "75"
	m.receivedStartDateInput = "2025-09-01"
	cmd := m.applySettingsEdit()

	if m.models.FwdingHist.StartTime != "-1w" {
		t.Fatalf("StartTime = %q, want %q", m.models.FwdingHist.StartTime, "-1w")
	}
	if m.models.FwdingHist.MaxNumEvents != 75 {
		t.Fatalf("MaxNumEvents = %d, want 75", m.models.FwdingHist.MaxNumEvents)
	}
	wantStart, err := parseReceivedStartDate("2025-09-01")
	if err != nil {
		t.Fatalf("parseReceivedStartDate() error = %v", err)
	}
	if m.models.Received.StartDateUnix != wantStart {
		t.Fatalf("StartDateUnix = %d, want %d", m.models.Received.StartDateUnix, wantStart)
	}
	if m.settingsOpen {
		t.Fatalf("expected settings popup to close")
	}
	if cmd == nil {
		t.Fatalf("expected reload commands")
	}
}

func TestApplySettingsEditRejectsInvalidValue(t *testing.T) {
	m := &model{
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1d"},
			Received:   &uimodels.Received{},
		},
		settingsOpen:          true,
		forwardingWindowInput: "yesterday",
	}

	cmd := m.applySettingsEdit()

	if cmd != nil {
		t.Fatalf("expected no command for invalid input")
	}
	if !m.settingsOpen {
		t.Fatalf("settings popup should stay open on invalid input")
	}
	if m.settingsErr == "" {
		t.Fatalf("expected validation error")
	}
	if m.models.FwdingHist.StartTime != "-1d" {
		t.Fatalf("StartTime = %q, want %q", m.models.FwdingHist.StartTime, "-1d")
	}
}

func TestHandleKeyF9StartsSettingsWithoutChangingView(t *testing.T) {
	m := &model{
		activeView: views.CHANNELS,
		inDetail:   true,
		menuOpen:   true,
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1w"},
			Received:   &uimodels.Received{},
		},
	}

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyF9})

	if m.activeView != views.CHANNELS {
		t.Fatalf("activeView = %q, want %q", m.activeView, views.CHANNELS)
	}
	if !m.inDetail {
		t.Fatalf("detail mode should stay unchanged")
	}
	if !m.menuOpen {
		t.Fatalf("menu state should stay unchanged")
	}
	if !m.settingsOpen {
		t.Fatalf("expected settings popup to open")
	}
	if m.forwardingWindowInput != "-1w" {
		t.Fatalf("forwardingWindowInput = %q, want %q", m.forwardingWindowInput, "-1w")
	}
	if cmd != nil {
		t.Fatalf("expected no command when only opening settings")
	}
}

func TestViewRendersSettingsPopup(t *testing.T) {
	channels := uimodels.NewChannels()
	channels.Add(&netmodels.Channel{
		Capacity:     10_000,
		LocalBalance: 4_000,
		RemotePubKey: "0123456789abcdef",
		Node:         &netmodels.Node{Alias: "alice"},
	})

	fwdingHist := &uimodels.FwdingHist{StartTime: "-1d"}
	fwdingHist.Update([]*netmodels.ForwardingEvent{{
		AmtOut:  8_000,
		FeeMsat: 80_000,
	}})

	models := &uimodels.Models{
		Info: &uimodels.Info{Info: &netmodels.Info{
			Alias:               "alice",
			Version:             "0.20.0-beta",
			Chains:              []string{"bitcoin"},
			Network:             "regtest",
			Synced:              true,
			BlockHeight:         100,
			NumPeers:            3,
			NumActiveChannels:   1,
			NumPendingChannels:  0,
			NumInactiveChannels: 0,
		}},
		Channels: channels,
		WalletBalance: &uimodels.WalletBalance{WalletBalance: &netmodels.WalletBalance{
			TotalBalance:              5_000,
			ConfirmedBalance:          4_000,
			UnconfirmedBalance:        1_000,
			LockedBalance:             200,
			ReservedBalanceAnchorChan: 300,
			AccountBalance:            map[string]*netmodels.WalletAccountBalance{},
		}},
		ChannelsBalance: &uimodels.ChannelsBalance{ChannelsBalance: &netmodels.ChannelsBalance{
			Balance: 4_000,
		}},
		Transactions: &uimodels.Transactions{},
		RoutingLog:   &uimodels.RoutingLog{},
		FwdingHist:   fwdingHist,
		Received:     &uimodels.Received{},
	}

	m := &model{
		width:                    120,
		height:                   28,
		activeView:               views.CHANNELS,
		models:                   models,
		views:                    views.New(config.Views{}, models),
		settingsOpen:             true,
		settingsCursor:           1,
		forwardingWindowInput:    "-1w",
		forwardingMaxEventsInput: "333",
		receivedStartDateInput:   "2025-09-01",
	}

	out := stripANSI(m.View())
	if !strings.Contains(out, "Data Settings") {
		t.Fatalf("popup title missing from view")
	}
	if !strings.Contains(out, "Forwarding Window") || !strings.Contains(out, "Forwarding Max Events") || !strings.Contains(out, "Received Start Date") {
		t.Fatalf("popup settings fields missing from view")
	}
	if !strings.Contains(out, "333") || !strings.Contains(out, "2025-09-01") {
		t.Fatalf("popup values missing from view")
	}
	if !strings.Contains(out, "Up/Down select  Enter apply  Esc cancel") {
		t.Fatalf("popup actions missing from view")
	}
	if !strings.Contains(out, "Accounting (FwdingHistory)") {
		t.Fatalf("base view should still render behind the popup")
	}
}

func TestHandleEventChannelsUpdatedRefreshesSummaryData(t *testing.T) {
	logger, err := logging.NewNopLogger()
	if err != nil {
		t.Fatalf("NewNopLogger() error = %v", err)
	}

	m := &model{
		app:    &app.App{},
		logger: logger,
		models: &uimodels.Models{
			Channels:        uimodels.NewChannels(),
			FwdingHist:      &uimodels.FwdingHist{},
			ChannelsBalance: &uimodels.ChannelsBalance{},
		},
	}

	cmd := m.handleEvent(events.NewWithData(events.ChannelsUpdated, []*netmodels.Channel{{
		ChannelPoint: "chan-1",
	}}))

	if m.models.Channels.Len() != 1 {
		t.Fatalf("expected channels update to be applied immediately")
	}
	if !m.channelsBalanceLoading {
		t.Fatalf("expected channel balance reload to start")
	}
	if !m.forwardingHistLoading {
		t.Fatalf("expected forwarding history reload to start")
	}
	if cmd == nil {
		t.Fatalf("expected reload command batch")
	}
}

var ansiTestRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiTestRE.ReplaceAllString(s, "")
}
