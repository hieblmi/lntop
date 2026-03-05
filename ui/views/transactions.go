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

var DefaultTransactionsColumns = []string{
	"DATE", "HEIGHT", "CONFIR", "AMOUNT", "FEE", "ADDRESSES",
}

type transactionsColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.TransactionsSort
	display func(*netmodels.Transaction, ...color.Option) string
}

type Transactions struct {
	cfg          *config.View
	columns      []transactionsColumn
	transactions *models.Transactions
	Cursor       int
	Offset       int
	ColCursor    int
}

func (c *Transactions) Name() string { return TRANSACTIONS }

func (c *Transactions) CursorDown() {
	if c.Cursor < c.transactions.Len()-1 {
		c.Cursor++
	}
}
func (c *Transactions) CursorUp() {
	if c.Cursor > 0 {
		c.Cursor--
	}
}
func (c *Transactions) ColumnRight() {
	if c.ColCursor < len(c.columns)-1 {
		c.ColCursor++
	}
}
func (c *Transactions) ColumnLeft() {
	if c.ColCursor > 0 {
		c.ColCursor--
	}
}
func (c *Transactions) Home()           { c.Cursor = 0 }
func (c *Transactions) End()            { c.Cursor = max(0, c.transactions.Len()-1) }
func (c *Transactions) PageDown(ps int) { c.Cursor = min(c.Cursor+ps, max(0, c.transactions.Len()-1)) }
func (c *Transactions) PageUp(ps int)   { c.Cursor = max(0, c.Cursor-ps) }
func (c *Transactions) Index() int      { return c.Cursor }

func (c *Transactions) Sort(column string, order models.Order) {
	if c.ColCursor >= len(c.columns) {
		return
	}
	col := c.columns[c.ColCursor]
	if col.sort == nil {
		return
	}
	c.transactions.Sort(col.sort(order))
	for i := range c.columns {
		c.columns[i].sorted = (i == c.ColCursor)
	}
}

func (c *Transactions) Render(width, height int) string {
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

	dataHeight := height - 2
	items := c.transactions.List()
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

	b.WriteString(renderFooter(width, "F2", "Menu", "Enter", "Transaction", "F10", "Quit"))
	return b.String()
}

func NewTransactions(cfg *config.View, txs *models.Transactions) *Transactions {
	transactions := &Transactions{cfg: cfg, transactions: txs}
	printer := message.NewPrinter(language.English)

	columns := DefaultTransactionsColumns
	if cfg != nil && len(cfg.Columns) != 0 {
		columns = cfg.Columns
	}
	transactions.columns = make([]transactionsColumn, len(columns))

	for i := range columns {
		switch columns[i] {
		case "DATE":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%-15s", columns[i]), width: 15,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.DateSort(&tx1.Date, &tx2.Date, order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.Cyan(opts...)(fmt.Sprintf("%15s", tx.Date.Format("15:04:05 Jan _2")))
				},
			}
		case "HEIGHT":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%8s", columns[i]), width: 8,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.Int32Sort(tx1.BlockHeight, tx2.BlockHeight, order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%8d", tx.BlockHeight))
				},
			}
		case "CONFIR":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%8s", columns[i]), width: 8,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.Int32Sort(tx1.NumConfirmations, tx2.NumConfirmations, order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					n := fmt.Sprintf("%8d", tx.NumConfirmations)
					if tx.NumConfirmations < 6 {
						return color.Yellow(opts...)(n)
					}
					return color.Green(opts...)(n)
				},
			}
		case "AMOUNT":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%13s", columns[i]), width: 13,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.Int64Sort(tx1.Amount, tx2.Amount, order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(printer.Sprintf("%13d", tx.Amount))
				},
			}
		case "FEE":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%8s", columns[i]), width: 8,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.Int64Sort(tx1.TotalFees, tx2.TotalFees, order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%8d", tx.TotalFees))
				},
			}
		case "ADDRESSES":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%10s", columns[i]), width: 10,
				sort: func(order models.Order) models.TransactionsSort {
					return func(tx1, tx2 *netmodels.Transaction) bool {
						return models.IntSort(len(tx1.DestAddresses), len(tx2.DestAddresses), order)
					}
				},
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%10d", len(tx.DestAddresses)))
				},
			}
		case "TXHASH":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%-64s", columns[i]), width: 64,
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%13s", tx.TxHash))
				},
			}
		case "BLOCKHASH":
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%-64s", columns[i]), width: 64,
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%13s", tx.TxHash))
				},
			}
		default:
			transactions.columns[i] = transactionsColumn{
				name: fmt.Sprintf("%-21s", columns[i]), width: 21,
				display: func(tx *netmodels.Transaction, opts ...color.Option) string {
					return "column does not exist"
				},
			}
		}
	}
	return transactions
}
