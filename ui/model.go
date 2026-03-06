package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hieblmi/lntop/app"
	"github.com/hieblmi/lntop/events"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/ui/models"
	"github.com/hieblmi/lntop/ui/views"
)

type model struct {
	app    *app.App
	logger logging.Logger
	models *models.Models
	views  *views.Views
	sub    chan *events.Event

	width, height int

	activeView  string
	inDetail    bool
	menuOpen    bool
	pulseFrame  int
	pulseActive bool

	infoLoading            bool
	walletBalanceLoading   bool
	channelsBalanceLoading bool
	transactionsLoading    bool
	forwardingHistLoading  bool
	channelsLoading        bool
	receivedLoading        bool
	currentNodeLoading     bool

	startupActive    bool
	startupFinishing bool
	startupTasks     map[string]bool
	startupWaiting   string
}

var startupTaskLabels = []struct {
	key   string
	label string
}{
	{"info", "Node info"},
	{"wallet", "Wallet balance"},
	{"channels_balance", "Channel balances"},
	{"transactions", "Transactions"},
	{"forwarding", "Forwarding history"},
	{"received", "Received invoices"},
	{"channels", "Channels"},
}

func newModel(a *app.App, sub chan *events.Event) *model {
	m := models.New(a)
	return &model{
		app:        a,
		logger:     a.Logger.With(logging.String("logger", "ui")),
		models:     m,
		views:      views.New(a.Config.Views, m),
		sub:        sub,
		activeView: views.CHANNELS,
	}
}

func (m *model) Init() tea.Cmd {
	m.startInitialLoad()
	return tea.Batch(
		waitForEvent(m.sub),
		m.ensurePulseTick(),
		m.loadInfoCmd(),
		m.loadWalletBalanceCmd(),
		m.loadChannelsBalanceCmd(),
		m.loadTransactionsCmd(),
		m.loadForwardingHistoryCmd(),
		m.loadReceivedCmd(),
		m.loadChannelsCmd(),
	)
}

func pulseTickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return pulseTickMsg{}
	})
}

func startupCompleteCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return startupCompleteMsg{}
	})
}

func startupRetryCmd(task string) tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return startupRetryMsg{task: task}
	})
}

func (m *model) loadInitialData(ctx context.Context) {
	fns := []func(context.Context) error{
		m.models.RefreshInfo,
		m.models.RefreshWalletBalance,
		m.models.RefreshChannelsBalance,
		m.models.RefreshTransactions,
		m.models.RefreshForwardingHistory,
		m.models.RefreshReceivedFromNetwork,
		m.models.RefreshChannels,
	}
	for _, fn := range fns {
		if err := fn(ctx); err != nil {
			m.logger.Error("init", logging.Error(err))
		}
	}
}

