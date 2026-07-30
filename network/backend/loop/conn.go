package loop

import (
	"crypto/tls"
	"os"
	"strings"

	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"

	"github.com/hieblmi/lntop/config"
)

const (
	defaultMaxMsgRecvSize  = 200 * 1024 * 1024
	defaultMacaroonTimeout = 60
)

func newClientConn(c *config.Loop) (*grpc.ClientConn, error) {
	macBytes, err := os.ReadFile(c.Macaroon)
	if err != nil {
		return nil, errors.Wrap(err, "loop: read macaroon")
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(macBytes); err != nil {
		return nil, errors.WithStack(err)
	}

	timeout := c.MacaroonTimeOut
	if timeout <= 0 {
		timeout = defaultMacaroonTimeout
	}
	constraints := []macaroons.Constraint{
		macaroons.TimeoutConstraint(timeout),
	}
	constrained, err := macaroons.AddConstraints(mac, constraints...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	macCred, err := macaroons.NewMacaroonCredential(constrained)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var creds credentials.TransportCredentials
	if c.Cert != "" {
		creds, err = credentials.NewClientTLSFromFile(c.Cert, "")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		creds = credentials.NewTLS(&tls.Config{})
	}

	maxRecv := c.MaxMsgRecvSize
	if maxRecv <= 0 {
		maxRecv = defaultMaxMsgRecvSize
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(macCred),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecv)),
	}

	address := strings.TrimPrefix(c.Address, "//")
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return conn, nil
}
