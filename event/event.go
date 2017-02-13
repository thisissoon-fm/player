package event

import (
	"encoding/json"
	"time"
)

const (
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
