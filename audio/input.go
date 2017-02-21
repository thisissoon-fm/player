package audio

import (
	"encoding/binary"
	"io"
	"sync"

	"player/logger"
)

// A input takes an audio input source and writes it to
// an audio output source
type Input struct {
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

// Reads the input audo source input and writes it to the
// audo output writer
func (i *Input) play() {
	logger.Debug("playing audio input")
	defer logger.Debug("stopped audio input")
	defer i.closeWg.Done()
	defer func(i *Input) {
		logger.Debug("audio input complete")
		i.endC <- true
	}(i)
	for {
		select {
		case <-i.stopC:
			select {
			case <-i.closeC:
				return
			case <-i.resumeC:
				continue
			}
		case <-i.closeC:
			return
		default:
			frames := make([]int16, FRAMES_PER_BUFFER)
			if err := binary.Read(i.input, binary.LittleEndian, frames); err != nil {
				switch err {
				case io.ErrShortBuffer:
					continue // Wait for the buffer to fill
				case io.EOF, io.ErrUnexpectedEOF:
					return // We have completed reading the reader
				default:
					logger.WithError(err).Error("unexpected audio input read error")
					return
				}
			}
			if _, err := i.output.Write(frames); err != nil {
				logger.WithError(err).Error("unexpected audio input write error")
				return
			}
		}
	}
}

// Starts the audip input play coroutine
func (i *Input) Play() {
	defer logger.Debug("play audio input")
	i.closeWg.Add(1)
	go i.play()
}

// Stop the input - stops reading the input audio source so no more
// data is written to the input writer
func (i *Input) Stop() {
	defer logger.Debug("stop audio input")
	i.stopC <- true
}

// Resume the input - resumes reading the input audio source to
// the input writer
func (i *Input) Resume() {
	logger.Debug("resume audio input")
	i.resumeC <- true
}

// Returns the End of the input
func (i *Input) End() <-chan bool {
	return (<-chan bool)(i.endC)
}

// Stops playing a input midway through playback
func (i *Input) Close() {
	defer logger.Debug("input ")
	close(i.closeC)
	i.closeWg.Wait()
}

// Create a new input
func NewInput(i io.Reader, o Writer) *Input {
	return &Input{
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
