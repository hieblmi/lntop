package views

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

var DefaultChannelsColumns = []string{
	"STATUS", "ALIAS", "GAUGE", "LOCAL", "REMOTE", "CAP",
	"SENT", "RECEIVED", "HTLC", "UNSETTLED", "CFEE",
	"LAST UPDATE", "AGE", "PRIVATE", "ID",
}

type channelsColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.ChannelsSort
	display func(*netmodels.Channel, ...color.Option) string
}

type Channels struct {
	cfg            *config.View
	columns        []channelsColumn
	channels       *models.Channels
	Cursor         int
	Offset         int
	ColCursor      int
	pulseFrame     int
	lastPulseFrame int
	prevHTLC       map[string]int
	prevUnsettled  map[string]int64
	prevSent       map[string]int64
	prevReceived   map[string]int64
	htlcBlink      map[string]int
	unsettledBlink map[string]int
	sentFlash      map[string]int
	receivedFlash  map[string]int
}

func (c *Channels) Name() string { return CHANNELS }

func (c *Channels) CursorDown() {
	if c.Cursor < c.channels.Len()-1 {
		c.Cursor++
	}
}

func (c *Channels) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}

func (c *Channels) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}

func (c *Channels) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}

func (c *Channels) Home() { c.Cursor = 0 }

func (c *Channels) End() {
	if n := c.channels.Len(); n > 0 {
		c.Cursor = n - 1
	}
}

func (c *Channels) PageDown(pageSize int) {
	n := c.channels.Len()
	c.Cursor += pageSize
	if c.Cursor >= n {
		c.Cursor = n - 1
	}
	if c.Cursor < 0 {
		c.Cursor = 0
	}
}

func (c *Channels) PageUp(pageSize int) {
	c.Cursor -= pageSize
	if c.Cursor < 0 {
		c.Cursor = 0
	}
}

func (c *Channels) Index() int { return c.Cursor }

func (c *Channels) SetPulseFrame(frame int) {
	if frame != c.lastPulseFrame {
		c.decayBlinks(c.htlcBlink)
		c.decayBlinks(c.unsettledBlink)
		c.decayBlinks(c.sentFlash)
		c.decayBlinks(c.receivedFlash)
		c.lastPulseFrame = frame
	}
	c.pulseFrame = frame
}

func (c *Channels) HasAnimatedAlerts() bool {
	if len(c.htlcBlink) > 0 || len(c.unsettledBlink) > 0 || len(c.sentFlash) > 0 || len(c.receivedFlash) > 0 {
		return true
	}
	for _, ch := range c.channels.List() {
		if len(ch.PendingHTLC) > 0 || ch.UnsettledBalance > 0 {
			return true
		}
	}
	return false
}

func (c *Channels) Sort(column string, order models.Order) {
	index := c.ColCursor
	if index >= len(c.columns) {
		return
	}
	col := c.columns[index]
	if col.sort == nil {
		return
	}
	c.channels.Sort(col.sort(order))
	for i := range c.columns {
		c.columns[i].sorted = (i == index)
	}
}

func (c *Channels) Render(width, height int) string {
	var b strings.Builder

	// Column header.
	var hdr strings.Builder
	for i, col := range c.columns {
		name := renderHeaderCell(col.name, col.width, DefaultColStyle)
		if i == c.ColCursor {
			name = renderHeaderCell(col.name, col.width, ActiveColStyle)
		} else if col.sorted {
			name = renderHeaderCell(col.name, col.width, SortedColStyle)
		}
		hdr.WriteString(name)
		hdr.WriteString(" ")
	}
	b.WriteString(HeaderBarStyle.Width(width).MaxWidth(width).Render(safeTruncRow(hdr.String(), width)))
	b.WriteString("\n")

	// Data rows.
	dataHeight := height - 2 // header + footer
	items := c.channels.List()
	c.syncAlertTransitions(items)

	if c.Cursor >= len(items) {
		c.Cursor = len(items) - 1
	}
	if c.Cursor < 0 {
		c.Cursor = 0
	}
	if c.Cursor < c.Offset {
		c.Offset = c.Cursor
	}
	if c.Cursor >= c.Offset+dataHeight {
		c.Offset = c.Cursor - dataHeight + 1
	}

	end := c.Offset + dataHeight
	if end > len(items) {
		end = len(items)
	}

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

	// Pad empty rows.
	for i := end - c.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	// Footer.
	b.WriteString(renderFooter(width, "F2", "Menu", "Enter", "Channel", "F10", "Quit"))
	return b.String()
}

