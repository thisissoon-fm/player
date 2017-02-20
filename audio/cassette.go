package audio

import (
	"encoding/binary"
	"io"
	"sync"

	"player/logger"
)

// A cassette takes an audio input source and writes it to
// an audio output source
type Cassette struct {
	// Audo input
	input io.Reader
	// Audio output
	output Writer
	// Orchestration channels
	stopC   chan bool // Stop reading the source
	resumeC chan bool // Resume reading the source
	endC    chan bool // bool sent when finished
	// Close orchestration
	closeC  chan bool
	closeWg *sync.WaitGroup
}

// Reads the cassette audo source input and writes it to the
// audo output writer
func (c *Cassette) play() {
	logger.Debug("playing cassette")
	defer logger.Debug("stopped cassette")
	defer c.closeWg.Done()
	defer func(c *Cassette) {
		logger.Debug("cassette finished")
		c.endC <- true
	}(c)
	for {
		select {
		case <-c.stopC:
			select {
			case <-c.closeC:
				return
			case <-c.resumeC:
				continue
			}
		case <-c.closeC:
			return
		default:
			frames := make([]int16, FRAMES_PER_BUFFER)
			if err := binary.Read(c.input, binary.LittleEndian, frames); err != nil {
				switch err {
				case io.ErrShortBuffer:
					continue // Wait for the buffer to fill
				case io.EOF, io.ErrUnexpectedEOF:
					return // We have completed reading the reader
				default:
					logger.WithError(err).Error("unexpected cassette read error")
					return
				}
			}
			if _, err := c.output.Write(frames); err != nil {
				logger.WithError(err).Error("unexpected cassette write error")
				return
			}
		}
	}
}

// Starts the cassette play coroutine
func (c *Cassette) Play() {
	defer logger.Debug("play cassette")
	c.closeWg.Add(1)
	go c.play()
}

// Stop the cassette - stops reading the cassette audio source so no more
// data is written to the cassette writer
func (c *Cassette) Stop() {
	defer logger.Debug("stop cassette")
	c.stopC <- true
}

// Resume the cassette - resumes reading the cassette audio source to
// the cassette writer
func (c *Cassette) Resume() {
	logger.Debug("resume cassette")
	c.resumeC <- true
}

// Returns the End of the cassette
func (c *Cassette) End() <-chan bool {
	return (<-chan bool)(c.endC)
}

// Stops playing a cassette midway through playback
func (c *Cassette) Eject() {
	defer logger.Debug("cassette ejected")
	close(c.closeC)
	c.closeWg.Wait()
}

// Create a new cassette
func NewCassette(i io.Reader, o Writer) *Cassette {
	return &Cassette{
		// I/O
		input:  i,
		output: o,
		// Orchestration Channels
		stopC:   make(chan bool, 1),
		resumeC: make(chan bool, 1),
		endC:    make(chan bool, 1),
		// Close orchestration
		closeC:  make(chan bool, 1),
		closeWg: &sync.WaitGroup{},
	}
}
