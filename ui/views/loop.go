package views

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/ui/color"
	"github.com/hieblmi/lntop/ui/models"
)

// Loop sub-tabs.
const (
	LoopTabSwaps    = 0
	LoopTabDeposits = 1
)

var DefaultLoopSwapsColumns = []string{
	"TYPE", "TIME", "STATE", "AMOUNT",
	"COST_SRV", "COST_CHAIN", "COST_OFFCH",
	"LABEL", "ID", "FAILURE",
}

var DefaultLoopDepositsColumns = []string{
	"STATE", "AMOUNT", "OUTPOINT",
	"CONF_HEIGHT", "BLOCKS_LEFT", "SWAP_HASH",
}

var (
	loopHeaderTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a78bfa")).
				Bold(true)

	loopHeaderLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6366f1"))

	loopHeaderValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e0e7ff")).
				Bold(true)

	loopAutoloopOnStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#22c55e")).
				Bold(true)

	loopAutoloopOffStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8"))

	loopTabActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#5b37b7")).
				Bold(true).
				Padding(0, 1)

	loopTabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8")).
				Background(lipgloss.Color("#1f1b4d")).
				Padding(0, 1)
)

// Loop is the parent view for the LOOP menu entry. It renders a small
// daemon-level header, a tab strip, then the active sub-tab's table.
type Loop struct {
	swapsCfg    *config.View
	depositsCfg *config.View

	info     *models.LoopInfoModel
	swaps    *LoopSwapsTable
	deposits *LoopDepositsTable

	ActiveTab int
	// Error populated when the most recent loop load failed. Cleared on a
	// successful refresh.
	Error string
}

func (l *Loop) Name() string { return LOOP }

func (l *Loop) SetActiveTab(t int) {
	if t < LoopTabSwaps || t > LoopTabDeposits {
		return
	}
	l.ActiveTab = t
}

func (l *Loop) CursorUp() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.CursorUp()
	case LoopTabDeposits:
		l.deposits.CursorUp()
	}
}

func (l *Loop) CursorDown() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.CursorDown()
	case LoopTabDeposits:
		l.deposits.CursorDown()
	}
}

func (l *Loop) ColumnLeft() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.ColumnLeft()
	case LoopTabDeposits:
		l.deposits.ColumnLeft()
	}
}

func (l *Loop) ColumnRight() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.ColumnRight()
	case LoopTabDeposits:
		l.deposits.ColumnRight()
	}
}

func (l *Loop) Home() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.Home()
	case LoopTabDeposits:
		l.deposits.Home()
	}
}

func (l *Loop) End() {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.End()
	case LoopTabDeposits:
		l.deposits.End()
	}
}

func (l *Loop) PageDown(ps int) {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.PageDown(ps)
	case LoopTabDeposits:
		l.deposits.PageDown(ps)
	}
}

func (l *Loop) PageUp(ps int) {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.PageUp(ps)
	case LoopTabDeposits:
		l.deposits.PageUp(ps)
	}
}

func (l *Loop) Sort(_ string, order models.Order) {
	switch l.ActiveTab {
	case LoopTabSwaps:
		l.swaps.Sort(order)
	case LoopTabDeposits:
		l.deposits.Sort(order)
	}
}

// Index returns the cursor row of the active sub-tab. Used by detail-open.
func (l *Loop) Index() int {
	switch l.ActiveTab {
	case LoopTabSwaps:
		return l.swaps.Index()
	case LoopTabDeposits:
		return l.deposits.Index()
	}
	return 0
}

// CurrentSwap returns the row under the cursor when the Swaps tab is active.
func (l *Loop) CurrentSwap() *netmodels.LoopSwap {
	if l.ActiveTab != LoopTabSwaps {
		return nil
	}
	return l.swaps.Current()
}

// CurrentDeposit returns the row under the cursor when Deposits is active.
func (l *Loop) CurrentDeposit() *netmodels.LoopDeposit {
	if l.ActiveTab != LoopTabDeposits {
		return nil
	}
	return l.deposits.Current()
}