func NewChannels(cfg *config.View, chans *models.Channels) *Channels {
	channels := &Channels{
		cfg:            cfg,
		channels:       chans,
		prevHTLC:       make(map[string]int),
		prevUnsettled:  make(map[string]int64),
		prevSent:       make(map[string]int64),
		prevReceived:   make(map[string]int64),
		htlcBlink:      make(map[string]int),
		unsettledBlink: make(map[string]int),
		sentFlash:      make(map[string]int),
		receivedFlash:  make(map[string]int),
	}

	printer := message.NewPrinter(language.English)

	columns := DefaultChannelsColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		columns = cfg.Columns
	}

	channels.columns = make([]channelsColumn, len(columns))

	for i := range columns {
		switch columns[i] {
		case "STATUS":
			channels.columns[i] = channelsColumn{
				width: 8,
				name:  fmt.Sprintf("%-8s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.IntSort(-c1.Status, -c2.Status, order)
					}
				},
				display: status,
			}
		case "ALIAS":
			channels.columns[i] = channelsColumn{
				width: 15,
				name:  fmt.Sprintf("%-15s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.StringSort(c1.Node.Alias, c2.Node.Alias, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					aliasColor := color.White(opts...)
					alias, forced := c.ShortAlias()
					if forced {
						aliasColor = color.Cyan(opts...)
					}
					return aliasColor(fmt.Sprintf("%-15s", alias))
				},
			}
		case "GAUGE":
			channels.columns[i] = channelsColumn{
				width: 21,
				name:  fmt.Sprintf("%-21s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Float64Sort(
							float64(c1.LocalBalance)*100/float64(c1.Capacity),
							float64(c2.LocalBalance)*100/float64(c2.Capacity),
							order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					pct := float64(c.LocalBalance) / float64(c.Capacity)
					filled := int(pct * 15)
					var buf strings.Builder
					for i := 0; i < 15; i++ {
						if i < filled {
							ratio := float64(i) / 15.0
							if ratio < 0.5 {
								buf.WriteString(gaugeGreenStyle.Render("\u2588"))
							} else if ratio < 0.75 {
								buf.WriteString(gaugeYellowStyle.Render("\u2588"))
							} else {
								buf.WriteString(gaugeRedStyle.Render("\u2588"))
							}
						} else {
							buf.WriteString(gaugeEmptyStyle.Render("\u2591"))
						}
					}
					return fmt.Sprintf("%s %2d%%", buf.String(), c.LocalBalance*100/c.Capacity)
				},
			}
		case "LOCAL":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.LocalBalance, c2.LocalBalance, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return color.Cyan(opts...)(printer.Sprintf("%12d", c.LocalBalance))
				},
			}
		case "REMOTE":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.RemoteBalance, c2.RemoteBalance, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return color.Cyan(opts...)(printer.Sprintf("%12d", c.RemoteBalance))
				},
			}
		case "CAP":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.Capacity, c2.Capacity, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%12d", c.Capacity))
				},
			}
		case "SENT":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.TotalAmountSent, c2.TotalAmountSent, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					value := printer.Sprintf("%12d", c.TotalAmountSent)
					if channels.sentFlash[c.ChannelPoint] > 0 {
						return channels.renderTrafficFlash(value, false)
					}
					return color.Cyan(opts...)(value)
				},
			}
		case "RECEIVED":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.TotalAmountReceived, c2.TotalAmountReceived, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					value := printer.Sprintf("%12d", c.TotalAmountReceived)
					if channels.receivedFlash[c.ChannelPoint] > 0 {
						return channels.renderTrafficFlash(value, true)
					}
					return color.Cyan(opts...)(value)
				},
			}
		case "HTLC":
			channels.columns[i] = channelsColumn{
				width: 5,
				name:  fmt.Sprintf("%5s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.IntSort(len(c1.PendingHTLC), len(c2.PendingHTLC), order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					count := len(c.PendingHTLC)
					value := fmt.Sprintf("%5d", count)
					if channels.htlcBlink[c.ChannelPoint] > 0 {
						return channels.renderExitBlink(value, true)
					}
					if count == 0 {
						return color.Yellow(opts...)(value)
					}
					return channels.renderAlertValue(value, true)
				},
			}
		case "UNSETTLED":
			channels.columns[i] = channelsColumn{
				width: 10,
				name:  fmt.Sprintf("%-10s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.UnsettledBalance, c2.UnsettledBalance, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					value := printer.Sprintf("%10d", c.UnsettledBalance)
					if channels.unsettledBlink[c.ChannelPoint] > 0 {
						return channels.renderExitBlink(value, false)
					}
					if c.UnsettledBalance == 0 {
						return color.Yellow(opts...)(value)
					}
					return channels.renderAlertValue(value, false)
				},
			}
		case "CFEE":
			channels.columns[i] = channelsColumn{
				width: 6,
				name:  fmt.Sprintf("%-6s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.Int64Sort(c1.CommitFee, c2.CommitFee, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%6d", c.CommitFee))
				},
			}
		case "LAST UPDATE":
			channels.columns[i] = channelsColumn{
				width: 15,
				name:  fmt.Sprintf("%-15s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.DateSort(c1.LastUpdate, c2.LastUpdate, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					if c.LastUpdate != nil {
						return color.Cyan(opts...)(
							fmt.Sprintf("%15s", c.LastUpdate.Format("15:04:05 Jan _2")),
						)
					}
					return fmt.Sprintf("%15s", "")
				},
			}
		case "PRIVATE":
			channels.columns[i] = channelsColumn{
				width: 7,
				name:  fmt.Sprintf("%-7s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.BoolSort(!c1.Private, !c2.Private, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					if c.Private {
						return color.Red(opts...)("private")
					}
					return color.Green(opts...)("public ")
				},
			}
		case "ID":
			channels.columns[i] = channelsColumn{
				width: 19,
				name:  fmt.Sprintf("%-19s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.UInt64Sort(c1.ID, c2.ID, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					if c.ID == 0 {
						return fmt.Sprintf("%-19s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%-19d", c.ID))
				},
			}
		case "SCID":
			channels.columns[i] = channelsColumn{
				width: 14,
				name:  fmt.Sprintf("%-14s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.UInt64Sort(c1.ID, c2.ID, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					if c.ID == 0 {
						return fmt.Sprintf("%-14s", "")
					}
					return color.White(opts...)(fmt.Sprintf("%-14s", ToScid(c.ID)))
				},
			}
		case "NUPD":
			channels.columns[i] = channelsColumn{
				width: 8,
				name:  fmt.Sprintf("%-8s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.UInt64Sort(c1.UpdatesCount, c2.UpdatesCount, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%8d", c.UpdatesCount))
				},
			}
		case "BASE_OUT":
			channels.columns[i] = channelsColumn{
				width: 8,
				name:  fmt.Sprintf("%-8s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f uint64
						if c1.LocalPolicy != nil {
							c1f = uint64(c1.LocalPolicy.FeeBaseMsat)
						}
						if c2.LocalPolicy != nil {
							c2f = uint64(c2.LocalPolicy.FeeBaseMsat)
						}
						return models.UInt64Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int64
					if c.LocalPolicy != nil {
						val = c.LocalPolicy.FeeBaseMsat
					}
					return color.White(opts...)(printer.Sprintf("%8d", val))
				},
			}
		case "RATE_OUT":
			channels.columns[i] = channelsColumn{
				width: 8,
				name:  fmt.Sprintf("%-8s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f uint64
						if c1.LocalPolicy != nil {
							c1f = uint64(c1.LocalPolicy.FeeRateMilliMsat)
						}
						if c2.LocalPolicy != nil {
							c2f = uint64(c2.LocalPolicy.FeeRateMilliMsat)
						}
						return models.UInt64Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int64
					if c.LocalPolicy != nil {
						val = c.LocalPolicy.FeeRateMilliMsat
					}
					return color.White(opts...)(printer.Sprintf("%8d", val))
				},
			}
		case "BASE_IN":
			channels.columns[i] = channelsColumn{
				width: 7,
				name:  fmt.Sprintf("%-7s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f uint64
						if c1.RemotePolicy != nil {
							c1f = uint64(c1.RemotePolicy.FeeBaseMsat)
						}
						if c2.RemotePolicy != nil {
							c2f = uint64(c2.RemotePolicy.FeeBaseMsat)
						}
						return models.UInt64Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int64
					if c.RemotePolicy != nil {
						val = c.RemotePolicy.FeeBaseMsat
					}
					return color.White(opts...)(printer.Sprintf("%7d", val))
				},
			}
		case "RATE_IN":
			channels.columns[i] = channelsColumn{
				width: 7,
				name:  fmt.Sprintf("%-7s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f uint64
						if c1.RemotePolicy != nil {
							c1f = uint64(c1.RemotePolicy.FeeRateMilliMsat)
						}
						if c2.RemotePolicy != nil {
							c2f = uint64(c2.RemotePolicy.FeeRateMilliMsat)
						}
						return models.UInt64Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int64
					if c.RemotePolicy != nil {
						val = c.RemotePolicy.FeeRateMilliMsat
					}
					return color.White(opts...)(printer.Sprintf("%7d", val))
				},
			}
		case "INBOUND_BASE":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f int32
						if c1.LocalPolicy != nil {
							c1f = c1.LocalPolicy.InboundFeeBaseMsat
						}
						if c2.LocalPolicy != nil {
							c2f = c2.LocalPolicy.InboundFeeBaseMsat
						}
						return models.Int32Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int32
					if c.LocalPolicy != nil {
						val = c.LocalPolicy.InboundFeeBaseMsat
					}
					return color.White(opts...)(printer.Sprintf("%12d", val))
				},
			}
		case "INBOUND_RATE":
			channels.columns[i] = channelsColumn{
				width: 12,
				name:  fmt.Sprintf("%12s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						var c1f, c2f int32
						if c1.LocalPolicy != nil {
							c1f = c1.LocalPolicy.InboundFeeRateMilliMsat
						}
						if c2.LocalPolicy != nil {
							c2f = c2.LocalPolicy.InboundFeeRateMilliMsat
						}
						return models.Int32Sort(c1f, c2f, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					var val int32
					if c.LocalPolicy != nil {
						val = c.LocalPolicy.InboundFeeRateMilliMsat
					}
					return color.White(opts...)(printer.Sprintf("%12d", val))
				},
			}
		case "AGE":
			channels.columns[i] = channelsColumn{
				width: 10,
				name:  fmt.Sprintf("%10s", columns[i]),
				sort: func(order models.Order) models.ChannelsSort {
					return func(c1, c2 *netmodels.Channel) bool {
						return models.UInt32Sort(c1.Age, c2.Age, order)
					}
				},
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					if c.ID == 0 {
						return fmt.Sprintf("%10s", "")
					}
					result := printer.Sprintf("%10s", FormatAge(c.Age))
					if cfg.Options.GetOption("AGE", "color") == "color" {
						return ColorizeAge(c.Age, result, opts...)
					}
					return color.White(opts...)(result)
				},
			}
		default:
			channels.columns[i] = channelsColumn{
				width: 21,
				name:  fmt.Sprintf("%-21s", columns[i]),
				display: func(c *netmodels.Channel, opts ...color.Option) string {
					return "column does not exist"
				},
			}
		}
	}

	return channels
}

