package views

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/hieblmi/lntop/ui/models"
)

type Transaction struct {
	transactions *models.Transactions
	Offset       int
}

func (c *Transaction) Name() string { return TRANSACTION }
func (c *Transaction) ScrollDown()  { c.Offset++ }
func (c *Transaction) ScrollUp()    { if c.Offset > 0 { c.Offset-- } }

func (c *Transaction) Render(width, height int) string {
	var b strings.Builder
	b.WriteString(DetailHeaderStyle.Width(width).Render("Transaction"))
	b.WriteString("\n")

	p := message.NewPrinter(language.English)
	tx := c.transactions.Current()
	dl := detailLabelStyle.Render

	var lines []string
	lines = append(lines, sectionTitleStyle.Render(" Transaction "))
	lines = append(lines, fmt.Sprintf("%s %s", dl("           Date:"), tx.Date.Format("15:04:05 Jan _2")))
	lines = append(lines, p.Sprintf("%s %d", dl("         Amount:"), tx.Amount))
	lines = append(lines, p.Sprintf("%s %d", dl("            Fee:"), tx.TotalFees))
	lines = append(lines, p.Sprintf("%s %d", dl("    BlockHeight:"), tx.BlockHeight))
	lines = append(lines, p.Sprintf("%s %d", dl("NumConfirmations:"), tx.NumConfirmations))
	lines = append(lines, p.Sprintf("%s %s", dl("       BlockHash:"), tx.BlockHash))
	lines = append(lines, fmt.Sprintf("%s %s", dl("         TxHash:"), tx.TxHash))
	lines = append(lines, "")
	lines = append(lines, sectionTitleStyle.Render(" Addresses "))
	for i := range tx.DestAddresses {
		lines = append(lines, fmt.Sprintf("%s %s", dl("               -"), tx.DestAddresses[i]))
	}

	dataHeight := height - 2
	if c.Offset > len(lines)-dataHeight {
		c.Offset = len(lines) - dataHeight
	}
	if c.Offset < 0 {
		c.Offset = 0
	}
	end := min(c.Offset+dataHeight, len(lines))
	for i := c.Offset; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	for i := end - c.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(width, "F2", "Menu", "Enter", "Transactions", "F10", "Quit"))
	return b.String()
}

func NewTransaction(transactions *models.Transactions) *Transaction {
	return &Transaction{transactions: transactions}
}