func waitForEvent(sub chan *events.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-sub
		if !ok {
			return tea.Quit()
		}
		return eventMsg{event}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case eventMsg:
		return m, tea.Batch(waitForEvent(m.sub), m.handleEvent(msg.event), m.ensurePulseTick())

	case pulseTickMsg:
		m.pulseFrame++
		if m.shouldAnimate() {
			m.pulseActive = true
			return m, pulseTickCmd()
		}
		m.pulseActive = false
		return m, nil

	case startupCompleteMsg:
		if m.startupFinishing && !m.hasStartupLoadsInFlight() && len(m.startupTasks) == 0 {
			m.startupActive = false
			m.startupFinishing = false
			m.startupWaiting = ""
		}
		return m, nil

	case startupRetryMsg:
		if !m.startupActive || !m.startupTasks[msg.task] {
			return m, nil
		}
		return m, m.startupRetryTaskCmd(msg.task)

	case infoLoadedMsg:
		m.infoLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Node info"
			m.logger.Error("refresh info failed", logging.Error(msg.err))
			return m, startupRetryCmd("info")
		}
		m.models.ApplyInfo(msg.info)
		m.finishStartupTask("info")
		return m, tea.Batch(m.ensurePulseTick(), m.refreshChannelAgesIfNeeded(), m.completeStartupCmdIfReady())

	case walletBalanceLoadedMsg:
		m.walletBalanceLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Wallet balance"
			m.logger.Error("refresh wallet balance failed", logging.Error(msg.err))
			return m, startupRetryCmd("wallet")
		}
		m.models.ApplyWalletBalance(msg.balance)
		m.finishStartupTask("wallet")
		return m, m.completeStartupCmdIfReady()

	case channelsBalanceLoadedMsg:
		m.channelsBalanceLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Channel balances"
			m.logger.Error("refresh channels balance failed", logging.Error(msg.err))
			return m, startupRetryCmd("channels_balance")
		}
		m.models.ApplyChannelsBalance(msg.balance)
		m.finishStartupTask("channels_balance")
		return m, m.completeStartupCmdIfReady()

	case transactionsLoadedMsg:
		m.transactionsLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Transactions"
			m.logger.Error("refresh transactions failed", logging.Error(msg.err))
			return m, startupRetryCmd("transactions")
		}
		m.models.ApplyTransactions(msg.transactions)
		m.finishStartupTask("transactions")
		return m, m.completeStartupCmdIfReady()

	case forwardingHistoryLoadedMsg:
		m.forwardingHistLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Forwarding history"
			m.logger.Error("refresh forwarding history failed", logging.Error(msg.err))
			return m, startupRetryCmd("forwarding")
		}
		m.models.ApplyForwardingHistory(msg.events)
		m.finishStartupTask("forwarding")
		return m, m.completeStartupCmdIfReady()

	case channelsLoadedMsg:
		m.channelsLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Channels"
			m.logger.Error("refresh channels failed", logging.Error(msg.err))
			return m, startupRetryCmd("channels")
		}
		m.models.ApplyChannels(msg.channels)
		m.finishStartupTask("channels")
		return m, m.completeStartupCmdIfReady()

	case receivedLoadedMsg:
		m.receivedLoading = false
		if msg.err != nil {
			m.startupWaiting = "Retrying Received invoices"
			m.logger.Error("refresh received failed", logging.Error(msg.err))
			return m, startupRetryCmd("received")
		}
		m.models.ApplyReceived(msg.invoices)
		m.finishStartupTask("received")
		return m, m.completeStartupCmdIfReady()

	case currentNodeLoadedMsg:
		m.currentNodeLoading = false
		if msg.err != nil {
			m.logger.Error("refresh current node failed", logging.Error(msg.err))
			return m, nil
		}
		cur := m.models.Channels.Current()
		if cur != nil && cur.RemotePubKey == msg.pubkey {
			m.models.ApplyCurrentNode(msg.node)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) handleEvent(e *events.Event) tea.Cmd {
	m.logger.Debug("event received", logging.String("type", e.Type))
	var cmds []tea.Cmd
	switch e.Type {
	case events.TransactionCreated:
		cmds = append(cmds, m.loadInfoCmd(), m.loadWalletBalanceCmd(), m.loadTransactionsCmd())
	case events.BlockReceived:
		cmds = append(cmds, m.loadInfoCmd(), m.loadTransactionsCmd())
	case events.WalletBalanceUpdated:
		cmds = append(cmds, m.loadInfoCmd(), m.loadWalletBalanceCmd(), m.loadTransactionsCmd(), m.loadForwardingHistoryCmd())
	case events.ChannelBalanceUpdated:
		cmds = append(cmds, m.loadInfoCmd(), m.loadChannelsBalanceCmd(), m.loadChannelsCmd(), m.loadForwardingHistoryCmd())
	case events.ChannelPending, events.ChannelActive, events.ChannelInactive:
		cmds = append(cmds, m.loadInfoCmd(), m.loadChannelsBalanceCmd(), m.loadChannelsCmd())
	case events.InvoiceSettled:
		cmds = append(cmds, m.loadInfoCmd(), m.loadChannelsBalanceCmd(), m.loadChannelsCmd(), m.loadForwardingHistoryCmd())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.models.RefreshReceived(e.Data)(ctx); err != nil {
			m.logger.Error("refresh received failed", logging.Error(err))
		}
	case events.PeerUpdated:
		cmds = append(cmds, m.loadInfoCmd(), m.loadForwardingHistoryCmd())
	case events.RoutingEventUpdated:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.models.RefreshRouting(e.Data)(ctx); err != nil {
			m.logger.Error("refresh routing failed", logging.Error(err))
		}
	case events.GraphUpdated:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.models.RefreshPolicies(e.Data)(ctx); err != nil {
			m.logger.Error("refresh policies failed", logging.Error(err))
		}
	}
	return tea.Batch(cmds...)
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit keys.
	switch msg.String() {
	case "ctrl+c", "f10", "q":
		return m, tea.Quit
	}

	// Menu toggle.
	if msg.String() == "f2" || msg.String() == "m" {
		if m.menuOpen {
			if preview := m.views.Menu.Current(); preview != "" {
				m.activeView = preview
			}
			m.menuOpen = false
		} else {
			m.views.Menu.SetCurrent(m.activeView)
			m.menuOpen = true
		}
		return m, tea.Batch(tea.ClearScreen, m.ensurePulseTick())
	}

	// If menu is open, handle menu navigation.
	if m.menuOpen {
		next, cmd := m.handleMenuKey(msg)
		return next, tea.Batch(cmd, m.ensurePulseTick())
	}

	// If in detail view, handle detail navigation.
	if m.inDetail {
		next, cmd := m.handleDetailKey(msg)
		return next, tea.Batch(cmd, m.ensurePulseTick())
	}

	// Table view navigation.
	next, cmd := m.handleTableKey(msg)
	return next, tea.Batch(cmd, m.ensurePulseTick())
}

