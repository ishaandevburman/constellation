package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ishaan/constellation/internal/subjects"
	"github.com/ishaan/constellation/pkg/event"
	"github.com/ishaan/constellation/pkg/natsx"
	"github.com/ishaan/constellation/pkg/service"
	"github.com/nats-io/nats.go"
)

type stepResult struct {
	agent  string
	data   []byte
	event  *event.Event
	result string
}

func main() {
	requestSubj := flag.String("request-subject", subjects.EventSubject("request"), "Subject to listen for requests")
	agents := flag.String("agents", "worker,critic", "Comma-separated agent chain (e.g. worker,critic)")
	agentTimeout := flag.Duration("agent-timeout", 30*time.Second, "Timeout per agent request")
	maxRetries := flag.Int("max-retries", 2, "Max retries per agent on failure")
	flag.Parse()

	chain := strings.Split(*agents, ",")
	for i := range chain {
		chain[i] = strings.TrimSpace(chain[i])
	}
	if len(chain) == 0 {
		log.Fatal("at least one agent required")
	}

	if err := service.Run(context.Background(), "orchestrator", func(ctx context.Context) error {
		return run(ctx, *requestSubj, chain, *agentTimeout, *maxRetries)
	}); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, requestSubj string, chain []string, agentTimeout time.Duration, maxRetries int) error {
	nc, err := natsx.Connect(natsx.DefaultConfig("orchestrator"))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer nc.Close()

	_, err = nc.EnsureStream(ctx, natsx.StreamConfig{
		Name:     "events",
		Subjects: []string{subjects.Prefix + ".event.>"},
		NoAck:    true,
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	log.Printf("orchestrator listening on %s → chain: %s", requestSubj, strings.Join(chain, " → "))

	sub, err := nc.Conn.Subscribe(requestSubj, func(msg *nats.Msg) {
		req, err := event.Unmarshal(msg.Data)
		if err != nil {
			log.Printf("unmarshal request: %v", err)
			return
		}

		log.Printf("received request %s from %s", req.ID, req.Source)

		currentData := req.Data
		var results []stepResult

		for _, agent := range chain {
			agentSubj := subjects.EventSourceSubject("task", agent)
			log.Printf("forwarding to %s on %s (retries=%d)", agent, agentSubj, maxRetries)

			var resp *nats.Msg
			var attempt int
			for attempt = 0; attempt <= maxRetries; attempt++ {
				task := event.New(event.TypeTask, "orchestrator")
				task.CorrelationID = req.ID
				task.Data = currentData
				payload, _ := task.Marshal()

				resp, err = nc.Conn.Request(agentSubj, payload, agentTimeout)
				if err == nil {
					break
				}
				log.Printf("  %s error (attempt %d/%d): %v", agent, attempt+1, maxRetries+1, err)
				if attempt < maxRetries {
					time.Sleep(time.Duration(1+attempt) * time.Second)
				}
			}
			if err != nil {
				log.Printf("  %s failed after %d attempts", agent, attempt+1)
				sendError(nc, msg.Reply, req.CorrelationID, fmt.Sprintf("%s failed after %d attempts: %v", agent, attempt+1, err))
				return
			}

			log.Printf("  %s responded (%d bytes)", agent, len(resp.Data))

			agentEvent, err := event.Unmarshal(resp.Data)
			if err != nil {
				log.Printf("  unmarshal %s response: %v", agent, err)
				sendError(nc, msg.Reply, req.CorrelationID, fmt.Sprintf("unmarshal %s: %v", agent, err))
				return
			}

			sr := stepResult{agent: agent, data: resp.Data, event: agentEvent}
			if agentEvent.Data != nil {
				var m map[string]string
				if json.Unmarshal(agentEvent.Data, &m) == nil {
					sr.result = m["result"]
				}
			}
			results = append(results, sr)

			nextPrompt, _ := json.Marshal(map[string]string{"prompt": sr.result})
			currentData = nextPrompt
		}

		lastResult := ""
		if len(results) > 0 {
			lastResult = results[len(results)-1].result
		}

		chainSummary := make([]map[string]string, len(results))
		for i, r := range results {
			chainSummary[i] = map[string]string{
				"agent":  r.agent,
				"result": r.result,
			}
		}
		summaryData, _ := json.Marshal(chainSummary)

		finalData, _ := json.Marshal(map[string]string{
			"result": lastResult,
		})

		finalResp := event.New(event.TypeResponse, "orchestrator")
		finalResp.CorrelationID = req.CorrelationID
		finalResp.Metadata = map[string]string{
			"chain": string(summaryData),
		}
		finalResp.Data = finalData

		payload, _ := finalResp.Marshal()
		nc.Conn.Publish(subjects.EventSubject(string(event.TypeResponse)), payload)

		if msg.Reply != "" {
			nc.Conn.Publish(msg.Reply, payload)
		}

		log.Printf("completed request %s (chain: %s)", req.ID, strings.Join(chain, " → "))
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}

func sendError(nc *natsx.Client, reply, correlationID, msg string) {
	errEvent := event.New(event.TypeError, "orchestrator")
	errEvent.CorrelationID = correlationID
	errEvent.Data, _ = json.Marshal(map[string]string{"error": msg})
	payload, _ := errEvent.Marshal()
	if reply != "" {
		nc.Conn.Publish(reply, payload)
	}
}
