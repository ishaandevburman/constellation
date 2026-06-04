package natsx

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type StreamConfig struct {
	Name     string
	Subjects []string
	NoAck    bool
}

func (c *Client) EnsureStream(ctx context.Context, cfg StreamConfig) (jetstream.Stream, error) {
	s, err := c.JS.CreateStream(ctx, jetstream.StreamConfig{
		Name:     cfg.Name,
		Subjects: cfg.Subjects,
		Storage:  jetstream.FileStorage,
		NoAck:    cfg.NoAck,
	})
	if err == nil {
		return s, nil
	}

	s, err = c.JS.Stream(ctx, cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("ensure stream %s: %w", cfg.Name, err)
	}
	return s, nil
}