func (m *model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.views.Menu.CursorUp()
	case "down", "j":
		m.views.Menu.CursorDown()
	case "enter":
		m.activeView = m.views.Menu.Current()
		m.menuOpen = false
		return m, tea.ClearScreen
	}
	return m, nil
}

func (m *model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.inDetail = false
	case "up", "k":
		switch m.activeView {
		case views.CHANNELS:
			m.views.Channel.ScrollUp()
		case views.TRANSACTIONS:
			m.views.Transaction.ScrollUp()
		}
	case "down", "j":
		switch m.activeView {
		case views.CHANNELS:
			m.views.Channel.ScrollDown()
		case views.TRANSACTIONS:
			m.views.Transaction.ScrollDown()
		}
	case "home", "g":
		if m.activeView == views.CHANNELS {
			m.views.Channel.ScrollHome()
		}
	case "c":
		if m.activeView == views.CHANNELS {
			cur := m.models.Channels.Current()
			if cur != nil {
				return m, m.loadCurrentNodeCmd(cur.RemotePubKey)
			}
		}
	}
	return m, nil
}

func (m *model) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pageSize := m.mainHeight() - 2

	switch msg.String() {
	case "up", "k":
		m.cursorUp()
	case "down", "j":
		m.cursorDown()
	case "left", "h":
		m.columnLeft()
	case "right", "l":
		m.columnRight()
	case "home", "g":
		m.home()
	case "end", "G":
		m.end()
	case "pgdown":
		m.pageDown(pageSize)
	case "pgup":
		m.pageUp(pageSize)
	case "enter":
		m.onEnter()
		if m.activeView == views.CHANNELS {
			if cur := m.models.Channels.Current(); cur != nil {
				return m, m.loadCurrentNodeCmd(cur.RemotePubKey)
			}
		}
	case "a":
		m.sort(models.Asc)
	case "d":
		m.sort(models.Desc)
	case "c":
		if m.activeView == views.CHANNELS {
			idx := m.views.Channels.Index()
			m.models.Channels.SetCurrent(idx)
			cur := m.models.Channels.Current()
			if cur != nil {
				return m, m.loadCurrentNodeCmd(cur.RemotePubKey)
			}
		}
	}
	return m, nil
}

