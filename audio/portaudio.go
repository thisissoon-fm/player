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
	// Portaudio
	params portaudio.StreamParameters
	// Orchestration
	wg      *sync.WaitGroup
	resumeC chan bool
	errorC  chan error
	stopC   chan bool
	doneC   chan bool
	closeC  chan bool
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
	frames := make([]int16, FRAMES_PER_BUFFER)
	logger.Debug("open portaudio stream")
	stream, err := portaudio.OpenStream(pa.params, &frames)
	if err != nil {
		pa.errorC <- err
		return
	}
	defer stream.Close()
	logger.Debug("start portstart stream")
	if err := stream.Start(); err != nil {
		pa.errorC <- err
		return
	}
	defer stream.Stop()
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
			if err := stream.Write(); err != nil {
				logger.WithError(err).Warn("stream write error")
			}
		}
	}
}

// Close the port audio stream
func (pa *PortAudio) Close() error {
	close(pa.closeC)
	pa.wg.Wait()
	portaudio.Terminate()
	return nil
}

// Construct a new port audio streamer
func New() (*PortAudio, error) {
	portaudio.Initialize()
	host, err := portaudio.DefaultHostApi()
	if err != nil {
		return nil, err
	}
	device := host.DefaultOutputDevice
	params := portaudio.HighLatencyParameters(nil, device)
	params.Output.Channels = CHANNELS
	params.SampleRate = float64(SAMPLE_RATE)
	params.FramesPerBuffer = FRAMES_PER_BUFFER
	pa := &PortAudio{
		params:  params,
		resumeC: make(chan bool, 1),
		stopC:   make(chan bool, 1),
		errorC:  make(chan error, 1),
		closeC:  make(chan bool, 1),
		doneC:   make(chan bool, 1),
		wg:      &sync.WaitGroup{},
	}
	return pa, nil
}
