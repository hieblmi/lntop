package pubsub

import (
	"context"
	"sync"

	"github.com/edouardparis/lntop/events"
	"github.com/edouardparis/lntop/logging"
	"github.com/edouardparis/lntop/network"
	"github.com/edouardparis/lntop/network/models"
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

			if invoice.Settled {
				sub <- events.NewWithData(
					events.InvoiceSettled, invoice,
				)
			} else {
				sub <- events.NewWithData(
					events.InvoiceCreated, invoice,
				)
			}
		}
	}()

	go func() {
		defer p.wg.Done()
		err := p.network.SubscribeInvoice(ctx, invoices)
		if err != nil {
			p.logger.Error("SubscribeInvoice returned an error", logging.Error(err))
		}
		// Close the data channel after the network subscription ends,
		// so the consumer goroutine drains remaining items and exits.
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
			sub <- events.New(events.TransactionCreated)
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
				sub <- events.NewWithData(events.RoutingEventUpdated, hu)
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
			sub <- events.NewWithData(events.GraphUpdated, gu)
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

func (p *PubSub) channels(ctx context.Context, sub chan *events.Event) {
	p.wg.Add(2)
	channels := make(chan *models.ChannelUpdate)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer p.wg.Done()
		for range channels {
			p.logger.Debug("channels updated")
			sub <- events.New(events.ChannelActive)
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
	p.ticker(ctx, sub,
		withTickerInfo(),
		withTickerChannelsBalance(),
	)

	<-p.stop
	cancel()
	p.wg.Wait()
}
