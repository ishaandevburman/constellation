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
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	replay := flag.Bool("replay", false, "Replay historical events from stream before live tail")
	tail := flag.Bool("tail", true, "Tail live events (used with --replay to do both)")
	flag.Parse()

	if err := service.Run(context.Background(), "observer", func(ctx context.Context) error {
		return run(ctx, *replay, *tail)
	}); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, replay, tail bool) error {
	nc, err := natsx.Connect(natsx.DefaultConfig("observer"))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer nc.Close()

	str, err := nc.EnsureStream(ctx, natsx.StreamConfig{
		Name:     "events",
		Subjects: []string{subjects.Prefix + ".event.>"},
		NoAck:    true,
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	fmt.Println("─── observer watching constellation.event.> ───")

	if replay {
		if err := replayHistory(ctx, nc, str); err != nil {
			return fmt.Errorf("replay: %w", err)
		}
		if !tail {
			return nil
		}
		fmt.Println("─── caught up, tailing live ───")
	}

	sub, err := nc.Conn.Subscribe(subjects.Prefix+".event.>", func(msg *nats.Msg) {
		render("live", msg.Subject, msg.Data, time.Now())
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}

func replayHistory(ctx context.Context, nc *natsx.Client, str jetstream.Stream) error {
	info, err := str.Info(ctx)
	if err != nil {
		return err
	}
	lastSeq := info.State.LastSeq
	if lastSeq == 0 {
		fmt.Println("  (no events in stream)")
		return nil
	}

	for seq := uint64(1); seq <= lastSeq; seq++ {
		m, err := str.GetMsg(ctx, seq)
		if err != nil {
			continue
		}
		render("replay", m.Subject, m.Data, m.Time)
	}
	return nil
}

func render(kind, subject string, data []byte, ts time.Time) {
	e, err := event.Unmarshal(data)
	if err != nil {
		return
	}

	var prompt, result string
	if e.Data != nil {
		var m map[string]string
		if json.Unmarshal(e.Data, &m) == nil {
			prompt = m["prompt"]
			result = m["result"]
		}
	}

	label := fmt.Sprintf("%-7s", string(e.Type))
	timeStr := ts.Format("15:04:05")
	shortID := e.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	var detail string
	switch {
	case result != "":
		detail = result
	case prompt != "":
		detail = prompt
	default:
		detail = string(e.Data)
	}

	if len(detail) > 120 {
		detail = detail[:120] + "..."
	}
	detail = strings.ReplaceAll(detail, "\n", " ")

	fmt.Printf("[%s] %s | %s %s\n", label, timeStr, e.Source, detail)
	if e.CorrelationID != "" {
		fmt.Printf("         └─ corr=%s id=%s\n", e.CorrelationID[:8], shortID)
	}
}
