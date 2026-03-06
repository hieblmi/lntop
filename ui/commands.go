package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network"
	"github.com/hieblmi/lntop/network/options"
)

type channelSnapshot struct {
	updatesCount    uint64
	hasLastUpdate   bool
	hasLocalPolicy  bool
	hasRemotePolicy bool
}

func loadInfoCmd(net *network.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		info, err := net.Info(ctx)
		return infoLoadedMsg{info: info, err: err}
	}
}

func loadWalletBalanceCmd(net *network.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		balance, err := net.GetWalletBalance(ctx)
		return walletBalanceLoadedMsg{balance: balance, err: err}
	}
}

func loadChannelsBalanceCmd(net *network.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		balance, err := net.GetChannelsBalance(ctx)
		return channelsBalanceLoadedMsg{balance: balance, err: err}
	}
}

func loadTransactionsCmd(net *network.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		txs, err := net.GetTransactions(ctx)
		return transactionsLoadedMsg{transactions: txs, err: err}
	}
}

func loadForwardingHistoryCmd(net *network.Network, startTime string, max uint32) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		events, err := net.GetForwardingHistory(ctx, startTime, max)
		return forwardingHistoryLoadedMsg{startTime: startTime, events: events, err: err}
	}
}

func loadCurrentNodeCmd(net *network.Network, pubkey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		node, err := net.GetNode(ctx, pubkey, true)
		return currentNodeLoadedMsg{pubkey: pubkey, node: node, err: err}
	}
}

func loadReceivedCmd(net *network.Network) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		invoices, err := net.ListInvoices(ctx)
		return receivedLoadedMsg{invoices: invoices, err: err}
	}
}

func loadChannelsCmd(net *network.Network, logger logging.Logger, blockHeight uint32, snapshot map[string]channelSnapshot) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		channels, err := net.ListChannels(ctx, options.WithChannelPending)
		if err != nil {
			return channelsLoadedMsg{err: err}
		}

		for i := range channels {
			if channels[i].ID > 0 && blockHeight > 0 {
				channels[i].Age = blockHeight - uint32(channels[i].ID>>40)
			}

			snap, ok := snapshot[channels[i].ChannelPoint]
			needsInfo := !ok || snap.updatesCount < channels[i].UpdatesCount ||
				!snap.hasLastUpdate || !snap.hasLocalPolicy || !snap.hasRemotePolicy
			if !needsInfo {
				continue
			}

			if err := net.GetChannelInfo(ctx, channels[i]); err != nil {
				return channelsLoadedMsg{err: err}
			}

			if channels[i].Node == nil {
				node, err := net.GetNode(ctx, channels[i].RemotePubKey, false)
				if err != nil {
					logger.Debug("loadChannelsCmd: cannot find Node", logging.String("pubkey", channels[i].RemotePubKey))
				} else {
					channels[i].Node = node
				}
			}
		}

		return channelsLoadedMsg{channels: channels}
	}
}
