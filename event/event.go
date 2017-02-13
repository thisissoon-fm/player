package event

import (
	"encoding/json"
	"time"
)

const (
	PlayEvent   string = "play"
	StopEvent   string = "stop"
	PauseEvent  string = "pause"
	ResumeEvent string = "resume"
)

type Reader interface {
	Read() ([]byte, error)
}

type Writer interface {
	Write(b []byte) (int, error)
}

type Closer interface {
	Close() error
}

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

type Event struct {
	Type    string          `json:"type"`
	Created time.Time       `json:"created"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type PlayPayload struct {
	Provider string `json:"provider"` // The provider name
	TrackID  string `json:"trackID"`  // The track id from the provider
}
