package event

import "io"

type Event interface {
	RawMessage() []byte
	ResponseWriter() io.Writer
}

type Producer interface {
	Events() <-chan Event
}