func (l *Loop) Render(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(l.renderTabs(width))
	b.WriteString("\n")

	bodyHeight := height - 1
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	switch l.ActiveTab {
	case LoopTabSwaps:
		b.WriteString(l.swaps.Render(width, bodyHeight))
	case LoopTabDeposits:
		b.WriteString(l.deposits.Render(width, bodyHeight))
	}

	return b.String()
}

func (l *Loop) renderHeader(width int) string {
	p := message.NewPrinter(language.English)
	info := l.info.LoopInfo

	if info == nil {
		msg := "Connecting to loopd…"
		if l.Error != "" {
			msg = fmt.Sprintf("loopd unreachable: %s", l.Error)
		}
		return loopHeaderTitleStyle.Render(msg)
	}

	autoloop := loopAutoloopOffStyle.Render("OFF")
	if info.AutoloopOn {
		autoloop = loopAutoloopOnStyle.Render("ON")
	}

	line1 := strings.Join([]string{
		loopHeaderTitleStyle.Render("Loop"),
		loopHeaderValueStyle.Render("v" + info.Version),
		loopHeaderLabelStyle.Render("net") + " " +
			loopHeaderValueStyle.Render(info.Network),
		loopHeaderLabelStyle.Render("autoloop") + " " + autoloop,
		p.Sprintf("%s %s",
			loopHeaderLabelStyle.Render("budget"),
			loopHeaderValueStyle.Render(formatSats(p, int64(info.AutoloopBudget)))),
	}, "  ")

	outLine := p.Sprintf("%s %d pending / %d ok / %d fail   Σ pend %s   Σ ok %s",
		loopHeaderLabelStyle.Render("OUT"),
		info.OutPending, info.OutSuccess, info.OutFail,
		loopHeaderValueStyle.Render(formatSats(p, info.OutSumPend)),
		loopHeaderValueStyle.Render(formatSats(p, info.OutSumOk)),
	)
	inLine := p.Sprintf("%s  %d pending / %d ok / %d fail   Σ pend %s   Σ ok %s",
		loopHeaderLabelStyle.Render("IN "),
		info.InPending, info.InSuccess, info.InFail,
		loopHeaderValueStyle.Render(formatSats(p, info.InSumPend)),
		loopHeaderValueStyle.Render(formatSats(p, info.InSumOk)),
	)

	parts := []string{line1, outLine, inLine}
	if info.StaticAddress != "" {
		parts = append(parts, loopHeaderLabelStyle.Render("StaticAddr"))
		parts = append(parts, loopHeaderValueStyle.Render(info.StaticAddress))
		parts = append(parts, p.Sprintf("%s %s   %s %s",
			loopHeaderLabelStyle.Render("deposited"),
			loopHeaderValueStyle.Render(formatSats(p, info.StaticDeposited)),
			loopHeaderLabelStyle.Render("looped-in"),
			loopHeaderValueStyle.Render(formatSats(p, info.StaticLoopedIn))))
	}
	if l.Error != "" {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fca5a5")).
				Render("warning: "+l.Error))
	}

	return strings.Join(parts, "\n")
}

func (l *Loop) renderTabs(width int) string {
	tabs := []struct {
		label string
		key   string
		index int
	}{
		{"Swaps", "1", LoopTabSwaps},
		{"Deposits", "2", LoopTabDeposits},
	}
	parts := make([]string, 0, len(tabs))
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.key, t.label)
		if t.index == l.ActiveTab {
			parts = append(parts, loopTabActiveStyle.Render(label))
		} else {
			parts = append(parts, loopTabInactiveStyle.Render(label))
		}
	}
	row := strings.Join(parts, " ")
	if lipgloss.Width(row) > width {
		row = ansi.Truncate(row, width, "")
	}
	return row
}

