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

var DefaultPaymentsColumns = []string{
	"TYPE", "TIME", "STATUS", "AMOUNT", "AMOUNT_MSAT", "FEE", "FEE_MSAT", "ATTEMPTS", "FAILURE", "INDEX", "HASH", "PREIMAGE", "REQUEST",
}

type paymentsColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.PaymentsSort
	display func(*netmodels.Payment, ...color.Option) string
}

type Payments struct {
	cfg       *config.View
	columns   []paymentsColumn
	payments  *models.Payments
	Cursor    int
	Offset    int
	ColCursor int
}

func (c *Payments) Name() string { return PAYMENTS }
func (c *Payments) CursorDown() {
	if c.Cursor < c.payments.Len()-1 {
		c.Cursor++
	}
}
func (c *Payments) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}
func (c *Payments) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}
func (c *Payments) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}
func (c *Payments) Home()           { c.Cursor = 0 }
func (c *Payments) End()            { c.Cursor = max(0, c.payments.Len()-1) }
func (c *Payments) PageDown(ps int) { c.Cursor = min(c.Cursor+ps, max(0, c.payments.Len()-1)) }
func (c *Payments) PageUp(ps int)   { c.Cursor = max(0, c.Cursor-ps) }

func (c *Payments) Sort(column string, order models.Order) {
	if c.ColCursor >= len(c.columns) {
		return
	}
	col := c.columns[c.ColCursor]
	if col.sort == nil {
		return
	}
	c.payments.Sort(col.sort(order))
	for i := range c.columns {
		c.columns[i].sorted = (i == c.ColCursor)
	}
}

func (c *Payments) Render(width, height int) string {
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
	items := c.payments.List()
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
		fmt.Sprintf("  Payments: %d", c.payments.Len())))
	return b.String()
}

func NewPayments(cfg *config.View, payments *models.Payments) *Payments {
	if payments == nil {
		payments = &models.Payments{}
	}
	view := &Payments{cfg: cfg, payments: payments}
	printer := message.NewPrinter(language.English)

	cols := DefaultPaymentsColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		cols = cfg.Columns
	}
	view.columns = make([]paymentsColumn, len(cols))
	timeColIndex := -1

	for i := range cols {
		switch cols[i] {
		case "TYPE":
			view.columns[i] = paymentsColumn{width: 7, name: fmt.Sprintf("%-7s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.IntSort(int(a.Kind), int(b.Kind), order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-7s", payment.Kind.String()))
				}}
		case "TIME":
			timeColIndex = i
			view.columns[i] = paymentsColumn{width: 25, name: fmt.Sprintf("%25s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.Int64Sort(a.CreatedAtNs(), b.CreatedAtNs(), order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					ts := payment.CreatedAtNs()
					if ts == 0 {
						return color.White(opts...)(fmt.Sprintf("%25s", "-"))
					}
					return color.White(opts...)(fmt.Sprintf("%25s", time.Unix(0, ts).Format("15:04:05 Jan _2 2006")))
				}}
		case "STATUS":
			view.columns[i] = paymentsColumn{width: 10, name: fmt.Sprintf("%10s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.IntSort(int(a.Status), int(b.Status), order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					label := fmt.Sprintf("%10s", payment.Status.String())
					switch payment.Status {
					case netmodels.PaymentStatusSucceeded:
						return color.Green(opts...)(label)
					case netmodels.PaymentStatusFailed:
						return color.Red(opts...)(label)
					case netmodels.PaymentStatusInFlight, netmodels.PaymentStatusInitiated:
						return color.Yellow(opts...)(label)
					default:
						return color.White(opts...)(label)
					}
				}}
		case "AMOUNT":
			view.columns[i] = paymentsColumn{width: 12, name: fmt.Sprintf("%12s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.Int64Sort(a.ValueSat, b.ValueSat, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%12d", payment.ValueSat))
				}}
		case "AMOUNT_MSAT":
			view.columns[i] = paymentsColumn{width: 14, name: fmt.Sprintf("%14s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.Int64Sort(a.ValueMsat, b.ValueMsat, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%14d", payment.ValueMsat))
				}}
		case "FEE":
			view.columns[i] = paymentsColumn{width: 10, name: fmt.Sprintf("%10s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.Int64Sort(a.FeeSat, b.FeeSat, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%10d", payment.FeeSat))
				}}
		case "FEE_MSAT":
			view.columns[i] = paymentsColumn{width: 12, name: fmt.Sprintf("%12s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.Int64Sort(a.FeeMsat, b.FeeMsat, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%12d", payment.FeeMsat))
				}}
		case "ATTEMPTS":
			view.columns[i] = paymentsColumn{width: 8, name: fmt.Sprintf("%8s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.IntSort(a.Attempts, b.Attempts, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%8d", payment.Attempts))
				}}
		case "FAILURE":
			view.columns[i] = paymentsColumn{width: 24, name: fmt.Sprintf("%-24s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.StringSort(a.FailureReason, b.FailureReason, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					value := payment.FailureReason
					if value == "" {
						value = "-"
					}
					return color.White(opts...)(fmt.Sprintf("%-24s", value))
				}}
		case "INDEX":
			view.columns[i] = paymentsColumn{width: 10, name: fmt.Sprintf("%10s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.UInt64Sort(a.PaymentIndex, b.PaymentIndex, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%10d", payment.PaymentIndex))
				}}
		case "HASH":
			view.columns[i] = paymentsColumn{width: 64, name: fmt.Sprintf("%-64s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.StringSort(a.PaymentHash, b.PaymentHash, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-64s", displayPaymentField(payment.PaymentHash)))
				}}
		case "PREIMAGE":
			view.columns[i] = paymentsColumn{width: 64, name: fmt.Sprintf("%-64s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.StringSort(a.PaymentPreimage, b.PaymentPreimage, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-64s", displayPaymentField(payment.PaymentPreimage)))
				}}
		case "REQUEST":
			view.columns[i] = paymentsColumn{width: 120, name: fmt.Sprintf("%-120s", cols[i]),
				sort: func(order models.Order) models.PaymentsSort {
					return func(a, b *netmodels.Payment) bool {
						return models.StringSort(a.PaymentRequest, b.PaymentRequest, order)
					}
				},
				display: func(payment *netmodels.Payment, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-120s", displayPaymentField(payment.PaymentRequest)))
				}}
		default:
			view.columns[i] = paymentsColumn{width: 10, name: fmt.Sprintf("%-10s", cols[i]),
				display: func(payment *netmodels.Payment, opts ...color.Option) string { return "" }}
		}
	}

	if timeColIndex >= 0 {
		if cmp := view.columns[timeColIndex].sort; cmp != nil {
			payments.Sort(cmp(models.Desc))
		}
	}

	return view
}

func displayPaymentField(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
