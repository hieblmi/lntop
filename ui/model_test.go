package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hieblmi/lntop/app"
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

func TestApplyForwardingWindowEditUpdatesStartTime(t *testing.T) {
	m := &model{
		app:        &app.App{},
		activeView: views.FWDINGHIST,
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1d", MaxNumEvents: 50},
		},
	}

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	if !m.forwardingWindowEditing {
		t.Fatalf("expected forwarding window editing to start")
	}
	if m.forwardingWindowInput != "-1d" {
		t.Fatalf("forwardingWindowInput = %q, want %q", m.forwardingWindowInput, "-1d")
	}

	m.forwardingWindowInput = "-1w"
	cmd := m.applyForwardingWindowEdit()

	if m.models.FwdingHist.StartTime != "-1w" {
		t.Fatalf("StartTime = %q, want %q", m.models.FwdingHist.StartTime, "-1w")
	}
	if m.forwardingWindowEditing {
		t.Fatalf("expected forwarding window editing to end")
	}
	if cmd == nil {
		t.Fatalf("expected forwarding history reload command")
	}
}

func TestApplyForwardingWindowEditRejectsInvalidValue(t *testing.T) {
	m := &model{
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1d"},
		},
		forwardingWindowEditing: true,
		forwardingWindowInput:   "yesterday",
	}

	cmd := m.applyForwardingWindowEdit()

	if cmd != nil {
		t.Fatalf("expected no command for invalid input")
	}
	if !m.forwardingWindowEditing {
		t.Fatalf("editing should stay active on invalid input")
	}
	if m.forwardingWindowErr == "" {
		t.Fatalf("expected validation error")
	}
	if m.models.FwdingHist.StartTime != "-1d" {
		t.Fatalf("StartTime = %q, want %q", m.models.FwdingHist.StartTime, "-1d")
	}
}

func TestHandleKeyF9StartsForwardingWindowEditorWithoutChangingView(t *testing.T) {
	m := &model{
		activeView: views.CHANNELS,
		inDetail:   true,
		menuOpen:   true,
		models: &uimodels.Models{
			FwdingHist: &uimodels.FwdingHist{StartTime: "-1w"},
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
	if !m.forwardingWindowEditing {
		t.Fatalf("expected forwarding window editing to start")
	}
	if m.forwardingWindowInput != "-1w" {
		t.Fatalf("forwardingWindowInput = %q, want %q", m.forwardingWindowInput, "-1w")
	}
	if cmd != nil {
		t.Fatalf("expected no command when only entering edit mode")
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