func NewLoop(
	swapsCfg *config.View, depositsCfg *config.View,
	info *models.LoopInfoModel,
	swaps *models.LoopSwaps, deposits *models.LoopDeposits,
) *Loop {
	if info == nil {
		info = &models.LoopInfoModel{}
	}
	if swaps == nil {
		swaps = &models.LoopSwaps{}
	}
	if deposits == nil {
		deposits = &models.LoopDeposits{}
	}
	return &Loop{
		swapsCfg:    swapsCfg,
		depositsCfg: depositsCfg,
		info:        info,
		swaps:       newLoopSwapsTable(swapsCfg, swaps),
		deposits:    newLoopDepositsTable(depositsCfg, deposits),
	}
}

// ---------- Swaps sub-tab table ----------

type loopSwapsColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.LoopSwapsSort
	display func(*netmodels.LoopSwap, ...color.Option) string
}

// LoopSwapsTable is the Swaps sub-tab table. Public type so tests can poke
// it directly; only the Loop view actually wires it up.
type LoopSwapsTable struct {
	cfg       *config.View
	columns   []loopSwapsColumn
	swaps     *models.LoopSwaps
	Cursor    int
	Offset    int
	ColCursor int
}

func (t *LoopSwapsTable) CursorDown() {
	if t.Cursor < t.swaps.Len()-1 {
		t.Cursor++
	}
}
func (t *LoopSwapsTable) CursorUp() {
	if t.Cursor > 0 {
		t.Cursor--
	}
}
func (t *LoopSwapsTable) ColumnRight() {
	if t.ColCursor < len(t.columns)-1 {
		t.ColCursor++
	}
}
func (t *LoopSwapsTable) ColumnLeft() {
	if t.ColCursor > 0 {
		t.ColCursor--
	}
}
func (t *LoopSwapsTable) Home() { t.Cursor = 0 }
func (t *LoopSwapsTable) End() {
	t.Cursor = max(0, t.swaps.Len()-1)
}
func (t *LoopSwapsTable) PageDown(ps int) {
	t.Cursor = min(t.Cursor+ps, max(0, t.swaps.Len()-1))
}
func (t *LoopSwapsTable) PageUp(ps int) {
	t.Cursor = max(0, t.Cursor-ps)
}
func (t *LoopSwapsTable) Index() int { return t.Cursor }

func (t *LoopSwapsTable) Current() *netmodels.LoopSwap {
	return t.swaps.Get(t.Cursor)
}

func (t *LoopSwapsTable) Sort(order models.Order) {
	if t.ColCursor >= len(t.columns) {
		return
	}
	col := t.columns[t.ColCursor]
	if col.sort == nil {
		return
	}
	t.swaps.Sort(col.sort(order))
	for i := range t.columns {
		t.columns[i].sorted = (i == t.ColCursor)
	}
}

