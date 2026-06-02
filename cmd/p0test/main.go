package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ishaan/constellation/pkg/event"
	"github.com/nats-io/nats.go"
)

func main() {
	prompt := flag.String("prompt", "Say hello in one word", "Prompt to send")
	subj := flag.String("subject", "constellation.event.request", "Subject to publish on")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout for response")
	flag.Parse()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	e := event.New(event.TypeRequest, "p0-test")
	e.CorrelationID = e.ID
	e.Data, _ = json.Marshal(map[string]string{"prompt": *prompt})

	payload, _ := e.Marshal()

	resp, err := nc.Request(*subj, payload, *timeout)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	respEvent, err := event.Unmarshal(resp.Data)
	if err != nil {
		log.Fatalf("unmarshal response: %v", err)
	}

	var result map[string]string
	if respEvent.Data != nil {
		json.Unmarshal(respEvent.Data, &result)
	}

	fmt.Printf("Response ID: %s\n", respEvent.ID)
	fmt.Printf("CorrelationID: %s\n", respEvent.CorrelationID)
	fmt.Printf("Result: %s\n", result["result"])
}
