package views

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

var DefaultRoutingColumns = []string{
	"DIR", "STATUS", "IN_CHANNEL", "IN_ALIAS",
	"OUT_CHANNEL", "OUT_ALIAS", "AMOUNT", "FEE",
	"LAST UPDATE", "DETAIL",
}

type routingColumn struct {
	name    string
	width   int
	display func(*netmodels.RoutingEvent, ...color.Option) string
}

type Routing struct {
	cfg           *config.View
	columns       []routingColumn
	routingEvents *models.RoutingLog
	Cursor        int
	Offset        int
	ColCursor     int
}

func (c *Routing) Name() string { return ROUTING }
func (c *Routing) CursorDown() {
	if c.Cursor < c.maxIndex() {
		c.Cursor++
	}
}
func (c *Routing) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}
func (c *Routing) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}
func (c *Routing) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}
func (c *Routing) Home()           { c.Cursor = 0 }
func (c *Routing) End()            { c.Cursor = c.maxIndex() }
func (c *Routing) PageDown(ps int) { c.Cursor = min(c.Cursor+ps, c.maxIndex()) }
func (c *Routing) PageUp(ps int)   { c.Cursor = max(0, c.Cursor-ps) }

func (c *Routing) maxIndex() int {
	n := len(c.routingEvents.Log)
	if n == 0 {
		return 0
	}
	return n - 1
}

func (c *Routing) Render(width, height int) string {
	var b strings.Builder

	// Column header.
	var hdr strings.Builder
	for i, col := range c.columns {
		name := renderHeaderCell(col.name, col.width, DefaultColStyle)
		if i == c.ColCursor {
			name = renderHeaderCell(col.name, col.width, ActiveColStyle)
		}
		hdr.WriteString(name)
		hdr.WriteString(" ")
	}
	b.WriteString(HeaderBarStyle.Width(width).MaxWidth(width).Render(safeTruncRow(hdr.String(), width)))
	b.WriteString("\n")

	dataHeight := height - 2
	items := c.routingEvents.Log

	if c.Cursor >= len(items) {
		c.Cursor = max(0, len(items)-1)
	}
	if c.Cursor < c.Offset {
		c.Offset = c.Cursor
	}
	if c.Cursor >= c.Offset+dataHeight {
		c.Offset = c.Cursor - dataHeight + 1
	}
	end := min(c.Offset+dataHeight, len(items))

	for idx := c.Offset; idx < end; idx++ {
		item := items[idx]
		var row strings.Builder
		for i, col := range c.columns {
			var opt color.Option
			if i == c.ColCursor {
				opt = color.Bold
			}
			row.WriteString(fitCell(col.display(item, opt), col.width))
			row.WriteString(" ")
		}
		line := row.String()
		if idx == c.Cursor {
			line = selectedRow(line, width)
		} else {
			line = safeTruncRow(line, width)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	for i := end - c.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(width, "F2", "Menu", "F10", "Quit"))
	return b.String()
}

func NewRouting(cfg *config.View, routingEvents *models.RoutingLog, channels *models.Channels) *Routing {
	routing := &Routing{cfg: cfg, routingEvents: routingEvents}
	printer := message.NewPrinter(language.English)

	columns := DefaultRoutingColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		columns = cfg.Columns
	}
	routing.columns = make([]routingColumn, len(columns))

	for i := range columns {
		switch columns[i] {
		case "DIR":
			routing.columns[i] = routingColumn{width: 4, name: fmt.Sprintf("%-4s", columns[i]), display: rdirection}
		case "STATUS":
			routing.columns[i] = routingColumn{width: 8, name: fmt.Sprintf("%-8s", columns[i]), display: rstatus}
		case "IN_ALIAS":
			routing.columns[i] = routingColumn{width: 25, name: fmt.Sprintf("%-25s", columns[i]), display: ralias(channels, false)}
		case "IN_CHANNEL":
			routing.columns[i] = routingColumn{width: 19, name: fmt.Sprintf("%19s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.IncomingChannelId == 0 {
						return fmt.Sprintf("%19s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%19d", c.IncomingChannelId))
				}}
		case "IN_SCID":
			routing.columns[i] = routingColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.IncomingChannelId == 0 {
						return fmt.Sprintf("%14s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%14s", ToScid(c.IncomingChannelId)))
				}}
		case "IN_TIMELOCK":
			routing.columns[i] = routingColumn{width: 10, name: fmt.Sprintf("%10s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.IncomingTimelock == 0 {
						return fmt.Sprintf("%10s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%10d", c.IncomingTimelock))
				}}
		case "IN_HTLC":
			routing.columns[i] = routingColumn{width: 10, name: fmt.Sprintf("%10s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.IncomingHtlcId == 0 {
						return fmt.Sprintf("%10s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%10d", c.IncomingHtlcId))
				}}
		case "OUT_ALIAS":
			routing.columns[i] = routingColumn{width: 25, name: fmt.Sprintf("%-25s", columns[i]), display: ralias(channels, true)}
		case "OUT_CHANNEL":
			routing.columns[i] = routingColumn{width: 19, name: fmt.Sprintf("%19s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.OutgoingChannelId == 0 {
						return fmt.Sprintf("%19s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%19d", c.OutgoingChannelId))
				}}
		case "OUT_SCID":
			routing.columns[i] = routingColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.OutgoingChannelId == 0 {
						return fmt.Sprintf("%14s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%14s", ToScid(c.OutgoingChannelId)))
				}}
		case "OUT_TIMELOCK":
			routing.columns[i] = routingColumn{width: 10, name: fmt.Sprintf("%10s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.OutgoingTimelock == 0 {
						return fmt.Sprintf("%10s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%10d", c.OutgoingTimelock))
				}}
		case "OUT_HTLC":
			routing.columns[i] = routingColumn{width: 10, name: fmt.Sprintf("%10s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					if c.OutgoingHtlcId == 0 {
						return fmt.Sprintf("%10s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%10d", c.OutgoingHtlcId))
				}}
		case "AMOUNT":
			routing.columns[i] = routingColumn{width: 12, name: fmt.Sprintf("%12s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					return color.Yellow(opts...)(printer.Sprintf("%12d", c.AmountMsat/1000))
				}}
		case "FEE":
			routing.columns[i] = routingColumn{width: 8, name: fmt.Sprintf("%8s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					return color.Yellow(opts...)(printer.Sprintf("%8d", c.FeeMsat/1000))
				}}
		case "LAST UPDATE":
			routing.columns[i] = routingColumn{width: 15, name: fmt.Sprintf("%-15s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					return color.Cyan(opts...)(fmt.Sprintf("%15s", c.LastUpdate.Format("15:04:05 Jan _2")))
				}}
		case "INBOUND_BASE_IN":
			routing.columns[i] = routingColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				display: rinboundFee(channels, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeBaseMsat })}
		case "INBOUND_RATE_IN":
			routing.columns[i] = routingColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				display: rinboundFee(channels, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeRateMilliMsat })}
		case "DETAIL":
			routing.columns[i] = routingColumn{width: 80, name: fmt.Sprintf("%-80s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string {
					return color.Cyan(opts...)(fmt.Sprintf("%-80s", c.FailureDetail))
				}}
		default:
			routing.columns[i] = routingColumn{width: 10, name: fmt.Sprintf("%-10s", columns[i]),
				display: func(c *netmodels.RoutingEvent, opts ...color.Option) string { return fmt.Sprintf("%-10s", "") }}
		}
	}
	return routing
}

