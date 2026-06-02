package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ishaan/constellation/internal/subjects"
	"github.com/ishaan/constellation/pkg/event"
	"github.com/ishaan/constellation/pkg/natsx"
	"github.com/ishaan/constellation/pkg/service"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	if err := service.Run(context.Background(), "consumer", run); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := natsx.DefaultConfig("consumer")
	client, err := natsx.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	stream, err := client.JS.Stream(ctx, "events")
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "consumer",
		FilterSubject: subjects.AllEvents,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		e, err := event.Unmarshal(msg.Data())
		if err != nil {
			log.Printf("unmarshal error: %v", err)
			msg.Nak()
			return
		}
		log.Printf("received [%s] %s from %s", e.Type, e.ID, e.Source)
		if err := msg.Ack(); err != nil {
			log.Printf("ack error: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	defer cc.Stop()

	log.Printf("consumer listening on %s", subjects.AllEvents)

	<-ctx.Done()
	return nil
}
