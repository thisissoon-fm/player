package audio

const (
	CHANNELS          = 2
	SAMPLE_RATE       = 44100
	FRAMES_PER_BUFFER = 1048
	INPUT_BUFFER_SIZE = 1
)

type Writer interface {
	Write([]int16) (int, error)
}
