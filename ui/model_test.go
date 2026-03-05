package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
