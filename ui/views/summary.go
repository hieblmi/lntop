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

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#a78bfa"))

	panelLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366f1"))
)

type Summary struct {
	info            *models.Info
	channelsBalance *models.ChannelsBalance
	walletBalance   *models.WalletBalance
	channels        *models.Channels
}

func (s *Summary) Render(width int) string {
	if s.info.Info == nil || s.channelsBalance.ChannelsBalance == nil || s.walletBalance.WalletBalance == nil {
		return ""
	}

	p := message.NewPrinter(language.English)
	green := color.Green()
	yellow := color.Yellow()
	red := color.Red()

	label := panelLabelStyle.Render

	// Border takes 2 chars each side + 1 padding each side = 6 chars per panel.
	borderOverhead := 6
	half := width / 2
	innerWidth := half - borderOverhead
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Left panel: channels.
	var left strings.Builder
	left.WriteString(panelTitleStyle.Render("Channels"))
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
	right.WriteString(panelTitleStyle.Render("Wallet"))
	right.WriteString("\n")
	right.WriteString(p.Sprintf("%s %s (%s|%s)",
		label("balance:"),
		formatAmount(s.walletBalance.TotalBalance),
		green(p.Sprintf("%s", formatAmount(s.walletBalance.ConfirmedBalance))),
		yellow(p.Sprintf("%s", formatAmount(s.walletBalance.UnconfirmedBalance))),
	))

	leftStr := channelsPanelStyle.Width(innerWidth).Render(left.String())
	rightStr := walletPanelStyle.Width(innerWidth).Render(right.String())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftStr, rightStr)
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
			var c lipgloss.Color
			if ratio < 0.5 {
				c = lipgloss.Color("#22c55e") // green
			} else if ratio < 0.75 {
				c = lipgloss.Color("#eab308") // yellow
			} else {
				c = lipgloss.Color("#ef4444") // red
			}
			buffer.WriteString(lipgloss.NewStyle().Foreground(c).Render("\u2588"))
		} else {
			buffer.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Render("\u2591"))
		}
	}
	return fmt.Sprintf("%s %2d%%", buffer.String(), balance*100/capacity)
}

func NewSummary(info *models.Info,
	channelsBalance *models.ChannelsBalance,
	walletBalance *models.WalletBalance,
	channels *models.Channels) *Summary {
	return &Summary{
		info:            info,
		channelsBalance: channelsBalance,
		walletBalance:   walletBalance,
		channels:        channels,
	}
}
