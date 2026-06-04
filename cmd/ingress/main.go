package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ishaan/constellation/internal/subjects"
	"github.com/ishaan/constellation/pkg/event"
	"github.com/ishaan/constellation/pkg/natsx"
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

	nc, err := natsx.Connect(natsx.DefaultConfig("ingress"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer nc.Close()

	ctx := context.Background()
	_, err = nc.EnsureStream(ctx, natsx.StreamConfig{
		Name:     "events",
		Subjects: []string{subjects.Prefix + ".event.>"},
		NoAck:    true,
	})
	if err != nil {
		log.Fatalf("ensure stream: %v", err)
	}

	e := event.New(event.TypeRequest, "cli")
	e.CorrelationID = e.ID
	e.Data, _ = json.Marshal(map[string]string{"prompt": p})

	payload, _ := e.Marshal()

	log.Printf("sending request on %s", *subj)
	resp, err := nc.Conn.Request(*subj, payload, *timeout)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	log.Printf("got response (%d bytes)", len(resp.Data))

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
