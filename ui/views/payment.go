package views

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/models"
)

type Payment struct {
	payments *models.Payments
	Offset   int
}

func (c *Payment) Name() string { return PAYMENTS }
func (c *Payment) ScrollDown()  { c.Offset++ }
func (c *Payment) ScrollUp() {
	if c.Offset > 0 {
		c.Offset--
	}
}
func (c *Payment) ScrollHome() { c.Offset = 0 }

func (c *Payment) Render(width, height int) string {
	var b strings.Builder
	b.WriteString(DetailHeaderStyle.Width(width).Render("Payment"))
	b.WriteString("\n")

	lines := c.buildContent()
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

	b.WriteString(renderFooter(width, "F2", "Menu", "Enter", "Payments", "F9", "Settings", "F10", "Quit"))
	return b.String()
}

func (c *Payment) buildContent() []string {
	payment := c.payments.Current()
	if payment == nil {
		return []string{
			sectionTitleStyle.Render(" Payment "),
			"No payment selected.",
		}
	}

	p := message.NewPrinter(language.English)
	dl := detailLabelStyle.Render
	lines := []string{
		sectionTitleStyle.Render(" Payment "),
		fmt.Sprintf("%s %s", dl("            Type:"), payment.Kind.String()),
		fmt.Sprintf("%s %s", dl("          Status:"), payment.Status.String()),
		fmt.Sprintf("%s %s", dl("         Created:"), formatNsTime(payment.CreatedAtNs())),
		p.Sprintf("%s %d", dl("           Index:"), payment.PaymentIndex),
		p.Sprintf("%s %d", dl("        Attempts:"), payment.Attempts),
		p.Sprintf("%s %d sat / %d msat", dl("          Amount:"), payment.ValueSat, payment.ValueMsat),
		p.Sprintf("%s %d sat / %d msat", dl("             Fee:"), payment.FeeSat, payment.FeeMsat),
		fmt.Sprintf("%s %s", dl("  Failure Reason:"), displayDetailValue(payment.FailureReason)),
		fmt.Sprintf("%s %s", dl("    Payment Hash:"), displayDetailValue(payment.PaymentHash)),
		fmt.Sprintf("%s %s", dl(" Payment Preimage:"), displayDetailValue(payment.PaymentPreimage)),
	}

	if payment.PayReq != nil {
		lines = append(lines, "")
		lines = append(lines, sectionTitleStyle.Render(" Invoice "))
		lines = append(lines, fmt.Sprintf("%s %s", dl("     Destination:"), displayDetailValue(payment.PayReq.Destination)))
		lines = append(lines, p.Sprintf("%s %d", dl("          Expiry:"), payment.PayReq.Expiry))
		lines = append(lines, fmt.Sprintf("%s %s", dl("     Description:"), displayDetailValue(payment.PayReq.Description)))
	}

	lines = append(lines, "")
	lines = append(lines, sectionTitleStyle.Render(" Route Summary "))
	lines = append(lines, routeSummaryLines(payment.Route)...)
	lines = append(lines, routeHopLines(payment.Route)...)

	lines = append(lines, "")
	lines = append(lines, sectionTitleStyle.Render(" Attempts "))
	if len(payment.AttemptDetails) == 0 {
		lines = append(lines, "No HTLC attempt details available.")
	} else {
		for i, attempt := range payment.AttemptDetails {
			lines = append(lines, fmt.Sprintf(" Attempt %d ", i+1))
			lines = append(lines, p.Sprintf("%s %d", dl("           ID:"), attempt.AttemptID))
			lines = append(lines, fmt.Sprintf("%s %s", dl("       Status:"), attempt.Status.String()))
			lines = append(lines, fmt.Sprintf("%s %s", dl("    Attempted:"), formatNsTime(attempt.AttemptTimeNs)))
			lines = append(lines, fmt.Sprintf("%s %s", dl("     Resolved:"), formatNsTime(attempt.ResolveTimeNs)))
			lines = append(lines, fmt.Sprintf("%s %s", dl(" Failure Code:"), displayDetailValue(attempt.FailureCode)))
			if attempt.FailureSourceIndex != 0 {
				lines = append(lines, p.Sprintf("%s %d", dl("Failure Source:"), attempt.FailureSourceIndex))
			}
			if attempt.FailureChannelID != 0 {
				lines = append(lines, fmt.Sprintf("%s %d (%s)", dl(" Failure Chan:"), attempt.FailureChannelID, ToScid(attempt.FailureChannelID)))
			}
			if attempt.FailureHTLCMsat != 0 {
				lines = append(lines, p.Sprintf("%s %d", dl(" Failure Msat:"), attempt.FailureHTLCMsat))
			}
			if attempt.FailureCLTVExpiry != 0 {
				lines = append(lines, p.Sprintf("%s %d", dl(" Failure CLTV:"), attempt.FailureCLTVExpiry))
			}
			if attempt.FailureHeight != 0 {
				lines = append(lines, p.Sprintf("%s %d", dl("Failure Height:"), attempt.FailureHeight))
			}
			if attempt.PaymentPreimage != "" {
				lines = append(lines, fmt.Sprintf("%s %s", dl("     Preimage:"), attempt.PaymentPreimage))
			}
			lines = append(lines, routeSummaryLines(attempt.Route)...)
			lines = append(lines, routeHopLines(attempt.Route)...)
			lines = append(lines, "")
		}
	}

	lines = append(lines, sectionTitleStyle.Render(" Raw "))
	lines = append(lines, fmt.Sprintf("%s %s", dl(" Payment Request:"), displayDetailValue(payment.PaymentRequest)))

	return lines
}

