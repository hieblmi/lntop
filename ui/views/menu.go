package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type menuItem struct {
	label    string
	viewName string
}

var baseMenuItems = []menuItem{
	{"CHANNEL", CHANNELS},
	{"TRANSAC", TRANSACTIONS},
	{"ROUTING", ROUTING},
	{"FWDINGHISTORY", FWDINGHIST},
	{"RECEIVED", RECEIVED},
	{"PAYMENTS", PAYMENTS},
}

var menuItemsWithLoop = append(append([]menuItem(nil), baseMenuItems...),
	menuItem{"LOOP", LOOP})

var (
	menuBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true, true, true, false).
			BorderForeground(lipgloss.Color("#5b37b7"))

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94a3b8"))

	menuActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e7ff")).
			Background(lipgloss.Color("#312e81")).
			Bold(true)

	menuTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a78bfa")).
			Bold(true)
)

type Menu struct {
	Cursor int
	items  []menuItem
}

func (m *Menu) Current() string {
	if m.Cursor >= 0 && m.Cursor < len(m.items) {
		return m.items[m.Cursor].viewName
	}
	return ""
}

func (m *Menu) SetCurrent(viewName string) {
	for i := range m.items {
		if m.items[i].viewName == viewName {
			m.Cursor = i
			return
		}
	}
}

func (m *Menu) CursorDown() {
	if m.Cursor < len(m.items)-1 {
		m.Cursor++
	}
}

func (m *Menu) CursorUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *Menu) Render(width, height int) string {
	if width < 1 {
		width = 1
	}
	// Border adds one visible cell on the right side.
	innerW := width - 1
	if innerW < 1 {
		innerW = 1
	}

	var b strings.Builder

	b.WriteString(menuTitleStyle.Render(padRight("MENU", innerW)))
	b.WriteString("\n")

	for i, item := range m.items {
		line := ansi.Truncate(fmt.Sprintf("%-*s", innerW, item.label), innerW, "")
		line = padRight(line, innerW)
		if i == m.Cursor {
			b.WriteString(menuActiveStyle.Render(line))
		} else {
			b.WriteString(menuItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Pad remaining height. Account for border top/bottom (2 lines).
	used := 1 + len(m.items) + 2
	for i := 0; i < height-used; i++ {
		b.WriteString("\n")
	}

	return menuBorderStyle.Render(b.String())
}

func NewMenu() *Menu { return &Menu{items: baseMenuItems} }

// NewMenuWithLoop returns a menu with the optional LOOP entry appended.
// Used when [loop] is enabled in the user's config.
func NewMenuWithLoop() *Menu { return &Menu{items: menuItemsWithLoop} }
