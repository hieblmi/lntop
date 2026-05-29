package network

import (
	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network/backend"
	"github.com/hieblmi/lntop/network/backend/lnd"
	"github.com/hieblmi/lntop/network/backend/loop"
	"github.com/hieblmi/lntop/network/backend/mock"
)

type Network struct {
	backend.Backend

	// Loop is set when the user enables [loop] in config. Nil otherwise.
	Loop *loop.Backend
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

	return &Network{Backend: b}, nil
}

// EnableLoop attaches a loopd backend to this Network. Safe to call with nil
// or disabled config — in either case Loop stays nil and the LOOP view is
// hidden by the UI layer.
func (n *Network) EnableLoop(cfg *config.Loop, logger logging.Logger) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	b, err := loop.New(cfg, logger.With(logging.String("network", "loop")))
	if err != nil {
		return err
	}
	n.Loop = b
	return nil
}
