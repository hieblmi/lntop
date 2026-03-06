package views

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

// View name constants.
const (
	CHANNELS     = "channels"
	CHANNEL      = "channel"
	TRANSACTIONS = "transactions"
	TRANSACTION  = "transaction"
	ROUTING      = "routing"
	FWDINGHIST   = "fwdinghist"
	RECEIVED     = "received"
	MENU         = "menu"
)

// Views holds all view components.
type Views struct {
	Header       *Header
	Summary      *Summary
	Menu         *Menu
	Channels     *Channels
	Channel      *Channel
	Transactions *Transactions
	Transaction  *Transaction
	Routing      *Routing
	FwdingHist   *FwdingHist
	Received     *Received
}

func New(cfg config.Views, m *models.Models) *Views {
	return &Views{
		Header:       NewHeader(m.Info),
		Summary:      NewSummary(m.Info, m.ChannelsBalance, m.WalletBalance, m.Channels, m.FwdingHist),
		Menu:         NewMenu(),
		Channels:     NewChannels(cfg.Channels, m.Channels),
		Channel:      NewChannel(m.Channels),
		Transactions: NewTransactions(cfg.Transactions, m.Transactions),
		Transaction:  NewTransaction(m.Transactions),
		Routing:      NewRouting(cfg.Routing, m.RoutingLog, m.Channels),
		FwdingHist:   NewFwdingHist(cfg.FwdingHist, m.FwdingHist, m.Channels),
		Received:     NewReceived(cfg.Received, m.Received),
	}
}

// Shared styles.
var (
	HeaderBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#09097a"))

	DefaultColStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#09097a")).
			Foreground(lipgloss.Color("#c4b5fd")).
			Bold(true)

	// ActiveColStyle highlights the column under the cursor in the header bar.
	ActiveColStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#09097a")).
			Foreground(lipgloss.Color("#f5d0fe")).
			Bold(true)

	// SortedColStyle highlights the sorted column in the header bar.
	SortedColStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#09097a")).
			Foreground(lipgloss.Color("#ddd6fe")).
			Bold(true)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#312e81")).
				Foreground(lipgloss.Color("#e0e7ff"))

	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1b4b")).
			Foreground(lipgloss.Color("#94a3b8"))

	DetailHeaderStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#5b37b7")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#312e81")).
			Foreground(lipgloss.Color("#c4b5fd")).
			Bold(true).
			Padding(0, 1)

	footerDescStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1b4b")).
			Foreground(lipgloss.Color("#94a3b8"))

	gaugeGreenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22c55e"))

	gaugeYellowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#eab308"))

	gaugeRedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ef4444"))

	gaugeEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))
)

// ansiRegex strips ANSI escape codes for selected-row rendering.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// padRight pads s with spaces to width w (based on visual width).
func padRight(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}

// truncRow truncates an ANSI-colored row string to fit within width.
func truncRow(s string, width int) string {
	return ansi.Truncate(s, width, "")
}

// safeRowWidth leaves a small right margin to avoid terminal autowrap.
func safeRowWidth(width int) int {
	w := width - 2
	if w < 1 {
		return 1
	}
	return w
}

// safeTruncRow truncates to a conservative width to avoid edge wrapping.
func safeTruncRow(s string, width int) string {
	return ansi.Truncate(s, safeRowWidth(width), "")
}

// fitCell ensures an ANSI-colored cell has exactly the requested visible width.
func fitCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	cell := ansi.Truncate(s, width, "")
	return padRight(cell, width)
}

func renderHeaderCell(label string, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	text := ansi.Truncate(strings.TrimSpace(label), width, "")
	return style.Width(width).Align(lipgloss.Center).Render(text)
}

func renderTableHeader(row string, width int) string {
	return HeaderBarStyle.Render(safeTruncRow(row, width))
}

func visibleColumnRange(width int, cursor int, colWidths []int) (start int, end int) {
	if len(colWidths) == 0 {
		return 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(colWidths) {
		cursor = len(colWidths) - 1
	}

	maxWidth := safeRowWidth(width)
	if maxWidth < 1 {
		return 0, 1
	}

	start = cursor
	used := colWidths[cursor]
	if used > maxWidth {
		return cursor, cursor + 1
	}

	for start > 0 {
		add := colWidths[start-1] + 1
		if used+add > maxWidth {
			break
		}
		start--
		used += add
	}

	end = cursor + 1
	for end < len(colWidths) {
		add := colWidths[end] + 1
		if used+add > maxWidth {
			break
		}
		used += add
		end++
	}

	return start, end
}

// selectedRow renders a row with the selected highlight, stripping colors
// and hard-truncating to prevent line wrapping.
func selectedRow(line string, width int) string {
	if width <= 0 {
		return ""
	}
	rowWidth := safeRowWidth(width)

	plain := stripAnsi(line)
	plain = ansi.Truncate(plain, rowWidth, "")
	plain = padRight(plain, rowWidth)
	return SelectedRowStyle.Render(plain)
}

// renderFooter renders a footer bar with styled key binding hints.
func renderFooter(width int, pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(footerKeyStyle.Render(pairs[i]))
		b.WriteString(footerDescStyle.Render(pairs[i+1]))
	}
	// If there's an odd trailing element (like a count), append it.
	if len(pairs)%2 == 1 {
		b.WriteString(footerDescStyle.Render(pairs[len(pairs)-1]))
	}
	return FooterStyle.Width(width).Render(b.String())
}

// ToScid converts a channel ID to short channel ID format.
func ToScid(id uint64) string {
	blocknum := id >> 40
	txnum := (id >> 16) & 0x00FFFFFF
	outnum := id & 0xFFFF
	return fmt.Sprintf("%dx%dx%d", blocknum, txnum, outnum)
}

// FormatAge formats a channel age in blocks to a human-readable string.
func FormatAge(age uint32) string {
	if age < 6 {
		return fmt.Sprintf("%02dm", age*10)
	} else if age < 144 {
		return fmt.Sprintf("%02dh", age/6)
	} else if age < 4383 {
		return fmt.Sprintf("%02dd%02dh", age/144, (age%144)/6)
	} else if age < 52596 {
		return fmt.Sprintf("%02dm%02dd%02dh", age/4383, (age%4383)/144, (age%144)/6)
	}
	return fmt.Sprintf("%02dy%02dm%02dd", age/52596, (age%52596)/4383, (age%4383)/144)
}

func interp(a, b [3]float64, r float64) (result [3]float64) {
	result[0] = a[0] + (b[0]-a[0])*r
	result[1] = a[1] + (b[1]-a[1])*r
	result[2] = a[2] + (b[2]-a[2])*r
	return
}

// ColorizeAge applies a gradient color to a channel age display.
func ColorizeAge(age uint32, text string, opts ...color.Option) string {
	ageColors := [][3]float64{
		{120, 0.9, 0.9},
		{60, 0.9, 0.6},
		{22, 1, 0.5},
	}
	var cur [3]float64
	if age < 26298 {
		cur = interp(ageColors[0], ageColors[1], float64(age)/26298)
	} else if age < 52596 {
		cur = interp(ageColors[1], ageColors[2], float64(age-26298)/26298)
	} else {
		cur = ageColors[2]
	}
	return color.HSL256(cur[0]/360, cur[1], cur[2], opts...)(text)
}
