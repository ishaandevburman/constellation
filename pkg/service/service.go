package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Run(ctx context.Context, name string, start func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- start(ctx)
	}()

	select {
	case sig := <-sigCh:
		log.Printf("[%s] received signal: %v", name, sig)
		cancel()
		return <-errCh
	case err := <-errCh:
		return err
	}
}
