// Port Audio Streaming

package audio

import (
	"encoding/binary"
	"io"
	"sync"

	"player/logger"

	"github.com/gordonklaus/portaudio"
)

// Port audio streamer
type PortAudio struct {
	// Audio Stream
	stream *Stream
	// Orchestration
	wg      *sync.WaitGroup
	resumeC chan bool
	errorC  chan error
	stopC   chan bool
	doneC   chan bool
	closeC  chan bool

	inputC chan []int16
	over   []int16
}

// Stop the stream
func (pa *PortAudio) Stop() {
	pa.stopC <- true
}

// Resume the stream
func (pa *PortAudio) Resume() {
	pa.resumeC <- true
}

// Returns a channel to watch for the stream finishing
func (pa *PortAudio) Done() <-chan bool {
	return (<-chan bool)(pa.doneC)
}

// Errors in the stream will be placed here
func (pa *PortAudio) Error() <-chan error {
	return (<-chan error)(pa.errorC)
}

// Streams an io.Reader to the port audio device stream
func (pa *PortAudio) Stream(r io.Reader) {
	pa.wg.Add(1)
	defer pa.wg.Done()
	for {
		select {
		case <-pa.closeC:
			logger.Debug("close stream")
			pa.doneC <- true
			return
		case <-pa.stopC:
			logger.Debug("stop stream")
			select {
			case <-pa.closeC:
				logger.Debug("close stream")
				pa.doneC <- true
				return
			case <-pa.resumeC:
				logger.Debug("resume stream")
				continue
			}
		default:
			frames := make([]int16, FRAMES_PER_BUFFER)
			if err := binary.Read(r, binary.LittleEndian, &frames); err != nil {
				switch err {
				case io.EOF, io.ErrUnexpectedEOF:
					pa.doneC <- true
					return
				case io.ErrShortBuffer:
					continue
				default:
					pa.errorC <- err
					return
				}
			}
			pa.stream.Push(frames)
		}
	}
}

// Close the port audio stream
func (pa *PortAudio) Close() error {
	close(pa.closeC)
	pa.wg.Wait()
	pa.stream.Close()
	portaudio.Terminate()
	return nil
}

// Construct a new port audio streamer
func New() (*PortAudio, error) {
	portaudio.Initialize()
	s := NewStream()
	stream, err := portaudio.OpenDefaultStream(
		0,
		CHANNELS,
		float64(SAMPLE_RATE),
		FRAMES_PER_BUFFER,
		s.Fetch)
	s.stream = stream
	if err != nil {
		return nil, err
	}
	s.stream.Start()
	pa := &PortAudio{
		stream:  s,
		resumeC: make(chan bool, 1),
		stopC:   make(chan bool, 1),
		errorC:  make(chan error, 1),
		closeC:  make(chan bool, 1),
		doneC:   make(chan bool, 1),
		wg:      &sync.WaitGroup{},
	}
	return pa, nil
}

type Stream struct {
	stream *portaudio.Stream
	inputC chan []int16
	over   []int16
}

func (s *Stream) Push(samples []int16) {
	s.inputC <- samples
}

func (s *Stream) Fetch(out []int16) {
	// Write previously saved samples.
	i := copy(out, s.over)
	s.over = s.over[i:]
	for i < len(out) {
		select {
		case d := <-s.inputC:
			n := copy(out[i:], d)
			if n < len(d) {
				// Save anything we didn't need this time.
				s.over = d[n:]
			}
			i += n
		default:
			z := make([]int16, len(out)-i)
			copy(out[i:], z)
			return
		}
	}
}

func (s *Stream) Start() error {
	return s.stream.Start()
}

func (s *Stream) Stop() error {
	return s.stream.Stop()
}

func (s *Stream) Close() error {
	if err := s.stream.Stop(); err != nil {
		return err
	}
	if err := s.stream.Close(); err != nil {
		return err
	}
	return nil
}

func NewStream() *Stream {
	return &Stream{
		inputC: make(chan []int16, 8),
	}
}