func (m *model) cursorDown() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.CursorDown()
	case views.TRANSACTIONS:
		m.views.Transactions.CursorDown()
	case views.ROUTING:
		m.views.Routing.CursorDown()
	case views.FWDINGHIST:
		m.views.FwdingHist.CursorDown()
	case views.RECEIVED:
		m.views.Received.CursorDown()
	}
}

func (m *model) cursorUp() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.CursorUp()
	case views.TRANSACTIONS:
		m.views.Transactions.CursorUp()
	case views.ROUTING:
		m.views.Routing.CursorUp()
	case views.FWDINGHIST:
		m.views.FwdingHist.CursorUp()
	case views.RECEIVED:
		m.views.Received.CursorUp()
	}
}

func (m *model) columnLeft() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.ColumnLeft()
	case views.TRANSACTIONS:
		m.views.Transactions.ColumnLeft()
	case views.ROUTING:
		m.views.Routing.ColumnLeft()
	case views.FWDINGHIST:
		m.views.FwdingHist.ColumnLeft()
	case views.RECEIVED:
		m.views.Received.ColumnLeft()
	}
}

func (m *model) columnRight() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.ColumnRight()
	case views.TRANSACTIONS:
		m.views.Transactions.ColumnRight()
	case views.ROUTING:
		m.views.Routing.ColumnRight()
	case views.FWDINGHIST:
		m.views.FwdingHist.ColumnRight()
	case views.RECEIVED:
		m.views.Received.ColumnRight()
	}
}

func (m *model) home() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.Home()
	case views.TRANSACTIONS:
		m.views.Transactions.Home()
	case views.ROUTING:
		m.views.Routing.Home()
	case views.FWDINGHIST:
		m.views.FwdingHist.Home()
	case views.RECEIVED:
		m.views.Received.Home()
	}
}

func (m *model) end() {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.End()
	case views.TRANSACTIONS:
		m.views.Transactions.End()
	case views.ROUTING:
		m.views.Routing.End()
	case views.FWDINGHIST:
		m.views.FwdingHist.End()
	case views.RECEIVED:
		m.views.Received.End()
	}
}

func (m *model) pageDown(ps int) {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.PageDown(ps)
	case views.TRANSACTIONS:
		m.views.Transactions.PageDown(ps)
	case views.ROUTING:
		m.views.Routing.PageDown(ps)
	case views.FWDINGHIST:
		m.views.FwdingHist.PageDown(ps)
	case views.RECEIVED:
		m.views.Received.PageDown(ps)
	}
}

func (m *model) pageUp(ps int) {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.PageUp(ps)
	case views.TRANSACTIONS:
		m.views.Transactions.PageUp(ps)
	case views.ROUTING:
		m.views.Routing.PageUp(ps)
	case views.FWDINGHIST:
		m.views.FwdingHist.PageUp(ps)
	case views.RECEIVED:
		m.views.Received.PageUp(ps)
	}
}

func (m *model) sort(order models.Order) {
	switch m.activeView {
	case views.CHANNELS:
		m.views.Channels.Sort("", order)
	case views.TRANSACTIONS:
		m.views.Transactions.Sort("", order)
	case views.ROUTING:
		m.views.Routing.Sort("", order)
	case views.FWDINGHIST:
		m.views.FwdingHist.Sort("", order)
	case views.RECEIVED:
		m.views.Received.Sort("", order)
	}
}

