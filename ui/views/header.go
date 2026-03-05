package views

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hieblmi/lntop/ui/models"
)

var versionReg = regexp.MustCompile(`(\d+\.)?(\d+\.)?(\*|\d+)`)

// headerStyle renders a full-width gradient bar for the header.
var headerStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#120c2c"))

var (
	headerAliasStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#7c3aed")).
				Bold(true).
				Padding(0, 1)

	headerInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e7ff")).
			Background(lipgloss.Color("#312e81")).
			Bold(true).
			Padding(0, 1)

	headerSyncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6ee7b7")).
			Background(lipgloss.Color("#064e3b")).
			Bold(true).
			Padding(0, 1)

	headerSyncingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fde68a")).
				Background(lipgloss.Color("#78350f")).
				Bold(true).
				Padding(0, 1)

	headerMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c4b5fd")).
			Background(lipgloss.Color("#1f1b4d")).
			Bold(true).
			Padding(0, 1)
)

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

	syncLabel := "syncing"
	syncStyle := headerSyncingStyle
	if h.Info.Synced {
		syncLabel = "synced"
		syncStyle = headerSyncStyle
	}

	parts := []string{
		headerAliasStyle.Render(h.Info.Alias),
		headerInfoStyle.Render("lnd-v" + version),
		headerInfoStyle.Render(chain + " " + network),
		syncStyle.Render(syncLabel),
		headerMetaStyle.Render(fmt.Sprintf("height %d", h.Info.BlockHeight)),
		headerMetaStyle.Render(fmt.Sprintf("peers %d", h.Info.NumPeers)),
	}
	content := strings.Join(parts, " ")

	// Pad the header bar to full width.
	vis := lipgloss.Width(content)
	if vis > width {
		content = ansi.Truncate(content, width, "")
		vis = lipgloss.Width(content)
	}
	if vis < width {
		content += strings.Repeat(" ", width-vis)
	}

	return headerStyle.Width(width).MaxWidth(width).Render(content)
}

func NewHeader(info *models.Info) *Header {
	return &Header{Info: info}
}
