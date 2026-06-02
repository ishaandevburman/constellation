package natsx

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

type Config struct {
	URL           string
	Name          string
	ReconnectWait time.Duration
	MaxReconnects int
}

func DefaultConfig(name string) Config {
	return Config{
		URL:           nats.DefaultURL,
		Name:          name,
		ReconnectWait: 2 * time.Second,
		MaxReconnects: 10,
	}
}

func Connect(cfg Config) (*Client, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.Name(cfg.Name),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("[%s] disconnected: %v", cfg.Name, err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[%s] reconnected to %s", cfg.Name, nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Printf("[%s] connection closed", cfg.Name)
		}),
	)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &Client{Conn: nc, JS: js}, nil
}

func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}