func (m *model) onEnter() {
	switch m.activeView {
	case views.CHANNELS:
		idx := m.views.Channels.Index()
		m.models.Channels.SetCurrent(idx)
		m.views.Channel.Offset = 0
		m.inDetail = true
	case views.TRANSACTIONS:
		idx := m.views.Transactions.Index()
		m.models.Transactions.SetCurrent(idx)
		m.views.Transaction.Offset = 0
		m.inDetail = true
	}
}

// mainHeight returns the height available for the main content area.
func (m *model) mainHeight() int {
	return m.mainHeightForSummary(m.views.Summary.Render(m.renderWidth()))
}

func (m *model) mainHeightForSummary(summary string) int {
	summaryLines := strings.Count(summary, "\n") + 1
	// header(1) + blank(1) + summary.
	used := 1 + 1 + summaryLines
	h := m.height - used
	if h < 3 {
		h = 3
	}
	return h
}

// renderWidth keeps one terminal column free to prevent autowrap artifacts.
func (m *model) renderWidth() int {
	// Keep a right margin to absorb terminal/library width differences
	// (notably around emoji/wide glyphs) and avoid autowrap artifacts.
	w := m.width - 6
	if w < 1 {
		return 1
	}
	return w
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.startupActive {
		return m.renderStartupView()
	}

	renderW := m.renderWidth()
	m.views.Channels.SetPulseFrame(m.pulseFrame)

	// Header.
	header := m.views.Header.Render(renderW)

	// Summary.
	summary := m.views.Summary.Render(renderW)

	// Main area.
	mainH := m.mainHeightForSummary(summary)
	if mainH < 3 {
		mainH = 3
	}

	var mainContent string
	if m.inDetail {
		switch m.activeView {
		case views.CHANNELS:
			mainContent = m.views.Channel.Render(renderW, mainH)
		case views.TRANSACTIONS:
			mainContent = m.views.Transaction.Render(renderW, mainH)
		default:
			mainContent = m.renderActiveTable(renderW, mainH)
		}
	} else if m.menuOpen {
		menuWidth := 16
		if menuWidth >= renderW {
			menuWidth = renderW / 2
			if menuWidth < 1 {
				menuWidth = 1
			}
		}
		contentWidth := renderW - menuWidth
		if contentWidth < 1 {
			contentWidth = 1
		}
		menuStr := m.views.Menu.Render(menuWidth, mainH)
		contentStr := m.renderTable(m.currentTableView(), contentWidth, mainH)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, menuStr, contentStr)
	} else {
		mainContent = m.renderTable(m.activeView, renderW, mainH)
	}

	result := header + "\n\n" + summary + "\n" + mainContent
	lines := strings.Split(result, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], renderW, "")
		vis := lipgloss.Width(lines[i])
		if vis < renderW {
			lines[i] += strings.Repeat(" ", renderW-vis)
		}
	}

	// Clamp to terminal height to prevent scrolling past the header.
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	// Keep a stable full-screen frame so stale lines from previous renders
	// cannot remain visible when layout shape changes.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", renderW))
	}

	return strings.Join(lines, "\n")
}

func (m *model) shouldAnimate() bool {
	if m.inDetail {
		return false
	}
	return m.currentTableView() == views.CHANNELS && m.views.Channels.HasAnimatedAlerts()
}

func (m *model) ensurePulseTick() tea.Cmd {
	if m.pulseActive || !m.shouldAnimate() {
		return nil
	}
	m.pulseActive = true
	return pulseTickCmd()
}

func (m *model) loadInfoCmd() tea.Cmd {
	if m.infoLoading {
		return nil
	}
	m.infoLoading = true
	return loadInfoCmd(m.app.Network)
}

func (m *model) loadWalletBalanceCmd() tea.Cmd {
	if m.walletBalanceLoading {
		return nil
	}
	m.walletBalanceLoading = true
	return loadWalletBalanceCmd(m.app.Network)
}