func (t *LoopSwapsTable) Render(width, height int) string {
	var b strings.Builder
	colWidths := make([]int, len(t.columns))
	for i := range t.columns {
		colWidths[i] = t.columns[i].width
	}
	visibleStart, visibleEnd := visibleColumnRange(width, t.ColCursor, colWidths)

	var hdr strings.Builder
	for i := visibleStart; i < visibleEnd; i++ {
		col := t.columns[i]
		style := DefaultColStyle
		if i == t.ColCursor {
			style = ActiveColStyle
		} else if col.sorted {
			style = SortedColStyle
		}
		hdr.WriteString(renderHeaderCell(col.name, col.width, style))
		hdr.WriteString(" ")
	}
	b.WriteString(renderTableHeader(hdr.String(), width))
	b.WriteString("\n")

	dataHeight := height - 2
	items := t.swaps.List()
	if t.Cursor >= len(items) {
		t.Cursor = max(0, len(items)-1)
	}
	if t.Cursor < t.Offset {
		t.Offset = t.Cursor
	}
	if t.Cursor >= t.Offset+dataHeight {
		t.Offset = t.Cursor - dataHeight + 1
	}
	end := min(t.Offset+dataHeight, len(items))

	for idx := t.Offset; idx < end; idx++ {
		item := items[idx]
		var row strings.Builder
		for i := visibleStart; i < visibleEnd; i++ {
			col := t.columns[i]
			var opt color.Option
			if i == t.ColCursor {
				opt = color.Bold
			}
			row.WriteString(fitCell(col.display(item, opt), col.width))
			row.WriteString(" ")
		}
		line := row.String()
		if idx == t.Cursor {
			line = selectedRow(line, width)
		} else {
			line = safeTruncRow(line, width)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	for i := end - t.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(width,
		"F2", "Menu",
		"Tab", "Switch tab",
		"Enter", "Detail",
		"F10", "Quit",
		fmt.Sprintf("  Swaps: %d", t.swaps.Len())))
	return b.String()
}

func newLoopSwapsTable(cfg *config.View, swaps *models.LoopSwaps) *LoopSwapsTable {
	t := &LoopSwapsTable{cfg: cfg, swaps: swaps}
	p := message.NewPrinter(language.English)

	cols := DefaultLoopSwapsColumns
	if cfg != nil && len(cfg.Columns) > 0 {
		cols = cfg.Columns
	}
	t.columns = make([]loopSwapsColumn, len(cols))
	for i := range cols {
		switch cols[i] {
		case "TYPE":
			t.columns[i] = loopSwapsColumn{
				width: 12, name: fmt.Sprintf("%-12s", cols[i]),
				sort: func(order models.Order) models.LoopSwapsSort {
					return func(a, b *netmodels.LoopSwap) bool {
						return models.IntSort(int(a.Type), int(b.Type), order)
					}
				},
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-12s", s.Type.String()))
				},
			}
		case "TIME":
			t.columns[i] = loopSwapsColumn{
				width: 25, name: fmt.Sprintf("%25s", cols[i]),
				sort: func(order models.Order) models.LoopSwapsSort {
					return func(a, b *netmodels.LoopSwap) bool {
						return models.DateSort(a.InitiationTime, b.InitiationTime, order)
					}
				},
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					if s.InitiationTime == nil {
						return color.White(opts...)(fmt.Sprintf("%25s", "-"))
					}
					return color.White(opts...)(fmt.Sprintf("%25s", s.InitiationTime.Format("15:04:05 Jan _2 2006")))
				},
			}
		case "STATE":
			t.columns[i] = loopSwapsColumn{
				width: 24, name: fmt.Sprintf("%-24s", cols[i]),
				sort: func(order models.Order) models.LoopSwapsSort {
					return func(a, b *netmodels.LoopSwap) bool {
						return models.StringSort(a.State, b.State, order)
					}
				},
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return colorizeSwapState(s.State, opts...)(fmt.Sprintf("%-24s", truncate(s.State, 24)))
				},
			}
		case "AMOUNT":
			t.columns[i] = loopSwapsColumn{
				width: 16, name: fmt.Sprintf("%16s", cols[i]),
				sort: func(order models.Order) models.LoopSwapsSort {
					return func(a, b *netmodels.LoopSwap) bool {
						return models.Int64Sort(a.Amount, b.Amount, order)
					}
				},
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return color.White(opts...)(p.Sprintf("%16d", s.Amount))
				},
			}
		case "COST_SRV":
			t.columns[i] = loopSwapsColumn{
				width: 14, name: fmt.Sprintf("%14s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return formatLoopCost(p, 14, s, s.CostServer, opts...)
				},
			}
		case "COST_CHAIN":
			t.columns[i] = loopSwapsColumn{
				width: 14, name: fmt.Sprintf("%14s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return formatLoopCost(p, 14, s, s.CostOnchain, opts...)
				},
			}
		case "COST_OFFCH":
			t.columns[i] = loopSwapsColumn{
				width: 14, name: fmt.Sprintf("%14s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return formatLoopCost(p, 14, s, s.CostOffchain, opts...)
				},
			}
		case "LABEL":
			t.columns[i] = loopSwapsColumn{
				width: 24, name: fmt.Sprintf("%-24s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-24s", truncate(s.Label, 24)))
				},
			}
		case "ID":
			t.columns[i] = loopSwapsColumn{
				width: 32, name: fmt.Sprintf("%-32s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-32s", truncate(s.ID, 32)))
				},
			}
		case "FAILURE":
			t.columns[i] = loopSwapsColumn{
				width: 24, name: fmt.Sprintf("%-24s", cols[i]),
				display: func(s *netmodels.LoopSwap, opts ...color.Option) string {
					if s.FailureReason == "" {
						return color.White(opts...)(fmt.Sprintf("%-24s", "-"))
					}
					return color.Red(opts...)(fmt.Sprintf("%-24s", truncate(s.FailureReason, 24)))
				},
			}
		default:
			t.columns[i] = loopSwapsColumn{
				width: 4, name: "?",
				display: func(_ *netmodels.LoopSwap, opts ...color.Option) string {
					return color.White(opts...)("-")
				},
			}
		}
	}
	return t
}

