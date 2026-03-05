package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var menuItems = []struct {
	label    string
	viewName string
}{
	{"CHANNEL", CHANNELS},
	{"TRANSAC", TRANSACTIONS},
	{"ROUTING", ROUTING},
	{"FWDHIST", FWDINGHIST},
	{"RECEIVED", RECEIVED},
}

var (
	menuBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true, true, true, false).
			BorderForeground(lipgloss.Color("#5b37b7"))

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94a3b8")).
			Padding(0, 1)

	menuActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e7ff")).
			Background(lipgloss.Color("#312e81")).
			Bold(true).
			Padding(0, 1)

	menuTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a78bfa")).
			Bold(true).
			Padding(0, 1)
)

type Menu struct {
	Cursor int
}

func (m *Menu) Current() string {
	if m.Cursor >= 0 && m.Cursor < len(menuItems) {
		return menuItems[m.Cursor].viewName
	}
	return ""
}

func (m *Menu) CursorDown() {
	if m.Cursor < len(menuItems)-1 {
		m.Cursor++
	}
}

func (m *Menu) CursorUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *Menu) Render(width, height int) string {
	// Border adds 1 char on right (left border disabled), top, bottom.
	// So inner content width = width - 1 (right border).
	innerW := width - 1
	if innerW < 12 {
		innerW = 12
	}

	var b strings.Builder

	b.WriteString(menuTitleStyle.Width(innerW).Render("MENU"))
	b.WriteString("\n")

	for i, item := range menuItems {
		line := fmt.Sprintf("%-9s", item.label)
		if i == m.Cursor {
			b.WriteString(menuActiveStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(menuItemStyle.Width(innerW).Render(line))
		}
		b.WriteString("\n")
	}

	// Pad remaining height. Account for border top/bottom (2 lines).
	used := 1 + len(menuItems) + 1 + 2
	for i := 0; i < height-used; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(innerW, "F2", "Close"))
	return menuBorderStyle.Width(innerW).Render(b.String())
}

func NewMenu() *Menu { return &Menu{} }