func (m *model) loadChannelsBalanceCmd() tea.Cmd {
	if m.channelsBalanceLoading {
		return nil
	}
	m.channelsBalanceLoading = true
	return loadChannelsBalanceCmd(m.app.Network)
}

func (m *model) loadTransactionsCmd() tea.Cmd {
	if m.transactionsLoading {
		return nil
	}
	m.transactionsLoading = true
	return loadTransactionsCmd(m.app.Network)
}

func (m *model) loadForwardingHistoryCmd() tea.Cmd {
	if m.forwardingHistLoading {
		return nil
	}
	m.forwardingHistLoading = true
	return loadForwardingHistoryCmd(m.app.Network, m.models.FwdingHist.StartTime, m.models.FwdingHist.MaxNumEvents)
}

func (m *model) loadReceivedCmd() tea.Cmd {
	if m.receivedLoading {
		return nil
	}
	m.receivedLoading = true
	return loadReceivedCmd(m.app.Network)
}

func (m *model) loadChannelsCmd() tea.Cmd {
	if m.channelsLoading {
		return nil
	}
	var blockHeight uint32
	if m.models.Info != nil && m.models.Info.Info != nil {
		blockHeight = m.models.Info.BlockHeight
	}
	m.channelsLoading = true
	return loadChannelsCmd(m.app.Network, m.logger, blockHeight, m.channelSnapshot())
}

func (m *model) loadCurrentNodeCmd(pubkey string) tea.Cmd {
	if pubkey == "" || m.currentNodeLoading {
		return nil
	}
	m.currentNodeLoading = true
	return loadCurrentNodeCmd(m.app.Network, pubkey)
}

func (m *model) channelSnapshot() map[string]channelSnapshot {
	snapshot := make(map[string]channelSnapshot, m.models.Channels.Len())
	for _, ch := range m.models.Channels.List() {
		snapshot[ch.ChannelPoint] = channelSnapshot{
			updatesCount:    ch.UpdatesCount,
			hasLastUpdate:   ch.LastUpdate != nil,
			hasLocalPolicy:  ch.LocalPolicy != nil,
			hasRemotePolicy: ch.RemotePolicy != nil,
		}
	}
	return snapshot
}

func (m *model) refreshChannelAgesIfNeeded() tea.Cmd {
	if m.channelsLoading || m.models.Info == nil || m.models.Info.Info == nil {
		return nil
	}
	for _, ch := range m.models.Channels.List() {
		if ch.ID > 0 && ch.Age == 0 {
			return m.loadChannelsCmd()
		}
	}
	return nil
}

func (m *model) startInitialLoad() {
	m.startupActive = true
	m.startupFinishing = false
	m.startupTasks = make(map[string]bool, len(startupTaskLabels))
	for _, task := range startupTaskLabels {
		m.startupTasks[task.key] = true
	}
}

func (m *model) finishStartupTask(key string) {
	if !m.startupActive || m.startupTasks == nil {
		return
	}
	delete(m.startupTasks, key)
	if len(m.startupTasks) > 0 {
		m.startupWaiting = ""
	}
	if len(m.startupTasks) == 0 && !m.hasStartupLoadsInFlight() {
		m.startupFinishing = true
	}
}

func (m *model) hasStartupLoadsInFlight() bool {
	return m.infoLoading ||
		m.walletBalanceLoading ||
		m.channelsBalanceLoading ||
		m.transactionsLoading ||
		m.forwardingHistLoading ||
		m.channelsLoading ||
		m.receivedLoading
}

func (m *model) completeStartupCmdIfReady() tea.Cmd {
	if !m.startupFinishing || m.hasStartupLoadsInFlight() || len(m.startupTasks) != 0 {
		return nil
	}
	return startupCompleteCmd()
}

