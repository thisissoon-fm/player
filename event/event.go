package event

import "io"

type Event interface {
	RawMessage() []byte
	ResponseWriter() io.Writer
}