func (c *Channels) renderAlertValue(value string, htlc bool) string {
	baseBg := lipgloss.Color("#22c55e")
	baseFg := lipgloss.Color("#111827")
	hotBg := lipgloss.Color("#ffff00")
	hotFg := lipgloss.Color("#111827")

	width := utf8.RuneCountInString(value)
	if width == 0 {
		return value
	}

	hot := c.pulseFrame % width
	trail := (hot - 1 + width) % width

	var b strings.Builder
	for pos, r := range value {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(baseFg).
			Background(baseBg)

		switch pos {
		case hot:
			style = style.Foreground(hotFg).Background(hotBg)
		case trail:
			style = style.Background(hotBg)
		}

		b.WriteString(style.Render(string(r)))
	}

	return b.String()
}

func (c *Channels) renderExitBlink(value string, htlc bool) string {
	flashFg := lipgloss.Color("#111827")
	flashBg := lipgloss.Color("#ffff00")
	idleFg := lipgloss.Color("#111827")
	idleBg := lipgloss.Color("#22c55e")

	style := lipgloss.NewStyle().Bold(true).Foreground(idleFg).Background(idleBg)
	if c.pulseFrame%2 == 0 {
		style = lipgloss.NewStyle().Bold(true).Foreground(flashFg).Background(flashBg)
	}
	return style.Render(value)
}

