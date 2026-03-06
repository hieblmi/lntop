package views

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

// Panel styles with rounded borders.
var (
	channelsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#5b37b7")).
				Padding(0, 1)

	walletPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#2563eb")).
				Padding(0, 1)

	accountingPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#0f766e")).
				Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#a78bfa"))

	panelLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366f1"))

	accountingSpinnerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f59e0b")).
				Bold(true)

	windowInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ecfeff")).
				Background(lipgloss.Color("#134e4a")).
				Bold(true).
				Padding(0, 1)

	windowInputIdleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ccfbf1")).
				Background(lipgloss.Color("#0f766e")).
				Padding(0, 1)

	windowErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fca5a5"))
)

type Summary struct {
	info                    *models.Info
	channelsBalance         *models.ChannelsBalance
	walletBalance           *models.WalletBalance
	channels                *models.Channels
	fwdingHist              *models.FwdingHist
	forwardingHistLoading   bool
	forwardingWindowInput   string
	forwardingWindowEditing bool
	forwardingWindowErr     string
	pulseFrame              int
}

func (s *Summary) Render(width int) string {
	if s.info.Info == nil || s.channelsBalance.ChannelsBalance == nil || s.walletBalance.WalletBalance == nil || s.fwdingHist == nil {
		return ""
	}

	p := message.NewPrinter(language.English)
	green := color.Green()
	yellow := color.Yellow()
	red := color.Red()

	label := panelLabelStyle.Render
	stats := s.fwdingHist.Stats()

	// Left panel: channels.
	var left strings.Builder
	left.WriteString(panelTitleStyle.Render("Channels"))
	left.WriteString("\n")
	left.WriteString("\n")
	left.WriteString(p.Sprintf("%s %s (%s|%s)",
		label("balance:"),
		formatAmount(s.channelsBalance.Balance+s.channelsBalance.PendingOpenBalance),
		green(p.Sprintf("%s", formatAmount(s.channelsBalance.Balance))),
		yellow(p.Sprintf("%s", formatAmount(s.channelsBalance.PendingOpenBalance))),
	))
	left.WriteString("\n")

	disabledLocal, disabledRemote := 0, 0
	for _, ch := range s.channels.List() {
		if ch.LocalPolicy != nil && ch.LocalPolicy.Disabled {
			disabledLocal++
		}
		if ch.RemotePolicy != nil && ch.RemotePolicy.Disabled {
			disabledRemote++
		}
	}
	left.WriteString(fmt.Sprintf("%s %d %s %d %s %d %s",
		label("state  :"),
		s.info.NumActiveChannels, green("on"),
		s.info.NumPendingChannels, yellow("pending"),
		s.info.NumInactiveChannels, red("off"),
	))
	left.WriteString("\n")
	if disabledLocal > 0 || disabledRemote > 0 {
		left.WriteString(fmt.Sprintf("%s %d %s %d %s",
			label("disabled:"),
			disabledLocal, red("local\u21c8"),
			disabledRemote, red("remote\u21ca"),
		))
		left.WriteString("\n")
	}
	left.WriteString(fmt.Sprintf("%s %s",
		label("gauge  :"),
		gaugeTotal(s.channelsBalance.Balance, s.channels.List()),
	))

	// Right panel: wallet.
	var right strings.Builder
	walletFields := []struct {
		label string
		value string
	}{
		{"total_balance", formatSats(p, s.walletBalance.TotalBalance)},
		{"confirmed_balance", formatSats(p, s.walletBalance.ConfirmedBalance)},
		{"unconfirmed_balance", formatSats(p, s.walletBalance.UnconfirmedBalance)},
		{"locked_balance", formatSats(p, s.walletBalance.LockedBalance)},
		{"reserved_balance_anchor_chan", formatSats(p, s.walletBalance.ReservedBalanceAnchorChan)},
	}
	walletLabelWidth := 0
	walletValueWidth := 0
	for _, field := range walletFields {
		if w := lipgloss.Width(field.label + ":"); w > walletLabelWidth {
			walletLabelWidth = w
		}
		if w := lipgloss.Width(field.value); w > walletValueWidth {
			walletValueWidth = w
		}
	}
	right.WriteString(panelTitleStyle.Render("Wallet"))
	right.WriteString("\n")
	right.WriteString("\n")
	for i, field := range walletFields {
		if i > 0 {
			right.WriteString("\n")
		}
		right.WriteString(renderAlignedField(field.label+":", field.value, walletLabelWidth, walletValueWidth))
	}

	// Accounting panel follows the currently filtered forwarding history.
	accountingFields := []struct {
		label string
		value string
	}{
		{"Profit", formatMsatAsSats(p, stats.FeesTotalMsat)},
		{"Total Forwarded", formatSats(p, int64(stats.ForwardedTotal))},
		{"Biggest Forward", formatSats(p, int64(stats.LargestForward))},
		{"Smallest Forward", formatSats(p, int64(stats.SmallestForward))},
		{"Most Profitable Forward", formatMsatAsSats(p, stats.MostProfitableFeeMsat)},
	}
	accountingLabelWidth := 0
	accountingValueWidth := 0
	for _, field := range accountingFields {
		if w := lipgloss.Width(field.label + ":"); w > accountingLabelWidth {
			accountingLabelWidth = w
		}
		if w := lipgloss.Width(field.value); w > accountingValueWidth {
			accountingValueWidth = w
		}
	}

	var accounting strings.Builder
	accounting.WriteString(panelTitleStyle.Render("Accounting (FwdingHistory)"))
	if s.forwardingHistLoading {
		accounting.WriteString(" ")
		accounting.WriteString(accountingSpinnerStyle.Render(summarySpinnerFrame(s.pulseFrame)))
		accounting.WriteString(" ")
		accounting.WriteString(panelLabelStyle.Render("loading"))
	}
	accounting.WriteString("\n")
	accounting.WriteString("\n")
	accounting.WriteString(fmt.Sprintf("%s %s",
		label("Forwarding Events Since:"),
		s.renderWindowInput(),
	))
	accounting.WriteString("\n")
	accounting.WriteString(panelLabelStyle.Render("(-1m, -1h, -1d, -1M, -1y)"))
	for _, field := range accountingFields {
		accounting.WriteString("\n")
		accounting.WriteString(renderAlignedField(field.label+":", field.value, accountingLabelWidth, accountingValueWidth))
	}
	accounting.WriteString("\n")
	accounting.WriteString(fmt.Sprintf("%s %s",
		label("Hottest link:"),
		s.hottestLinkDisplay(stats),
	))
	if s.forwardingWindowErr != "" {
		accounting.WriteString("\n")
		accounting.WriteString(windowErrorStyle.Render(s.forwardingWindowErr))
	}

	return s.layoutPanels(width, left.String(), right.String(), accounting.String())
}

