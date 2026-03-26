package views

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

var DefaultReceivedColumns = []string{
	"TYPE", "TIME", "AMOUNT", "MEMO", "R_HASH",
}

type receivedColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.ReceivedSort
	display func(*netmodels.Invoice, ...color.Option) string
}

type Received struct {
	cfg       *config.View
	columns   []receivedColumn
	received  *models.Received
	Cursor    int
	Offset    int
	ColCursor int
}

func (c *Received) Name() string { return RECEIVED }
func (c *Received) CursorDown() {
	if c.Cursor < c.received.Len()-1 {
		c.Cursor++
	}
}
func (c *Received) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}
func (c *Received) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}
func (c *Received) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}
func (c *Received) Home()           { c.Cursor = 0 }
func (c *Received) End()            { c.Cursor = max(0, c.received.Len()-1) }
func (c *Received) PageDown(ps int) { c.Cursor = min(c.Cursor+ps, max(0, c.received.Len()-1)) }
func (c *Received) PageUp(ps int)   { c.Cursor = max(0, c.Cursor-ps) }

func (c *Received) Sort(column string, order models.Order) {
	if c.ColCursor >= len(c.columns) {
		return
	}
	col := c.columns[c.ColCursor]
	if col.sort == nil {
		return
	}
	c.received.Sort(col.sort(order))
	for i := range c.columns {
		c.columns[i].sorted = (i == c.ColCursor)
	}
}

func (c *Received) Render(width, height int) string {
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
	items := c.received.List()
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
		"F2", "Menu", "F9", "Settings", "F10", "Quit",
		fmt.Sprintf("  Invoices: %d", c.received.Len())))
	return b.String()
}

func NewReceived(cfg *config.View, rec *models.Received) *Received {
	received := &Received{cfg: cfg, received: rec}
	printer := message.NewPrinter(language.English)

	cols := DefaultReceivedColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		cols = cfg.Columns
	}
	received.columns = make([]receivedColumn, len(cols))
	var timeColIndex = -1

	for i := range cols {
		switch cols[i] {
		case "TYPE":
			received.columns[i] = receivedColumn{width: 7, name: fmt.Sprintf("%-7s", cols[i]),
				sort: func(order models.Order) models.ReceivedSort {
					return func(a, b *netmodels.Invoice) bool { return models.IntSort(int(a.Kind), int(b.Kind), order) }
				},
				display: func(inv *netmodels.Invoice, opts ...color.Option) string {
					label := "invoice"
					if inv.Kind == netmodels.KindKeysend || inv.PaymentRequest == "" {
						label = "keysend"
					}
					return color.White(opts...)(fmt.Sprintf("%-7s", label))
				}}
		case "TIME":
			timeColIndex = i
			received.columns[i] = receivedColumn{width: 25, name: fmt.Sprintf("%25s", cols[i]),
				sort: func(order models.Order) models.ReceivedSort {
					return func(a, b *netmodels.Invoice) bool {
						at := a.SettleDate
						if at == 0 {
							at = a.CreationDate
						}
						bt := b.SettleDate
						if bt == 0 {
							bt = b.CreationDate
						}
						return models.Int64Sort(at, bt, order)
					}
				},
				display: func(inv *netmodels.Invoice, opts ...color.Option) string {
					ts := inv.SettleDate
					if ts == 0 {
						ts = inv.CreationDate
					}
					return color.White(opts...)(fmt.Sprintf("%25s", time.Unix(ts, 0).Format("15:04:05 Jan _2 2006")))
				}}
		case "AMOUNT":
			received.columns[i] = receivedColumn{width: 12, name: fmt.Sprintf("%12s", cols[i]),
				sort: func(order models.Order) models.ReceivedSort {
					return func(a, b *netmodels.Invoice) bool {
						av := a.AmountPaid
						if av == 0 {
							av = a.Amount
						}
						bv := b.AmountPaid
						if bv == 0 {
							bv = b.Amount
						}
						return models.Int64Sort(av, bv, order)
					}
				},
				display: func(inv *netmodels.Invoice, opts ...color.Option) string {
					amt := inv.AmountPaid
					if amt == 0 {
						amt = inv.Amount
					}
					return color.White(opts...)(printer.Sprintf("%12d", amt))
				}}
		case "MEMO":
			received.columns[i] = receivedColumn{width: 40, name: fmt.Sprintf("%-40s", cols[i]),
				sort: func(order models.Order) models.ReceivedSort {
					return func(a, b *netmodels.Invoice) bool { return models.StringSort(a.Description, b.Description, order) }
				},
				display: func(inv *netmodels.Invoice, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-40s", inv.Description))
				}}
		case "R_HASH":
			received.columns[i] = receivedColumn{width: 64, name: fmt.Sprintf("%-64s", cols[i]),
				sort: func(order models.Order) models.ReceivedSort {
					return func(a, b *netmodels.Invoice) bool { return models.StringSort(a.GetRHash(), b.GetRHash(), order) }
				},
				display: func(inv *netmodels.Invoice, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-64s", inv.GetRHash()))
				}}
		default:
			received.columns[i] = receivedColumn{width: 10, name: fmt.Sprintf("%-10s", cols[i]),
				display: func(inv *netmodels.Invoice, opts ...color.Option) string { return "" }}
		}
	}

	// Default sort by TIME descending.
	if timeColIndex >= 0 {
		if cmp := received.columns[timeColIndex].sort; cmp != nil {
			rec.Sort(cmp(models.Desc))
		}
	}
	return received
}
