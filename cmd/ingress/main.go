package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ishaan/constellation/pkg/event"
	"github.com/nats-io/nats.go"
)

func main() {
	prompt := flag.String("prompt", "", "Prompt to send (reads from stdin if empty)")
	subj := flag.String("subject", "constellation.event.request", "Subject to publish on")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout for response")
	flag.Parse()

	p := *prompt
	if p == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		p = strings.TrimSpace(string(data))
	}
	if p == "" {
		log.Fatal("no prompt provided")
	}

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer nc.Close()

	e := event.New(event.TypeRequest, "cli")
	e.CorrelationID = e.ID
	e.Data, _ = json.Marshal(map[string]string{"prompt": p})

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

	fmt.Println(result["result"])

	if result["result"] == "" {
		os.Exit(1)
	}
}
