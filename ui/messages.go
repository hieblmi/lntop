package ui

import (
	"github.com/hieblmi/lntop/events"
	netmodels "github.com/hieblmi/lntop/network/models"
)

// eventMsg wraps an LND event for delivery to the bubbletea update loop.
type eventMsg struct {
	event *events.Event
}

type pulseTickMsg struct{}

type infoLoadedMsg struct {
	info *netmodels.Info
	err  error
}

type walletBalanceLoadedMsg struct {
	balance *netmodels.WalletBalance
	err     error
}

type channelsBalanceLoadedMsg struct {
	balance *netmodels.ChannelsBalance
	err     error
}

type transactionsLoadedMsg struct {
	transactions []*netmodels.Transaction
	err          error
}

type forwardingHistoryLoadedMsg struct {
	events []*netmodels.ForwardingEvent
	err    error
}

type channelsLoadedMsg struct {
	channels []*netmodels.Channel
	err      error
}

type currentNodeLoadedMsg struct {
	pubkey string
	node   *netmodels.Node
	err    error
}
