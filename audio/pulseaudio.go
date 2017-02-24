// +build pulseaudio

package audio

import (
	"bytes"
	"encoding/binary"
	"player/logger"
	"sync"

	pulse "github.com/mesilliac/pulse-simple"
)

func paStream() (*pulse.Stream, error) {
	return pulse.Playback("SOON_ FM", "SOON_ FM", &pulse.SampleSpec{
		pulse.SAMPLE_S16LE,
		SAMPLE_RATE,
		CHANNELS,
	})
}

// Audio output stream handler
var output *Output

// Initialises the audio device for streaming
func Open() error {
	output = NewOutput(INPUT_BUFFER_SIZE)
	return output.Start() // Start the outout writter
}

// Closes the audio output and terminates portaudio
func Close() error {
	return output.Close()
}

// Returs the current audio output writer
func Get() (*Output, error) {
	if output == nil {
		return nil, ErrNoOutput
	}
	return output, nil
}

// Output handles writting to audio stream
type Output struct {
	stream *pulse.Stream
	// Input bytes from audio source
	inputC chan []int16
	// Close orchestration
	closeWg *sync.WaitGroup
	closeC  chan bool
}

// Opens and starts a portaudio stream
func (output *Output) Start() error {
	if output.stream == nil {
		logger.Debug("setup pulseaudio output stream")
		stream, err := paStream()
		if err != nil {
			return nil
		}
		output.stream = stream
		go output.write()
	}
	return nil
}

// Writes output to pulseaudio
func (output *Output) write() {
	output.closeWg.Add(1)
	defer output.closeWg.Done()
	for {
		select {
		case <-output.closeC:
			return
		case samples := <-output.inputC:
			buf := new(bytes.Buffer)
			for _, s := range samples {
				_ = binary.Write(buf, binary.LittleEndian, s)
			}
			if _, err := output.stream.Write(buf.Bytes()); err != nil {
				logger.WithError(err).Warn("pulse audio stream write error")

			}
		}
	}
}

// Push data onto our input channel queue
func (output *Output) Write(data []int16) (int, error) {
	output.inputC <- data
	return len(data), nil
}

// Close output
func (output *Output) Close() error {
	close(output.closeC)
	output.closeWg.Wait()
	if output.stream != nil {
		defer output.stream.Free()
		defer output.stream.Drain()
	}
	return nil
}

// Construct a new output handler
func NewOutput(bufferSize int) *Output {
	return &Output{
		inputC:  make(chan []int16, bufferSize),
		closeWg: &sync.WaitGroup{},
		closeC:  make(chan bool),
	}
}