func (c *Channels) renderTrafficFlash(value string, received bool) string {
	flashFg := lipgloss.Color("#f8fafc")
	flashBg := lipgloss.Color("#dc2626")
	if received {
		flashBg = lipgloss.Color("#16a34a")
	}

	style := lipgloss.NewStyle().Bold(true).Foreground(flashFg)
	if c.pulseFrame%2 == 0 {
		style = style.Background(flashBg)
	}
	return style.Render(value)
}

func (c *Channels) syncAlertTransitions(items []*netmodels.Channel) {
	active := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := item.ChannelPoint
		active[key] = struct{}{}

		curHTLC := len(item.PendingHTLC)
		if prev, ok := c.prevHTLC[key]; ok && prev > 0 && curHTLC == 0 {
			c.htlcBlink[key] = 4
		}
		c.prevHTLC[key] = curHTLC

		curUnsettled := item.UnsettledBalance
		if prev, ok := c.prevUnsettled[key]; ok && prev > 0 && curUnsettled == 0 {
			c.unsettledBlink[key] = 4
		}
		c.prevUnsettled[key] = curUnsettled

		curSent := item.TotalAmountSent
		if prev, ok := c.prevSent[key]; ok && curSent > prev {
			c.sentFlash[key] = 2
		}
		c.prevSent[key] = curSent

		curReceived := item.TotalAmountReceived
		if prev, ok := c.prevReceived[key]; ok && curReceived > prev {
			c.receivedFlash[key] = 2
		}
		c.prevReceived[key] = curReceived
	}

	for key := range c.prevHTLC {
		if _, ok := active[key]; !ok {
			delete(c.prevHTLC, key)
			delete(c.prevUnsettled, key)
			delete(c.prevSent, key)
			delete(c.prevReceived, key)
			delete(c.htlcBlink, key)
			delete(c.unsettledBlink, key)
			delete(c.sentFlash, key)
			delete(c.receivedFlash, key)
		}
	}
}

