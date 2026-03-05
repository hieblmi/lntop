package ui

import (
	"github.com/hieblmi/lntop/events"
)

// eventMsg wraps an LND event for delivery to the bubbletea update loop.
type eventMsg struct {
	event *events.Event
}
