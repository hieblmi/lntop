package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
