package views

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/hieblmi/lntop/ui/models"
)

var versionReg = regexp.MustCompile(`(\d+\.)?(\d+\.)?(\*|\d+)`)

// headerStyle renders a full-width gradient bar for the header.
var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#ffffff")).
	Background(lipgloss.Color("#5b37b7"))

type Header struct {
	Info *models.Info
}

func (h *Header) Render(width int) string {
	if h.Info == nil || h.Info.Info == nil {
		return ""
	}

	version := h.Info.Version
	matches := versionReg.FindStringSubmatch(h.Info.Version)
	if len(matches) > 0 {
		version = matches[0]
	}

	chain := ""
	if len(h.Info.Chains) > 0 {
		chain = h.Info.Chains[0]
	}

	network := h.Info.Network
	if network == "" {
		network = "mainnet"
	}

	syncLabel := " [syncing]"
	syncStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffcc00")).
		Background(lipgloss.Color("#5b37b7")).
		Bold(true)
	if h.Info.Synced {
		syncLabel = " [synced]"
		syncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff88")).
			Background(lipgloss.Color("#5b37b7")).
			Bold(true)
	}

	// Build the alias badge with a distinct background.
	alias := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#7c3aed")).
		Bold(true).
		Padding(0, 1).
		Render(h.Info.Alias)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a78bfa")).
		Background(lipgloss.Color("#5b37b7"))
	valStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e0e0e0")).
		Background(lipgloss.Color("#5b37b7"))

	content := fmt.Sprintf("%s %s %s %s%s %s%d %s%d",
		alias,
		valStyle.Render("lnd-v"+version),
		valStyle.Render(chain+" "+network),
		syncStyle.Render(syncLabel),
		"",
		labelStyle.Render(" height:"), h.Info.BlockHeight,
		labelStyle.Render(" peers:"), h.Info.NumPeers,
	)

	// Pad the header bar to full width.
	vis := lipgloss.Width(content)
	if vis < width {
		content += strings.Repeat(" ", width-vis)
	}

	return headerStyle.Width(width).MaxWidth(width).Render(content)
}

func NewHeader(info *models.Info) *Header {
	return &Header{Info: info}
}