func (m *model) startupRetryTaskCmd(task string) tea.Cmd {
	switch task {
	case "info":
		return m.loadInfoCmd()
	case "wallet":
		return m.loadWalletBalanceCmd()
	case "channels_balance":
		return m.loadChannelsBalanceCmd()
	case "transactions":
		return m.loadTransactionsCmd()
	case "forwarding":
		return m.loadForwardingHistoryCmd()
	case "received":
		return m.loadReceivedCmd()
	case "channels":
		return m.loadChannelsCmd()
	default:
		return nil
	}
}

func (m *model) renderStartupView() string {
	renderW := m.renderWidth()
	progressW := renderW / 2
	if progressW < 16 {
		progressW = 16
	}
	if progressW > 40 {
		progressW = 40
	}
	titleStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#120c2c")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true)
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a78bfa")).
		Bold(true)
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6366f1"))
	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	total := len(startupTaskLabels)
	remaining := len(m.startupTasks)
	completed := total - remaining
	filled := 0
	if total > 0 {
		filled = progressW * completed / total
	}

	var bar strings.Builder
	for i := 0; i < progressW; i++ {
		if i < filled {
			bar.WriteString(doneStyle.Render("\u2588"))
		} else {
			bar.WriteString(pendingStyle.Render("\u2591"))
		}
	}

	waiting := "Finalizing"
	if m.startupWaiting != "" {
		waiting = m.startupWaiting
	} else {
		for _, task := range startupTaskLabels {
			if m.startupTasks[task.key] {
				waiting = task.label
				break
			}
		}
	}

	var body []string
	body = append(body, titleStyle.Align(lipgloss.Center).Width(renderW).Render(" Starting lntop "))
	body = append(body, "")
	body = append(body, lipgloss.NewStyle().Align(lipgloss.Center).Width(renderW).Render(sectionStyle.Render(" Initial Load ")))
	body = append(body, lipgloss.NewStyle().Align(lipgloss.Center).Width(renderW).Render(
		fmt.Sprintf("%s %d/%d", labelStyle.Render("Completed:"), completed, total),
	))
	body = append(body, lipgloss.NewStyle().Align(lipgloss.Center).Width(renderW).Render(
		fmt.Sprintf("%s %s", labelStyle.Render("Waiting for:"), waiting),
	))
	body = append(body, lipgloss.NewStyle().Align(lipgloss.Center).Width(renderW).Render(fmt.Sprintf("[%s]", bar.String())))

	for _, task := range startupTaskLabels {
		status := pendingStyle.Render("\u25cb")
		if !m.startupTasks[task.key] {
			status = doneStyle.Render("\u25cf")
		}
		body = append(body, lipgloss.NewStyle().Align(lipgloss.Center).Width(renderW).Render(
			fmt.Sprintf("%s %s", status, task.label),
		))
	}

	topPad := 0
	if m.height > len(body) {
		topPad = (m.height - len(body)) / 2
	}
	lines := make([]string, 0, m.height)
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, body...)
	for len(lines) < m.height {
		lines = append(lines, "")
	}

	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], renderW, "")
		vis := lipgloss.Width(lines[i])
		if vis < renderW {
			lines[i] += strings.Repeat(" ", renderW-vis)
		}
	}

	return strings.Join(lines, "\n")
}

func (m *model) currentTableView() string {
	if m.menuOpen {
		if preview := m.views.Menu.Current(); preview != "" {
			return preview
		}
	}
	return m.activeView
}

func (m *model) renderActiveTable(width, height int) string {
	return m.renderTable(m.activeView, width, height)
}

func (m *model) renderTable(viewName string, width, height int) string {
	switch viewName {
	case views.CHANNELS:
		return m.views.Channels.Render(width, height)
	case views.TRANSACTIONS:
		return m.views.Transactions.Render(width, height)
	case views.ROUTING:
		return m.views.Routing.Render(width, height)
	case views.FWDINGHIST:
		return m.views.FwdingHist.Render(width, height)
	case views.RECEIVED:
		return m.views.Received.Render(width, height)
	}
	return ""
}
