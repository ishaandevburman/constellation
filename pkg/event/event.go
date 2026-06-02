package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeRequest  Type = "request"
	TypeResponse Type = "response"
	TypeTask     Type = "task"
	TypeError    Type = "error"
)

type Event struct {
	ID            string            `json:"id"`
	Type          Type              `json:"type"`
	Source        string            `json:"source"`
	Timestamp     time.Time         `json:"timestamp"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Data          json.RawMessage   `json:"data,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func New(typ Type, source string) *Event {
	return &Event{
		ID:        uuid.NewString(),
		Type:      typ,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}
}

func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func Unmarshal(data []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
