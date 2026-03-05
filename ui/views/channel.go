package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

var sectionTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#a78bfa")).
	Bold(true)

var detailLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6366f1"))

type Channel struct {
	channels *models.Channels
	Offset   int
}

func (c *Channel) Name() string { return CHANNEL }

func (c *Channel) ScrollDown()  { c.Offset++ }
func (c *Channel) ScrollUp()    { if c.Offset > 0 { c.Offset-- } }
func (c *Channel) ScrollHome()  { c.Offset = 0 }

func (c *Channel) PageDown(n int) { c.Offset += n }
func (c *Channel) PageUp(n int) {
	c.Offset -= n
	if c.Offset < 0 {
		c.Offset = 0
	}
}

func (c *Channel) Render(width, height int) string {
	var b strings.Builder

	// Header.
	b.WriteString(DetailHeaderStyle.Width(width).Render("Channel"))
	b.WriteString("\n")

	// Build content lines.
	lines := c.buildContent()

	// Apply scroll offset and render visible lines.
	dataHeight := height - 2 // header + footer
	if c.Offset > len(lines)-dataHeight {
		c.Offset = len(lines) - dataHeight
	}
	if c.Offset < 0 {
		c.Offset = 0
	}

	end := c.Offset + dataHeight
	if end > len(lines) {
		end = len(lines)
	}

	for i := c.Offset; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	for i := end - c.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	// Footer.
	b.WriteString(renderFooter(width, "F2", "Menu", "Enter", "Channels", "C", "Get disabled", "F10", "Quit"))
	return b.String()
}

func (c *Channel) buildContent() []string {
	channel := c.channels.Current()

	var lines []string
	add := func(format string, a ...interface{}) {
		lines = append(lines, fmt.Sprintf(format, a...))
	}

	add("%s", sectionTitleStyle.Render(" Channel "))
	add("%s %s", detailLabelStyle.Render("             Status:"), status(channel))
	if channel.Status == netmodels.ChannelForceClosing {
		add("%s %d blocks", detailLabelStyle.Render("         Matured in:"), channel.BlocksTilMaturity)
	}
	add("%s %d (%s)", detailLabelStyle.Render("                 ID:"), channel.ID, ToScid(channel.ID))
	add("%s %s", detailLabelStyle.Render("           Capacity:"), formatAmount(channel.Capacity))
	add("%s %s", detailLabelStyle.Render("      Local Balance:"), formatAmount(channel.LocalBalance))
	add("%s %s", detailLabelStyle.Render("     Remote Balance:"), formatAmount(channel.RemoteBalance))
	add("%s %s", detailLabelStyle.Render("      Channel Point:"), channel.ChannelPoint)
	add("")
	add("%s", sectionTitleStyle.Render(" Node "))
	add("%s %s", detailLabelStyle.Render("         PubKey:"), channel.RemotePubKey)

	if channel.Node != nil {
		alias, forced := channel.ShortAlias()
		if forced {
			alias = color.Cyan()(alias)
		}
		add("%s %s", detailLabelStyle.Render("          Alias:"), alias)
		add("%s %s", detailLabelStyle.Render(" Total Capacity:"), formatAmount(channel.Node.TotalCapacity))
		add("%s %d", detailLabelStyle.Render(" Total Channels:"), channel.Node.NumChannels)

		if c.channels.CurrentNode != nil && c.channels.CurrentNode.PubKey == channel.RemotePubKey {
			disabledOut, disabledIn := 0, 0
			for _, ch := range c.channels.CurrentNode.Channels {
				if ch.LocalPolicy != nil && ch.LocalPolicy.Disabled {
					disabledOut++
				}
				if ch.RemotePolicy != nil && ch.RemotePolicy.Disabled {
					disabledIn++
				}
			}
			add("")
			add(" %s %s", detailLabelStyle.Render("Disabled from node:"), formatDisabledCount(disabledOut, channel.Node.NumChannels))
			add(" %s %s", detailLabelStyle.Render("Disabled to node:  "), formatDisabledCount(disabledIn, channel.Node.NumChannels))
		}
	}

	if channel.LocalPolicy != nil {
		lines = append(lines, policyLines(channel.LocalPolicy, true)...)
	}
	if channel.RemotePolicy != nil {
		lines = append(lines, policyLines(channel.RemotePolicy, false)...)
	}

	if len(channel.PendingHTLC) > 0 {
		add("")
		add("%s", sectionTitleStyle.Render(" Pending HTLCs "))
		for _, htlc := range channel.PendingHTLC {
			add("%s %t", detailLabelStyle.Render("   Incoming:"), htlc.Incoming)
			add("%s %s", detailLabelStyle.Render("     Amount:"), formatAmount(htlc.Amount))
			add("%s %d", detailLabelStyle.Render(" Expiration:"), htlc.ExpirationHeight)
			add("")
		}
	}

	return lines
}

func policyLines(policy *netmodels.RoutingPolicy, outgoing bool) []string {
	red := color.Red()
	dl := detailLabelStyle.Render

	direction := "Outgoing"
	if !outgoing {
		direction = "Incoming"
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, sectionTitleStyle.Render(fmt.Sprintf(" %s Policy ", direction)))
	if policy.Disabled {
		lines = append(lines, red("disabled"))
	}
	lines = append(lines, fmt.Sprintf("%s %d",
		dl("                           Time lock delta:"), policy.TimeLockDelta))
	lines = append(lines, fmt.Sprintf("%s %s",
		dl("             Min htlc (msat):"), formatAmount(policy.MinHtlc)))
	lines = append(lines, fmt.Sprintf("%s %s",
		dl("              Max htlc (sat):"), formatAmount(int64(policy.MaxHtlc/1000))))
	lines = append(lines, fmt.Sprintf("%s %s",
		dl("               Fee base msat:"), formatAmount(policy.FeeBaseMsat)))
	lines = append(lines, fmt.Sprintf("%s %d",
		dl("         Fee rate milli msat:"), policy.FeeRateMilliMsat))
	lines = append(lines, fmt.Sprintf("%s %d",
		dl("       Inbound fee base msat:"), policy.InboundFeeBaseMsat))
	lines = append(lines, fmt.Sprintf("%s %d",
		dl(" Inbound fee rate milli msat:"), policy.InboundFeeRateMilliMsat))
	return lines
}

func formatAmount(amt int64) string {
	btc := amt / 1e8
	ms := amt % 1e8 / 1e6
	ts := amt % 1e6 / 1e3
	s := amt % 1e3
	if btc > 0 {
		return fmt.Sprintf("%d.%02d,%03d,%03d", btc, ms, ts, s)
	}
	if ms > 0 {
		return fmt.Sprintf("%d,%03d,%03d", ms, ts, s)
	}
	if ts > 0 {
		return fmt.Sprintf("%d,%03d", ts, s)
	}
	if s >= 0 {
		return fmt.Sprintf("%d", s)
	}
	return fmt.Sprintf("error: %d", amt)
}

func formatDisabledCount(cnt int, total uint32) string {
	perc := uint32(cnt) * 100 / total
	disabledStr := ""
	if perc >= 25 && perc < 50 {
		disabledStr = color.Yellow(color.Bold)(fmt.Sprintf("%4d", cnt))
	} else if perc >= 50 {
		disabledStr = color.Red(color.Bold)(fmt.Sprintf("%4d", cnt))
	} else {
		disabledStr = fmt.Sprintf("%4d", cnt)
	}
	return fmt.Sprintf("%s / %d (%d%%)", disabledStr, total, perc)
}

func NewChannel(channels *models.Channels) *Channel {
	return &Channel{channels: channels}
}