// gaugeTotal renders a gradient-colored balance gauge.
func gaugeTotal(balance int64, channels []*netmodels.Channel) string {
	capacity := int64(0)
	for i := range channels {
		capacity += channels[i].Capacity
	}
	if capacity == 0 {
		return fmt.Sprintf("[%20s]  0%%", "")
	}

	pct := float64(balance) / float64(capacity)
	filled := int(pct * 20)
	var buffer bytes.Buffer

	for i := 0; i < 20; i++ {
		if i < filled {
			// Gradient from green (low) through yellow to red (high local balance).
			ratio := float64(i) / 20.0
			if ratio < 0.5 {
				buffer.WriteString(gaugeGreenStyle.Render("\u2588"))
			} else if ratio < 0.75 {
				buffer.WriteString(gaugeYellowStyle.Render("\u2588"))
			} else {
				buffer.WriteString(gaugeRedStyle.Render("\u2588"))
			}
		} else {
			buffer.WriteString(gaugeEmptyStyle.Render("\u2591"))
		}
	}
	return fmt.Sprintf("%s %2d%%", buffer.String(), balance*100/capacity)
}

func NewSummary(info *models.Info,
	channelsBalance *models.ChannelsBalance,
	walletBalance *models.WalletBalance,
	channels *models.Channels,
	fwdingHist *models.FwdingHist) *Summary {
	return &Summary{
		info:            info,
		channelsBalance: channelsBalance,
		walletBalance:   walletBalance,
		channels:        channels,
		fwdingHist:      fwdingHist,
	}
}

func (s *Summary) SetPulseFrame(frame int) {
	s.pulseFrame = frame
}

func (s *Summary) SetForwardingState(loading bool, editing bool, input string, err string) {
	s.forwardingHistLoading = loading
	s.forwardingWindowEditing = editing
	s.forwardingWindowInput = input
	s.forwardingWindowErr = err
}

