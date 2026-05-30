package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ishaan/constellation/internal/subjects"
	"github.com/ishaan/constellation/pkg/event"
	"github.com/ishaan/constellation/pkg/natsx"
	"github.com/ishaan/constellation/pkg/service"
)

func main() {
	if err := service.Run(context.Background(), "publisher", run); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := natsx.DefaultConfig("publisher")
	client, err := natsx.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	stream, err := client.EnsureStream(ctx, natsx.StreamConfig{
		Name:     "events",
		Subjects: []string{subjects.Prefix + ".event.>"},
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}
	log.Printf("ensured stream: %s", stream.CachedInfo().Config.Name)

	data, _ := json.Marshal(map[string]string{"message": "hello from publisher"})

	e := event.New(event.TypeRequest, "publisher")
	e.Data = data
	e.Metadata["env"] = "development"

	payload, err := e.Marshal()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	subj := subjects.EventSubject(string(e.Type))
	if _, err := client.JS.Publish(ctx, subj, payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	log.Printf("published event %s to %s", e.ID, subj)

	<-ctx.Done()
	return nil
}
