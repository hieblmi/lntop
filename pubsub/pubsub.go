package pubsub

import (
	"context"
	"sync"
	"time"

	"github.com/hieblmi/lntop/events"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network"
	"github.com/hieblmi/lntop/network/models"
)

type PubSub struct {
	stop    chan struct{}
	logger  logging.Logger
	network *network.Network
	wg      *sync.WaitGroup
}

func New(logger logging.Logger, network *network.Network) *PubSub {
	return &PubSub{
		logger:  logger.With(logging.String("logger", "pubsub")),
		network: network,
		wg:      &sync.WaitGroup{},
		stop:    make(chan struct{}),
	}
}

func (p *PubSub) invoices(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	invoices := make(chan *models.Invoice)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for invoice := range invoices {
			p.logger.Debug(
				"receive invoice",
				logging.Object("invoice", invoice),
			)

			var event *events.Event
			if invoice.Settled {
				event = events.NewWithData(events.InvoiceSettled, invoice)
			} else {
				event = events.NewWithData(events.InvoiceCreated, invoice)
			}
			select {
			case sub <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeInvoice(ctx, invoices)
		if err != nil {
			p.logger.Error("SubscribeInvoice returned an error", logging.Error(err))
		}
		close(invoices)
		cancel()
	}()
}

func (p *PubSub) transactions(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	transactions := make(chan *models.Transaction)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for tx := range transactions {
			p.logger.Debug("receive transaction", logging.String("tx_hash", tx.TxHash))
			select {
			case sub <- events.New(events.TransactionCreated):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeTransactions(ctx, transactions)
		if err != nil {
			p.logger.Error("SubscribeTransactions returned an error", logging.Error(err))
		}
		close(transactions)
		cancel()
	}()
}

func (p *PubSub) routingUpdates(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	routingUpdates := make(chan *models.RoutingEvent)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for hu := range routingUpdates {
			p.logger.Debug("receive htlcUpdate")
			if !hu.IsEmpty() {
				select {
				case sub <- events.NewWithData(events.RoutingEventUpdated, hu):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeRoutingEvents(ctx, routingUpdates)
		if err != nil {
			p.logger.Error("SubscribeRoutingEvents returned an error", logging.Error(err))
		}
		close(routingUpdates)
		cancel()
	}()
}

func (p *PubSub) graphUpdates(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	graphUpdates := make(chan *models.ChannelEdgeUpdate)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for gu := range graphUpdates {
			p.logger.Debug("receive graph update")
			select {
			case sub <- events.NewWithData(events.GraphUpdated, gu):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeGraphEvents(ctx, graphUpdates)
		if err != nil {
			p.logger.Error("SubscribeGraphEvents returned an error", logging.Error(err))
		}
		close(graphUpdates)
		cancel()
	}()
}

// loopSwaps streams swap state changes from loopd's Monitor RPC into the
// shared event channel as LoopSwapUpdated. Runs only when network.Loop is
// set; reconnects with a 5-second backoff if the stream dies, until the
// context is canceled.
func (p *PubSub) loopSwaps(ctx context.Context, sub chan *events.Event) {
	if p.network.Loop == nil {
		return
	}
	p.wg.Add(2)
	swaps := make(chan *models.LoopSwap)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for swap := range swaps {
			p.logger.Debug("receive loop swap update",
				logging.String("id", swap.ID))
			select {
			case sub <- events.NewWithData(events.LoopSwapUpdated, swap):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		defer close(swaps)
		for {
			err := p.network.Loop.SubscribeSwaps(ctx, swaps)
			if err != nil {
				p.logger.Info("loop: monitor stream ended",
					logging.Error(err))
			}
			select {
			case <-ctx.Done():
				cancel()
				return
			case <-time.After(5 * time.Second):
				// Retry the subscription.
			}
		}
	}()
}

func (p *PubSub) channels(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	channels := make(chan *models.ChannelUpdate)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for range channels {
			p.logger.Debug("channels updated")
			select {
			case sub <- events.New(events.ChannelActive):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeChannels(ctx, channels)
		if err != nil {
			p.logger.Error("SubscribeChannels returned an error", logging.Error(err))
		}
		close(channels)
		cancel()
	}()
}

func (p *PubSub) Stop() {
	close(p.stop)
	p.logger.Debug("Received signal, gracefully stopping")
}

func (p *PubSub) Run(ctx context.Context, sub chan *events.Event) {
	p.logger.Debug("Starting...")

	// Create a cancellable context that all subscriptions share.
	// When Stop() closes p.stop, we cancel this context so all
	// network subscriptions return, which then close their data
	// channels and let consumer goroutines exit cleanly.
	ctx, cancel := context.WithCancel(ctx)

	p.invoices(ctx, sub)
	p.transactions(ctx, sub)
	p.routingUpdates(ctx, sub)
	p.channels(ctx, sub)
	p.graphUpdates(ctx, sub)
	p.loopSwaps(ctx, sub)
	tickerFns := []tickerFunc{
		withTickerInfo(),
		withTickerWalletBalance(),
		withTickerChannelsBalance(),
		withTickerChannels(),
	}
	if p.network.Loop != nil {
		tickerFns = append(tickerFns, withTickerLoopTick())
	}
	p.ticker(ctx, sub, tickerFns...)

	<-p.stop
	cancel()
	p.wg.Wait()
}
