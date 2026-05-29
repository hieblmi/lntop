package app

import (
	"github.com/hieblmi/lntop/config"
	"github.com/hieblmi/lntop/logging"
	"github.com/hieblmi/lntop/network"
)

type App struct {
	Config  *config.Config
	Logger  logging.Logger
	Network *network.Network
}

func New(cfg *config.Config) (*App, error) {
	logger, err := logging.New(cfg.Logger)
	if err != nil {
		return nil, err
	}

	net, err := network.New(&cfg.Network, logger)
	if err != nil {
		return nil, err
	}

	if err := net.EnableLoop(cfg.Loop, logger); err != nil {
		// A loopd misconfiguration should not block lntop startup —
		// log and continue with Loop disabled for this session.
		logger.Info("loop integration disabled due to error",
			logging.Error(err))
	}

	return &App{
		Config:  cfg,
		Logger:  logger,
		Network: net,
	}, nil
}
