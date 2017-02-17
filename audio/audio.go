package audio

import "io"

const (
	CHANNELS          = 2
	SAMPLE_RATE       = 44100
	FRAMES_PER_BUFFER = 8 * 1024
)

type Streamer interface {
	Stream(io.Reader)
	Done() <-chan bool
	Error() <-chan error
	Stop()
	Resume()
	Close() error
}
