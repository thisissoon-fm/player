// Port Audio Streaming

// +build portaudio

package audio

import (
	"sync"

	"player/logger"

	"github.com/gordonklaus/portaudio"
)

// Audio output stream handler
var output *Output

// Initialises the audio device for streaming
func Open() error {
	logger.Debug("initialize portaudio")
	// Start portaudio
	if err := portaudio.Initialize(); err != nil {
		return err
	}
	// Get host api
	host, err := portaudio.DefaultHostApi()
	if err != nil {
		return err
	}
	device := host.DefaultOutputDevice
	logger.WithFields(logger.F{
		"name":       device.Name,
		"sampleRate": device.DefaultSampleRate,
	}).Debug("using portaudio device")
	// Setup output writer
	output = NewOutput(device, INPUT_BUFFER_SIZE)
	return output.Start() // Start the outout writter
}

// Closes the audio output and terminates portaudio
func Close() error {
	logger.Debug("close audio")
	defer logger.Debug("closed audio")
	if output != nil {
		if err := output.Close(); err != nil {
			return err
		}
	}
	return portaudio.Terminate()
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
	// Portaudio Stream
	device *portaudio.DeviceInfo
	stream *portaudio.Stream
	// Input bytes from audio source
	inputC   chan []int16
	leftover []int16
	// Close orchestration
	closeWg *sync.WaitGroup
	closeC  chan bool
}

// Opens and starts a portaudio stream
func (output *Output) Start() error {
	if output.stream == nil {
		logger.Debug("setup portaudio output stream")
		// Stream parameters
		params := portaudio.HighLatencyParameters(nil, output.device)
		params.Output.Channels = CHANNELS
		params.SampleRate = float64(SAMPLE_RATE)
		params.FramesPerBuffer = FRAMES_PER_BUFFER
		// Setup Stream
		stream, err := portaudio.OpenStream(params, output.write)
		if err != nil {
			return err
		}
		// Start the portaudio stream
		if err := stream.Start(); err != nil {
			if err := stream.Close(); err != nil {
				return err
			}
			return err
		}
		output.stream = stream
		return nil
	}
	return nil
}

// Writes output to the audio device
func (output *Output) write(out []int16) {
	// Write previously saved samples.
	i := copy(out, output.leftover)
	output.leftover = output.leftover[i:]
	for i < len(out) {
		select {
		case s := <-output.inputC:
			n := copy(out[i:], s)
			if n < len(s) {
				// Save anything we didn't need this time.
				output.leftover = s[n:]
			}
			i += n
		default:
			z := make([]int16, len(out)-i)
			copy(out[i:], z)
			return
		}
	}
}

// Push data onto our input channel queue
func (output *Output) Write(data []int16) (int, error) {
	output.inputC <- data
	return len(data), nil
}

// Stops the output stream writter and stops/closes the
// portaudio stream
func (output *Output) Close() error {
	logger.Debug("close audio output")
	defer logger.Debug("closed audio output")
	close(output.closeC)
	output.closeWg.Wait() // Wait for the writer to exit
	if output.stream != nil {
		if err := output.stream.Stop(); err != nil {
			return err
		}
		if err := output.stream.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Construct a new output handler
func NewOutput(device *portaudio.DeviceInfo, bufferSize int) *Output {
	return &Output{
		device:  device,
		inputC:  make(chan []int16, bufferSize),
		closeWg: &sync.WaitGroup{},
		closeC:  make(chan bool),
	}
}