func rstatus(c *netmodels.RoutingEvent, opts ...color.Option) string {
	switch c.Status {
	case netmodels.RoutingStatusActive:
		return color.Yellow(opts...)(fmt.Sprintf("%-8s", "active"))
	case netmodels.RoutingStatusSettled:
		return color.Green(opts...)(fmt.Sprintf("%-8s", "settled"))
	case netmodels.RoutingStatusFailed:
		return color.Red(opts...)(fmt.Sprintf("%-8s", "failed"))
	case netmodels.RoutingStatusLinkFailed:
		return color.Red(opts...)(fmt.Sprintf("%-8s", "linkfail"))
	}
	return ""
}

func rdirection(c *netmodels.RoutingEvent, opts ...color.Option) string {
	switch c.Direction {
	case netmodels.RoutingSend:
		return color.White(opts...)(fmt.Sprintf("%-4s", "send"))
	case netmodels.RoutingReceive:
		return color.White(opts...)(fmt.Sprintf("%-4s", "recv"))
	case netmodels.RoutingForward:
		return color.White(opts...)(fmt.Sprintf("%-4s", "forw"))
	}
	return "   "
}

func rinboundFee(channels *models.Channels, extract func(*netmodels.RoutingPolicy) int32) func(*netmodels.RoutingEvent, ...color.Option) string {
	return func(c *netmodels.RoutingEvent, opts ...color.Option) string {
		if c.IncomingChannelId == 0 {
			return fmt.Sprintf("%14s", "")
		}
		for _, ch := range channels.List() {
			if ch.ID == c.IncomingChannelId && ch.LocalPolicy != nil {
				return color.White(opts...)(fmt.Sprintf("%14d", extract(ch.LocalPolicy)))
			}
		}
		return fmt.Sprintf("%14d", 0)
	}
}

func ralias(channels *models.Channels, out bool) func(*netmodels.RoutingEvent, ...color.Option) string {
	return func(c *netmodels.RoutingEvent, opts ...color.Option) string {
		id := c.IncomingChannelId
		if out {
			id = c.OutgoingChannelId
		}
		if id == 0 {
			return color.White(opts...)(fmt.Sprintf("%-25s", ""))
		}
		var alias string
		var forced bool
		aliasColor := color.White(opts...)
		for _, ch := range channels.List() {
			if ch.ID == id {
				alias, forced = ch.ShortAlias()
				if forced {
					aliasColor = color.Cyan(opts...)
				}
				break
			}
		}
		return aliasColor(fmt.Sprintf("%-25s", alias))
	}
}
