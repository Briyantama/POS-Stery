package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/pos-stery/pos-stery/services/_shared/config"
)

func NewClient(cfg config.NATSConfig) (nats.JetStreamContext, error) {
	nc, err := nats.Connect(
		cfg.URL,
		nats.UserInfo(cfg.User, cfg.Password),
		nats.Name("pos-stery"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*1e9), // 2s
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}

	return js, nil
}
