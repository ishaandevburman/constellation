package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/ishaan/constellation/internal/subjects"
	"github.com/ishaan/constellation/pkg/event"
	"github.com/ishaan/constellation/pkg/natsx"
	"github.com/ishaan/constellation/pkg/ollama"
	"github.com/ishaan/constellation/pkg/service"
	"github.com/nats-io/nats.go"
)

func main() {
	model := flag.String("model", "constellation-worker", "Ollama model name")
	subj := flag.String("subject", subjects.AllEvents, "NATS subject to subscribe to")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama API base URL")
	flag.Parse()

	if err := service.Run(context.Background(), *model, func(ctx context.Context) error {
		return run(ctx, *model, *subj, *ollamaURL)
	}); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, model, subj, ollamaURL string) error {
	nc, err := natsx.Connect(natsx.DefaultConfig("agent-" + model))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer nc.Close()

	oll := ollama.NewClient(ollamaURL)

	log.Printf("agent [%s] subscribing to %s", model, subj)

	sub, err := nc.Conn.Subscribe(subj, func(msg *nats.Msg) {
		e, err := event.Unmarshal(msg.Data)
		if err != nil {
			log.Printf("unmarshal error: %v", err)
			return
		}

		log.Printf("received [%s] %s", e.Type, e.ID)

		var prompt string
		if e.Data != nil {
			var data map[string]string
			if err := json.Unmarshal(e.Data, &data); err == nil {
				prompt = data["prompt"]
			}
		}
		if prompt == "" {
			prompt = string(e.Data)
		}

		result, err := oll.Generate(model, prompt)
		if err != nil {
			log.Printf("ollama error: %v", err)
			respErr := event.New(event.TypeError, model)
			respErr.CorrelationID = e.CorrelationID
			respErr.Data, _ = json.Marshal(map[string]string{"error": err.Error()})
			payload, _ := respErr.Marshal()
			if reply := msg.Reply; reply != "" {
				nc.Conn.Publish(reply, payload)
			}
			nc.Conn.Publish(subjects.EventSubject(string(event.TypeError)), payload)
			return
		}

		resp := event.New(event.TypeResponse, model)
		resp.CorrelationID = e.CorrelationID
		resp.Data, _ = json.Marshal(map[string]string{"result": result})

		payload, err := resp.Marshal()
		if err != nil {
			log.Printf("marshal error: %v", err)
			return
		}

		if reply := msg.Reply; reply != "" {
			nc.Conn.Publish(reply, payload)
		}
		nc.Conn.Publish(subjects.EventSubject(string(event.TypeResponse)), payload)

		log.Printf("published response %s", resp.ID)
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}
