package views

import (
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/models"
)

// LoopSwap is the detail view shown when Enter is pressed on a Swaps row.
type LoopSwap struct {
	swaps  *models.LoopSwaps
	Offset int
}

func (v *LoopSwap) ScrollDown() { v.Offset++ }
func (v *LoopSwap) ScrollUp() {
	if v.Offset > 0 {
		v.Offset--
	}
}
func (v *LoopSwap) ScrollHome() { v.Offset = 0 }

func (v *LoopSwap) Render(width, height int) string {
	var b strings.Builder

	b.WriteString(DetailHeaderStyle.Width(width).Render("Loop swap"))
	b.WriteString("\n")

	swap := v.swaps.Current()
	lines := v.buildContent(swap)

	dataHeight := height - 2
	if v.Offset > len(lines)-dataHeight {
		v.Offset = len(lines) - dataHeight
	}
	if v.Offset < 0 {
		v.Offset = 0
	}
	end := v.Offset + dataHeight
	if end > len(lines) {
		end = len(lines)
	}
	for i := v.Offset; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	for i := end - v.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(width,
		"F2", "Menu", "Enter", "Back", "F10", "Quit"))
	return b.String()
}

func (v *LoopSwap) buildContent(swap *netmodels.LoopSwap) []string {
	if swap == nil {
		return []string{detailLabelStyle.Render("no swap selected")}
	}

	p := message.NewPrinter(language.English)
	label := detailLabelStyle.Render
	title := sectionTitleStyle.Render

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	add(title("Identity"))
	add(fmt.Sprintf("%s %s", label("type   :"), swap.Type.String()))
	add(fmt.Sprintf("%s %s", label("hash   :"), swap.ID))
	add(fmt.Sprintf("%s %s", label("state  :"), swap.State))
	if swap.FailureReason != "" {
		add(fmt.Sprintf("%s %s", label("failure:"), swap.FailureReason))
	}
	if swap.Label != "" {
		add(fmt.Sprintf("%s %s", label("label  :"), swap.Label))
	}
	add("")

	add(title("Amounts"))
	add(fmt.Sprintf("%s %s", label("amount         :"), formatSats(p, swap.Amount)))
	if swap.PaymentReqAmount > 0 {
		add(fmt.Sprintf("%s %s", label("invoiced       :"), formatSats(p, swap.PaymentReqAmount)))
	}
	if hasLoopCostColumns(swap) {
		add(fmt.Sprintf("%s %s", label("cost server    :"), formatSats(p, swap.CostServer)))
		add(fmt.Sprintf("%s %s", label("cost on-chain  :"), formatSats(p, swap.CostOnchain)))
		add(fmt.Sprintf("%s %s", label("cost off-chain :"), formatSats(p, swap.CostOffchain)))
	}
	add("")

	add(title("Timing"))
	if swap.InitiationTime != nil {
		add(fmt.Sprintf("%s %s", label("initiated  :"),
			swap.InitiationTime.Format("2006-01-02 15:04:05")))
	}
	if swap.LastUpdate != nil {
		add(fmt.Sprintf("%s %s", label("last update:"),
			swap.LastUpdate.Format("2006-01-02 15:04:05")))
	}
	add("")

	if swap.HtlcAddrP2WSH != "" || swap.HtlcAddrP2TR != "" {
		add(title("HTLC addresses"))
		if swap.HtlcAddrP2WSH != "" {
			add(fmt.Sprintf("%s %s", label("p2wsh:"), swap.HtlcAddrP2WSH))
		}
		if swap.HtlcAddrP2TR != "" {
			add(fmt.Sprintf("%s %s", label("p2tr :"), swap.HtlcAddrP2TR))
		}
		add("")
	}

	if len(swap.LastHop) > 0 {
		add(title("Routing"))
		add(fmt.Sprintf("%s %s", label("last hop:"),
			hex.EncodeToString(swap.LastHop)))
		if len(swap.OutgoingChans) > 0 {
			chans := make([]string, 0, len(swap.OutgoingChans))
			for _, c := range swap.OutgoingChans {
				chans = append(chans, fmt.Sprintf("%d", c))
			}
			add(fmt.Sprintf("%s %s", label("out chans:"),
				strings.Join(chans, ", ")))
		}
		add("")
	}

	if len(swap.DepositOutpoints) > 0 {
		add(title("Static-address deposits used"))
		for _, op := range swap.DepositOutpoints {
			add("  " + op)
		}
		add("")
	}

	if swap.Type == netmodels.LoopSwapTypeInstantOut {
		add(title("Instant out"))
		if swap.SweepTxID != "" {
			add(fmt.Sprintf("%s %s", label("sweep tx:"), swap.SweepTxID))
		}
		if len(swap.ReservationIDs) > 0 {
			ids := make([]string, 0, len(swap.ReservationIDs))
			for _, r := range swap.ReservationIDs {
				ids = append(ids, hex.EncodeToString(r))
			}
			add(fmt.Sprintf("%s %s", label("reservations:"),
				strings.Join(ids, ", ")))
		}
	}

	return lines
}

func NewLoopSwap(swaps *models.LoopSwaps) *LoopSwap {
	if swaps == nil {
		swaps = &models.LoopSwaps{}
	}
	return &LoopSwap{swaps: swaps}
}