func routeSummaryLines(route *netmodels.Route) []string {
	dl := detailLabelStyle.Render
	p := message.NewPrinter(language.English)
	if route == nil {
		return []string{"No route details available."}
	}

	lines := []string{
		p.Sprintf("%s %d", dl("      Hop Count:"), len(route.Hops)),
		p.Sprintf("%s %d", dl("      Time Lock:"), route.TimeLock),
		p.Sprintf("%s %d sat / %d msat", dl("   Total Amount:"), route.Amount, route.AmountMsat),
		p.Sprintf("%s %d sat / %d msat", dl("      Total Fee:"), route.Fee, route.FeeMsat),
	}
	if route.FirstHopAmountMsat != 0 {
		lines = append(lines, p.Sprintf("%s %d", dl("First Hop Msat:"), route.FirstHopAmountMsat))
	}
	return lines
}

func routeHopLines(route *netmodels.Route) []string {
	if route == nil || len(route.Hops) == 0 {
		return nil
	}

	dl := detailLabelStyle.Render
	p := message.NewPrinter(language.English)
	lines := []string{sectionTitleStyle.Render(" Route Hops ")}
	for i, hop := range route.Hops {
		lines = append(lines, fmt.Sprintf(" Hop %d ", i+1))
		lines = append(lines, fmt.Sprintf("%s %d (%s)", dl("      Channel:"), hop.ChanID, ToScid(hop.ChanID)))
		lines = append(lines, fmt.Sprintf("%s %s", dl("       PubKey:"), displayDetailValue(hop.PubKey)))
		if hop.Alias != "" {
			lines = append(lines, fmt.Sprintf("%s %s", dl("        Alias:"), hop.Alias))
		}
		lines = append(lines, p.Sprintf("%s %d", dl("     Capacity:"), hop.ChanCapacity))
		lines = append(lines, p.Sprintf("%s %d sat / %d msat", dl(" Forward Amt:"), hop.Amount, hop.AmountMsat))
		lines = append(lines, p.Sprintf("%s %d sat / %d msat", dl("          Fee:"), hop.Fee, hop.FeeMsat))
		lines = append(lines, p.Sprintf("%s %d", dl("       Expiry:"), hop.Expiry))
	}
	return lines
}

func formatNsTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(0, ts).Format("15:04:05 Jan _2 2006")
}

func displayDetailValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func NewPayment(payments *models.Payments) *Payment {
	if payments == nil {
		payments = &models.Payments{}
	}
	return &Payment{payments: payments}
}
