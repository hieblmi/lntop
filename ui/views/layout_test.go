package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

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
