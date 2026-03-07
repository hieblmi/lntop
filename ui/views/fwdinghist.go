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

var DefaultFwdinghistColumns = []string{
	"ALIAS_IN", "ALIAS_OUT", "AMT_IN", "AMT_OUT",
	"FEE", "TIMESTAMP_NS", "CHAN_ID_IN", "CHAN_ID_OUT",
}

type fwdinghistColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.FwdinghistSort
	display func(*netmodels.ForwardingEvent, ...color.Option) string
}

type FwdingHist struct {
	cfg        *config.View
	columns    []fwdinghistColumn
	fwdinghist *models.FwdingHist
	Cursor     int
	Offset     int
	ColCursor  int
}

func (c *FwdingHist) Name() string { return FWDINGHIST }
func (c *FwdingHist) CursorDown() {
	if c.Cursor < c.fwdinghist.Len()-1 {
		c.Cursor++
	}
}
func (c *FwdingHist) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}
func (c *FwdingHist) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}
func (c *FwdingHist) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}
func (c *FwdingHist) Home()           { c.Cursor = 0 }
func (c *FwdingHist) End()            { c.Cursor = max(0, c.fwdinghist.Len()-1) }
func (c *FwdingHist) PageDown(ps int) { c.Cursor = min(c.Cursor+ps, max(0, c.fwdinghist.Len()-1)) }
func (c *FwdingHist) PageUp(ps int)   { c.Cursor = max(0, c.Cursor-ps) }

func (c *FwdingHist) Sort(column string, order models.Order) {
	if c.ColCursor >= len(c.columns) {
		return
	}
	col := c.columns[c.ColCursor]
	if col.sort == nil {
		return
	}
	c.fwdinghist.Sort(col.sort(order))
	for i := range c.columns {
		c.columns[i].sorted = (i == c.ColCursor)
	}
}

