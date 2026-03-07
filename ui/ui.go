package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hieblmi/lntop/app"
	"github.com/hieblmi/lntop/events"
)

func Run(_ context.Context, app *app.App, sub chan *events.Event) error {
	m := newModel(app, sub)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