// ---------- Deposits sub-tab table ----------

type loopDepositsColumn struct {
	name    string
	width   int
	sorted  bool
	sort    func(models.Order) models.LoopDepositsSort
	display func(*netmodels.LoopDeposit, ...color.Option) string
}

type LoopDepositsTable struct {
	cfg       *config.View
	columns   []loopDepositsColumn
	deposits  *models.LoopDeposits
	Cursor    int
	Offset    int
	ColCursor int
}

func (t *LoopDepositsTable) CursorDown() {
	if t.Cursor < t.deposits.Len()-1 {
		t.Cursor++
	}
}
func (t *LoopDepositsTable) CursorUp() {
	if t.Cursor > 0 {
		t.Cursor--
	}
}
func (t *LoopDepositsTable) ColumnRight() {
	if t.ColCursor < len(t.columns)-1 {
		t.ColCursor++
	}
}
func (t *LoopDepositsTable) ColumnLeft() {
	if t.ColCursor > 0 {
		t.ColCursor--
	}
}
func (t *LoopDepositsTable) Home() { t.Cursor = 0 }
func (t *LoopDepositsTable) End() {
	t.Cursor = max(0, t.deposits.Len()-1)
}
func (t *LoopDepositsTable) PageDown(ps int) {
	t.Cursor = min(t.Cursor+ps, max(0, t.deposits.Len()-1))
}
func (t *LoopDepositsTable) PageUp(ps int) {
	t.Cursor = max(0, t.Cursor-ps)
}
func (t *LoopDepositsTable) Index() int { return t.Cursor }

func (t *LoopDepositsTable) Current() *netmodels.LoopDeposit {
	return t.deposits.Get(t.Cursor)
}

func (t *LoopDepositsTable) Sort(order models.Order) {
	if t.ColCursor >= len(t.columns) {
		return
	}
	col := t.columns[t.ColCursor]
	if col.sort == nil {
		return
	}
	t.deposits.Sort(col.sort(order))
	for i := range t.columns {
		t.columns[i].sorted = (i == t.ColCursor)
	}
}