func (c *FwdingHist) Render(width, height int) string {
	var b strings.Builder
	colWidths := make([]int, len(c.columns))
	for i := range c.columns {
		colWidths[i] = c.columns[i].width
	}
	visibleStart, visibleEnd := visibleColumnRange(width, c.ColCursor, colWidths)

	var hdr strings.Builder
	for i := visibleStart; i < visibleEnd; i++ {
		col := c.columns[i]
		name := renderHeaderCell(col.name, col.width, DefaultColStyle)
		if i == c.ColCursor {
			name = renderHeaderCell(col.name, col.width, ActiveColStyle)
		} else if col.sorted {
			name = renderHeaderCell(col.name, col.width, SortedColStyle)
		}
		hdr.WriteString(name)
		hdr.WriteString(" ")
	}
	b.WriteString(renderTableHeader(hdr.String(), width))
	b.WriteString("\n")

	dataHeight := height - 2
	items := c.fwdinghist.List()
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
		for i := visibleStart; i < visibleEnd; i++ {
			col := c.columns[i]
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

	b.WriteString(renderFooter(width,
		"F2", "Menu", "W", "Settings", "F9", "Settings", "F10", "Quit",
		fmt.Sprintf("  Total: %d", c.fwdinghist.Len())))
	return b.String()
}

func NewFwdingHist(cfg *config.View, hist *models.FwdingHist, channels *models.Channels) *FwdingHist {
	fh := &FwdingHist{cfg: cfg, fwdinghist: hist}
	printer := message.NewPrinter(language.English)

	columns := DefaultFwdinghistColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		columns = cfg.Columns
	}
	fh.columns = make([]fwdinghistColumn, len(columns))

	for i := range columns {
		switch columns[i] {
		case "ALIAS_IN":
			fh.columns[i] = fwdinghistColumn{width: 30, name: fmt.Sprintf("%30s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.StringSort(e1.PeerAliasIn, e2.PeerAliasIn, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%30s", e.PeerAliasIn))
				}}
		case "ALIAS_OUT":
			fh.columns[i] = fwdinghistColumn{width: 30, name: fmt.Sprintf("%30s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.StringSort(e1.PeerAliasOut, e2.PeerAliasOut, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%30s", e.PeerAliasOut))
				}}
		case "CHAN_ID_IN":
			fh.columns[i] = fwdinghistColumn{width: 19, name: fmt.Sprintf("%19s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.UInt64Sort(e1.ChanIdIn, e2.ChanIdIn, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%19d", e.ChanIdIn))
				}}
		case "CHAN_ID_OUT":
			fh.columns[i] = fwdinghistColumn{width: 19, name: fmt.Sprintf("%19s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.UInt64Sort(e1.ChanIdOut, e2.ChanIdOut, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%19d", e.ChanIdOut))
				}}
		case "AMT_IN":
			fh.columns[i] = fwdinghistColumn{width: 12, name: fmt.Sprintf("%12s", "RECEIVED"),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.UInt64Sort(e1.AmtIn, e2.AmtIn, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%12d", e.AmtIn))
				}}
		case "AMT_OUT":
			fh.columns[i] = fwdinghistColumn{width: 12, name: fmt.Sprintf("%12s", "SENT"),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.UInt64Sort(e1.AmtOut, e2.AmtOut, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%12d", e.AmtOut))
				}}
		case "FEE":
			fh.columns[i] = fwdinghistColumn{name: fmt.Sprintf("%9s", "EARNED"), width: 9,
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.UInt64Sort(e1.Fee, e2.Fee, order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string { return fee(e.Fee) }}
		case "INBOUND_BASE_IN":
			fh.columns[i] = fwdinghistColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.Int32Sort(
							fwdInboundFee(channels, e1.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeBaseMsat }),
							fwdInboundFee(channels, e2.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeBaseMsat }),
							order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					val := fwdInboundFee(channels, e.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeBaseMsat })
					return color.White(opts...)(fmt.Sprintf("%14d", val))
				}}
		case "INBOUND_RATE_IN":
			fh.columns[i] = fwdinghistColumn{width: 14, name: fmt.Sprintf("%14s", columns[i]),
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.Int32Sort(
							fwdInboundFee(channels, e1.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeRateMilliMsat }),
							fwdInboundFee(channels, e2.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeRateMilliMsat }),
							order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					val := fwdInboundFee(channels, e.ChanIdIn, func(p *netmodels.RoutingPolicy) int32 { return p.InboundFeeRateMilliMsat })
					return color.White(opts...)(fmt.Sprintf("%14d", val))
				}}
		case "TIMESTAMP_NS":
			fh.columns[i] = fwdinghistColumn{name: fmt.Sprintf("%15s", "TIME"), width: 20,
				sort: func(order models.Order) models.FwdinghistSort {
					return func(e1, e2 *netmodels.ForwardingEvent) bool {
						return models.Int64Sort(e1.EventTime.UnixNano(), e2.EventTime.UnixNano(), order)
					}
				},
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%20s", e.EventTime.Format("15:04:05 Jan _2")))
				}}
		default:
			fh.columns[i] = fwdinghistColumn{name: fmt.Sprintf("%-21s", columns[i]), width: 21,
				display: func(e *netmodels.ForwardingEvent, opts ...color.Option) string { return "column does not exist" }}
		}
	}
	return fh
}

func fwdInboundFee(channels *models.Channels, chanID uint64, extract func(*netmodels.RoutingPolicy) int32) int32 {
	for _, ch := range channels.List() {
		if ch.ID == chanID && ch.LocalPolicy != nil {
			return extract(ch.LocalPolicy)
		}
	}
	return 0
}

func fee(f uint64, opts ...color.Option) string {
	if f < 100 {
		return color.Cyan(opts...)(fmt.Sprintf("%9d", f))
	}
	if f < 999 {
		return color.Green(opts...)(fmt.Sprintf("%9d", f))
	}
	return color.Yellow(opts...)(fmt.Sprintf("%9d", f))
}
