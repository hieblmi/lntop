package pubsub

import (
	"bytes"
	"context"
	"time"

	"github.com/hieblmi/lntop/events"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network"
	"github.com/hieblmi/lntop/network/models"
	"github.com/hieblmi/lntop/network/options"
)

type tickerFunc func(context.Context, logging.Logger, *network.Network, chan *events.Event)

func (p *PubSub) ticker(ctx context.Context, sub chan *events.Event, fn ...tickerFunc) {
	p.wg.Add(1)
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		defer p.wg.Done()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for i := range fn {
					fn[i](ctx, p.logger, p.network, sub)
				}
			}
		}
	}()
}

// withTickerInfo checks if general information did not changed changed in the ticker interval.
func withTickerInfo() tickerFunc {
	var old *models.Info
	return func(ctx context.Context, logger logging.Logger, net *network.Network, sub chan *events.Event) {
		info, err := net.Info(ctx)
		if err != nil {
			logger.Error("network info returned an error", logging.Error(err))
		}
		if old != nil && info != nil {
			if old.BlockHeight != info.BlockHeight {
				select {
				case sub <- events.New(events.BlockReceived):
				case <-ctx.Done():
					return
				}
			}

			if old.NumPeers != info.NumPeers {
				select {
				case sub <- events.New(events.PeerUpdated):
				case <-ctx.Done():
					return
				}
			}

			if old.NumPendingChannels < info.NumPendingChannels {
				select {
				case sub <- events.New(events.ChannelPending):
				case <-ctx.Done():
					return
				}
			}

			if old.NumActiveChannels < info.NumActiveChannels {
				select {
				case sub <- events.New(events.ChannelActive):
				case <-ctx.Done():
					return
				}
			}

			if old.NumInactiveChannels < info.NumInactiveChannels {
				select {
				case sub <- events.New(events.ChannelInactive):
				case <-ctx.Done():
					return
				}
			}
		}
		old = info
	}
}

// withTickerChannelsBalance checks if channels balance and pending balance
// changed in the ticker interval.
func withTickerChannelsBalance() tickerFunc {
	var old *models.ChannelsBalance
	return func(ctx context.Context, logger logging.Logger, net *network.Network, sub chan *events.Event) {
		channelsBalance, err := net.GetChannelsBalance(ctx)
		if err != nil {
			logger.Error("network channels balance returned an error", logging.Error(err))
		}
		if old != nil && channelsBalance != nil {
			if old.Balance != channelsBalance.Balance ||
				old.PendingOpenBalance != channelsBalance.PendingOpenBalance {
				select {
				case sub <- events.New(events.ChannelBalanceUpdated):
				case <-ctx.Done():
					return
				}
			}
		}
		old = channelsBalance
	}
}

// withTickerWalletBalance checks if wallet balances changed in the ticker interval.
func withTickerWalletBalance() tickerFunc {
	var old *models.WalletBalance
	return func(ctx context.Context, logger logging.Logger, net *network.Network, sub chan *events.Event) {
		walletBalance, err := net.GetWalletBalance(ctx)
		if err != nil {
			logger.Error("network wallet balance returned an error", logging.Error(err))
		}
		if old != nil && walletBalance != nil {
			if old.TotalBalance != walletBalance.TotalBalance ||
				old.ConfirmedBalance != walletBalance.ConfirmedBalance ||
				old.UnconfirmedBalance != walletBalance.UnconfirmedBalance ||
				old.LockedBalance != walletBalance.LockedBalance ||
				old.ReservedBalanceAnchorChan != walletBalance.ReservedBalanceAnchorChan {
				select {
				case sub <- events.New(events.WalletBalanceUpdated):
				case <-ctx.Done():
					return
				}
			}
		}
		old = walletBalance
	}
}

// withTickerChannels detects per-channel state changes that don't affect the
// aggregate channel balance, such as sent/received totals or HTLC counts.
func withTickerChannels() tickerFunc {
	var old []*models.Channel
	return func(ctx context.Context, logger logging.Logger, net *network.Network, sub chan *events.Event) {
		channels, err := net.ListChannels(ctx, options.WithChannelPending)
		if err != nil {
			logger.Error("network channels returned an error", logging.Error(err))
			return
		}
		if old != nil && channelsChanged(old, channels) {
			select {
			case sub <- events.NewWithData(events.ChannelsUpdated, channels):
			case <-ctx.Done():
				return
			}
		}
		old = cloneChannels(channels)
	}
}

func channelsChanged(old, current []*models.Channel) bool {
	if len(old) != len(current) {
		return true
	}

	oldIndex := make(map[string]*models.Channel, len(old))
	for _, channel := range old {
		oldIndex[channel.ChannelPoint] = channel
	}

	for _, channel := range current {
		prev, ok := oldIndex[channel.ChannelPoint]
		if !ok || channelStateChanged(prev, channel) {
			return true
		}
	}

	return false
}

func channelStateChanged(old, current *models.Channel) bool {
	if old == nil || current == nil {
		return old != current
	}

	if old.Status != current.Status ||
		old.LocalBalance != current.LocalBalance ||
		old.RemoteBalance != current.RemoteBalance ||
		old.UnsettledBalance != current.UnsettledBalance ||
		old.TotalAmountSent != current.TotalAmountSent ||
		old.TotalAmountReceived != current.TotalAmountReceived ||
		len(old.PendingHTLC) != len(current.PendingHTLC) {
		return true
	}

	for i := range current.PendingHTLC {
		oldHTLC := old.PendingHTLC[i]
		currentHTLC := current.PendingHTLC[i]
		if oldHTLC == nil || currentHTLC == nil {
			if oldHTLC != currentHTLC {
				return true
			}
			continue
		}

		if oldHTLC.Incoming != currentHTLC.Incoming ||
			oldHTLC.Amount != currentHTLC.Amount ||
			oldHTLC.ExpirationHeight != currentHTLC.ExpirationHeight ||
			!bytes.Equal(oldHTLC.Hashlock, currentHTLC.Hashlock) {
			return true
		}
	}

	return false
}

func cloneChannels(channels []*models.Channel) []*models.Channel {
	cloned := make([]*models.Channel, 0, len(channels))
	for _, channel := range channels {
		if channel == nil {
			cloned = append(cloned, nil)
			continue
		}

		copyChannel := *channel
		if len(channel.PendingHTLC) > 0 {
			copyChannel.PendingHTLC = make([]*models.HTLC, 0, len(channel.PendingHTLC))
			for _, htlc := range channel.PendingHTLC {
				if htlc == nil {
					copyChannel.PendingHTLC = append(copyChannel.PendingHTLC, nil)
					continue
				}

				copyHTLC := *htlc
				if len(htlc.Hashlock) > 0 {
					copyHTLC.Hashlock = append([]byte(nil), htlc.Hashlock...)
				}
				copyChannel.PendingHTLC = append(copyChannel.PendingHTLC, &copyHTLC)
			}
		}

		cloned = append(cloned, &copyChannel)
	}

	return cloned
}