func (t *LoopDepositsTable) Render(width, height int) string {
	var b strings.Builder
	colWidths := make([]int, len(t.columns))
	for i := range t.columns {
		colWidths[i] = t.columns[i].width
	}
	visibleStart, visibleEnd := visibleColumnRange(width, t.ColCursor, colWidths)

	var hdr strings.Builder
	for i := visibleStart; i < visibleEnd; i++ {
		col := t.columns[i]
		style := DefaultColStyle
		if i == t.ColCursor {
			style = ActiveColStyle
		} else if col.sorted {
			style = SortedColStyle
		}
		hdr.WriteString(renderHeaderCell(col.name, col.width, style))
		hdr.WriteString(" ")
	}
	b.WriteString(renderTableHeader(hdr.String(), width))
	b.WriteString("\n")

	dataHeight := height - 2
	items := t.deposits.List()
	if t.Cursor >= len(items) {
		t.Cursor = max(0, len(items)-1)
	}
	if t.Cursor < t.Offset {
		t.Offset = t.Cursor
	}
	if t.Cursor >= t.Offset+dataHeight {
		t.Offset = t.Cursor - dataHeight + 1
	}
	end := min(t.Offset+dataHeight, len(items))

	for idx := t.Offset; idx < end; idx++ {
		item := items[idx]
		var row strings.Builder
		for i := visibleStart; i < visibleEnd; i++ {
			col := t.columns[i]
			var opt color.Option
			if i == t.ColCursor {
				opt = color.Bold
			}
			row.WriteString(fitCell(col.display(item, opt), col.width))
			row.WriteString(" ")
		}
		line := row.String()
		if idx == t.Cursor {
			line = selectedRow(line, width)
		} else {
			line = safeTruncRow(line, width)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	for i := end - t.Offset; i < dataHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(renderFooter(width,
		"F2", "Menu",
		"Tab", "Switch tab",
		"F10", "Quit",
		fmt.Sprintf("  Deposits: %d", t.deposits.Len())))
	return b.String()
}

func newLoopDepositsTable(cfg *config.View, deposits *models.LoopDeposits) *LoopDepositsTable {
	t := &LoopDepositsTable{cfg: cfg, deposits: deposits}
	p := message.NewPrinter(language.English)

	cols := DefaultLoopDepositsColumns
	if cfg != nil && len(cfg.Columns) > 0 {
		cols = cfg.Columns
	}
	t.columns = make([]loopDepositsColumn, len(cols))
	for i := range cols {
		switch cols[i] {
		case "STATE":
			t.columns[i] = loopDepositsColumn{
				width: 24, name: fmt.Sprintf("%-24s", cols[i]),
				sort: func(order models.Order) models.LoopDepositsSort {
					return func(a, b *netmodels.LoopDeposit) bool {
						return models.IntSort(int(a.State), int(b.State), order)
					}
				},
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					return colorizeDepositState(d.State, opts...)(fmt.Sprintf("%-24s", truncate(d.State.String(), 24)))
				},
			}
		case "AMOUNT":
			t.columns[i] = loopDepositsColumn{
				width: 16, name: fmt.Sprintf("%16s", cols[i]),
				sort: func(order models.Order) models.LoopDepositsSort {
					return func(a, b *netmodels.LoopDeposit) bool {
						return models.Int64Sort(a.Value, b.Value, order)
					}
				},
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					return color.White(opts...)(p.Sprintf("%16d", d.Value))
				},
			}
		case "OUTPOINT":
			t.columns[i] = loopDepositsColumn{
				width: 72, name: fmt.Sprintf("%-72s", cols[i]),
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					return color.White(opts...)(fmt.Sprintf("%-72s", truncate(d.Outpoint, 72)))
				},
			}
		case "CONF_HEIGHT":
			t.columns[i] = loopDepositsColumn{
				width: 11, name: fmt.Sprintf("%11s", cols[i]),
				sort: func(order models.Order) models.LoopDepositsSort {
					return func(a, b *netmodels.LoopDeposit) bool {
						return models.Int64Sort(a.ConfirmationHeight, b.ConfirmationHeight, order)
					}
				},
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					return color.White(opts...)(p.Sprintf("%11d", d.ConfirmationHeight))
				},
			}
		case "BLOCKS_LEFT":
			t.columns[i] = loopDepositsColumn{
				width: 11, name: fmt.Sprintf("%11s", cols[i]),
				sort: func(order models.Order) models.LoopDepositsSort {
					return func(a, b *netmodels.LoopDeposit) bool {
						return models.Int64Sort(a.BlocksUntilExpiry, b.BlocksUntilExpiry, order)
					}
				},
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					return colorizeBlocksLeft(d.BlocksUntilExpiry, opts...)(p.Sprintf("%11d", d.BlocksUntilExpiry))
				},
			}
		case "SWAP_HASH":
			t.columns[i] = loopDepositsColumn{
				width: 32, name: fmt.Sprintf("%-32s", cols[i]),
				display: func(d *netmodels.LoopDeposit, opts ...color.Option) string {
					if len(d.SwapHash) == 0 {
						return color.White(opts...)(fmt.Sprintf("%-32s", "-"))
					}
					return color.White(opts...)(fmt.Sprintf("%-32s", truncate(hex.EncodeToString(d.SwapHash), 32)))
				},
			}
		default:
			t.columns[i] = loopDepositsColumn{
				width: 4, name: "?",
				display: func(_ *netmodels.LoopDeposit, opts ...color.Option) string {
					return color.White(opts...)("-")
				},
			}
		}
	}
	return t
}