func (c *Channels) decayBlinks(blinks map[string]int) {
	for key, count := range blinks {
		if count <= 1 {
			delete(blinks, key)
			continue
		}
		blinks[key] = count - 1
	}
}

func channelDisabled(c *netmodels.Channel, opts ...color.Option) string {
	outgoing := false
	incoming := false
	if c.LocalPolicy != nil && c.LocalPolicy.Disabled {
		outgoing = true
	}
	if c.RemotePolicy != nil && c.RemotePolicy.Disabled {
		incoming = true
	}
	result := ""
	if incoming && outgoing {
		result = "⇅"
	} else if incoming {
		result = "⇊"
	} else if outgoing {
		result = "⇈"
	}
	if result == "" {
		return result
	}
	return color.Red(opts...)(result)
}

func status(c *netmodels.Channel, opts ...color.Option) string {
	disabled := channelDisabled(c, opts...)
	switch c.Status {
	case netmodels.ChannelActive:
		label := "on "
		if disabled != "" {
			return color.Green(opts...)(fmt.Sprintf("%-5s", label)) + disabled
		}
		return color.Green(opts...)(fmt.Sprintf("%-8s", label))
	case netmodels.ChannelInactive:
		label := "off"
		if disabled != "" {
			return color.Red(opts...)(fmt.Sprintf("%-5s", label)) + disabled
		}
		return color.Red(opts...)(fmt.Sprintf("%-8s", label))
	case netmodels.ChannelOpening:
		return color.Yellow(opts...)(fmt.Sprintf("%-8s", "opening"))
	case netmodels.ChannelClosing:
		return color.Yellow(opts...)(fmt.Sprintf("%-8s", "closing"))
	case netmodels.ChannelForceClosing:
		return color.Yellow(opts...)(fmt.Sprintf("%-8s", "f-close"))
	case netmodels.ChannelWaitingClose:
		return color.Yellow(opts...)(fmt.Sprintf("%-8s", "w-close"))
	case netmodels.ChannelClosed:
		return color.Red(opts...)(fmt.Sprintf("%-8s", "closed"))
	}
	return ""
}
