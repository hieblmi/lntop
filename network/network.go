package network

import (
	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network/backend"
	"github.com/hieblmi/lntop/network/backend/lnd"
	"github.com/hieblmi/lntop/network/backend/mock"
)

type Network struct {
	backend.Backend
}

func New(c *config.Network, logger logging.Logger) (*Network, error) {
	var (
		err error
		b   backend.Backend
	)
	if c.Type == "mock" {
		b = mock.New(c)
	} else {
		b, err = lnd.New(c, logger.With(logging.String("network", "lnd")))
		if err != nil {
			return nil, err
		}
	}

	err = b.Ping()
	if err != nil {
		return nil, err
	}

	return &Network{b}, nil
}