// ---------- helpers ----------

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// truncateMid keeps the start and end of a long string, replacing the middle
// with "…". Useful for taproot addresses which lose information if truncated
// at the end only.
func truncateMid(s string, n int) string {
	if len(s) <= n || n < 5 {
		return truncate(s, n)
	}
	left := (n - 1) / 2
	right := n - 1 - left
	return s[:left] + "…" + s[len(s)-right:]
}

func formatLoopCost(
	p *message.Printer, width int, s *netmodels.LoopSwap, cost int64,
	opts ...color.Option,
) string {
	if !hasLoopCostColumns(s) {
		return color.White(opts...)(fmt.Sprintf("%*s", width, "-"))
	}
	return color.White(opts...)(p.Sprintf("%*d", width, cost))
}

func hasLoopCostColumns(s *netmodels.LoopSwap) bool {
	if s == nil {
		return false
	}
	return s.Type == netmodels.LoopSwapTypeOut ||
		s.Type == netmodels.LoopSwapTypeIn ||
		s.CostServer != 0 ||
		s.CostOnchain != 0 ||
		s.CostOffchain != 0
}

func colorizeSwapState(state string, opts ...color.Option) func(...interface{}) string {
	switch state {
	case "SUCCESS":
		return color.Green(opts...)
	case "FAILED":
		return color.Red(opts...)
	case "INITIATED", "PREIMAGE_REVEALED", "HTLC_PUBLISHED", "INVOICE_SETTLED":
		return color.Yellow(opts...)
	default:
		return color.White(opts...)
	}
}

func colorizeDepositState(s netmodels.LoopDepositState, opts ...color.Option) func(...interface{}) string {
	switch s {
	case netmodels.LoopDepositStateDeposited,
		netmodels.LoopDepositStateLoopedIn,
		netmodels.LoopDepositStateWithdrawn,
		netmodels.LoopDepositStateChannelPublished:
		return color.Green(opts...)
	case netmodels.LoopDepositStateExpired,
		netmodels.LoopDepositStateSweepHTLCTimeout,
		netmodels.LoopDepositStatePublishExpired:
		return color.Red(opts...)
	case netmodels.LoopDepositStateWithdrawing,
		netmodels.LoopDepositStateLoopingIn,
		netmodels.LoopDepositStateOpeningChannel,
		netmodels.LoopDepositStateHTLCTimeoutSwept,
		netmodels.LoopDepositStateWaitForExpirySweep:
		return color.Yellow(opts...)
	default:
		return color.White(opts...)
	}
}

func colorizeBlocksLeft(blocks int64, opts ...color.Option) func(...interface{}) string {
	switch {
	case blocks <= 0:
		return color.Red(opts...)
	case blocks < 144: // ~1 day
		return color.Red(opts...)
	case blocks < 1008: // ~1 week
		return color.Yellow(opts...)
	default:
		return color.Green(opts...)
	}
}
