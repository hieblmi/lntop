package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
