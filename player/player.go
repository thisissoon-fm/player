package player

import (
	"bufio"
	"errors"
	"io"

	"github.com/korandiz/mpa"
	pulse "github.com/mesilliac/pulse-simple"
)

// All streamers must implement this interface
type Streamer interface {
	Name() string
	Stream(track string) (io.ReadCloser, error)
}

// A store of Streamers
type Streamers map[string]Streamer

// Get a streamer by name
func (m Streamers) Get(name string) Streamer {
	s, ok := m[name]
	if !ok {
		return nil
	}
	return s
}

// Add streamer convenience method
func (m Streamers) Add(streamer Streamer) {
	m[streamer.Name()] = streamer
}

// Delete streamer convenience method
func (m Streamers) Del(name string) {
	delete(m, name)
}

type Player struct {
	// Streamers
	streamers Streamers
	// Audio Stream
	stream *pulse.Stream
}

func (p *Player) Close() error {
	if p.stream != nil {
		p.stream.Drain()
		p.stream.Free()
	}
	return nil
}

func (p *Player) Play(s string, t string) error {
	// Get the streamer
	streamer := p.streamers.Get(s)
	if streamer == nil {
		return errors.New("unknown streamer")
	}
	// Get track stream
	stream, err := streamer.Stream(t)
	if err != nil {
		return err
	}
	defer stream.Close() // When we done, close the track stream
	// Read buffer
	buffer := bufio.NewReader(stream)
	// Data buffer
	data := make([]byte, 1024*8)
	// MPEG Decoder
	decoder := &mpa.Reader{Decoder: &mpa.Decoder{Input: buffer}}
	// Steam track to audio stream
	for {
		i, err := decoder.Read(data)
		if err == io.EOF || i == 0 {
			return nil
		}
		i, err = p.stream.Write(data)
		if err != nil {
			return nil
		}
	}
}

// Consturcts a new Player with the given steamers
func New(s ...Streamer) (*Player, error) {
	spec := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	stream, err := pulse.Playback("sfm", "sfm", &spec)
	if err != nil {
		return nil, err
	}
	streamers := make(Streamers)
	for _, streamer := range s {
		streamers.Add(streamer)
	}
	p := &Player{
		streamers: streamers,
		stream:    stream,
	}
	return p, nil
}