func (s *Summary) renderWindowInput() string {
	value := s.fwdingHist.StartTime
	if s.forwardingWindowEditing {
		value = s.forwardingWindowInput
	}
	if value == "" {
		value = "all"
	}

	const width = 8
	text := value
	style := windowInputIdleStyle
	if s.forwardingWindowEditing {
		style = windowInputStyle
		cursor := "|"
		if len(text) >= width-1 {
			text = text[len(text)-(width-1):]
		}
		text = fmt.Sprintf("%-*s%s", width-1, text, cursor)
	} else {
		text = fmt.Sprintf("%-*s", width, text)
	}

	return style.Render(text)
}

func (s *Summary) layoutPanels(width int, channelsPanel string, walletPanel string, accountingPanel string) string {
	const gap = 1

	panels := equalizePanelHeights(
		channelsPanel,
		walletPanel,
		accountingPanel,
	)
	channelsPanel, walletPanel, accountingPanel = panels[0], panels[1], panels[2]

	if width < 72 {
		return strings.Join([]string{
			renderSummaryPanel(channelsPanelStyle, channelsPanel, width),
			renderSummaryPanel(walletPanelStyle, walletPanel, width),
			renderSummaryPanel(accountingPanelStyle, accountingPanel, width),
		}, "\n")
	}

	if width < 110 {
		return strings.Join([]string{
			renderSummaryPanel(channelsPanelStyle, channelsPanel, width),
			renderSummaryPanel(walletPanelStyle, walletPanel, width),
			renderSummaryPanel(accountingPanelStyle, accountingPanel, width),
		}, "\n")
	}

	available := width - gap*2
	channelsWidth := available / 3
	walletWidth := available / 3
	accountingWidth := available - channelsWidth - walletWidth

	return lipgloss.JoinHorizontal(lipgloss.Top,
		renderSummaryPanel(channelsPanelStyle, channelsPanel, channelsWidth),
		" ",
		renderSummaryPanel(walletPanelStyle, walletPanel, walletWidth),
		" ",
		renderSummaryPanel(accountingPanelStyle, accountingPanel, accountingWidth),
	)
}

func renderSummaryPanel(style lipgloss.Style, content string, width int) string {
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	return style.Width(innerWidth).Render(content)
}

func alignRight(value string, width int) string {
	valueWidth := lipgloss.Width(value)
	if width <= valueWidth {
		return value
	}
	return strings.Repeat(" ", width-valueWidth) + value
}

func formatSats(p *message.Printer, amt int64) string {
	return p.Sprintf("%d sats", amt)
}

func formatMsatAsSats(p *message.Printer, msat uint64) string {
	whole := msat / 1000
	frac := msat % 1000
	return p.Sprintf("%d.%03d sats", whole, frac)
}

func renderAlignedField(labelText string, value string, labelWidth int, valueWidth int) string {
	labelCol := panelLabelStyle.Width(labelWidth).Render(labelText)
	valueCol := alignRight(value, valueWidth)
	return labelCol + " " + valueCol
}

func (s *Summary) hottestLinkDisplay(stats models.FwdingHistStats) string {
	inAlias := s.channelAlias(stats.HottestLinkInChanID, stats.HottestLinkInAlias)
	outAlias := s.channelAlias(stats.HottestLinkOutChanID, stats.HottestLinkOutAlias)
	if inAlias == "" && outAlias == "" {
		return "-"
	}
	return inAlias + " -> " + outAlias
}

func (s *Summary) channelAlias(chanID uint64, fallback string) string {
	if chanID != 0 {
		for _, ch := range s.channels.List() {
			if ch.ID == chanID {
				alias, _ := ch.ShortAlias()
				if alias != "" {
					return alias
				}
			}
		}
	}
	if fallback != "" {
		return fallback
	}
	if chanID == 0 {
		return ""
	}
	return fmt.Sprintf("%d", chanID)
}

func equalizePanelHeights(panels ...string) []string {
	maxLines := 0
	for _, panel := range panels {
		lines := strings.Count(panel, "\n") + 1
		if lines > maxLines {
			maxLines = lines
		}
	}

	result := make([]string, len(panels))
	for i, panel := range panels {
		lines := strings.Count(panel, "\n") + 1
		if lines < maxLines {
			panel += strings.Repeat("\n", maxLines-lines)
		}
		result[i] = panel
	}

	return result
}

func summarySpinnerFrame(frame int) string {
	frames := []string{"|", "/", "-", "\\"}
	if frame < 0 {
		frame = 0
	}
	return frames[frame%len(frames)]
}
