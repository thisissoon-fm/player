package player

import (
	"errors"
	"io"

	"player/logger"

	"github.com/korandiz/mpa"
	pulse "github.com/mesilliac/pulse-simple"
)

var ErrUnknownStreamer = errors.New("unknown streamer")

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

// Close the pulse audio stream
func (p *Player) Close() error {
	logger.Debug("close player")
	defer logger.Info("closed player")
	if p.stream != nil {
		p.stream.Drain()
		p.stream.Free()
	}
	return nil
}

// Play a track from a service
func (p *Player) Play(s string, t string) error {
	f := logger.F{"service": s, "track": t}
	logger.WithFields(f).Info("play track")
	defer logger.WithFields(f).Info("finished track")
	// Get the streamer
	streamer := p.streamers.Get(s)
	if streamer == nil {
		return ErrUnknownStreamer
	}
	// Get track stream
	stream, err := streamer.Stream(t)
	if err != nil {
		return err
	}
	defer stream.Close()
	// MPEG Decoder
	decoder := &mpa.Reader{Decoder: &mpa.Decoder{Input: stream}}
	for {
		data := make([]byte, 1024*8)
		if _, err := decoder.Read(data); err != nil {
			if err == io.ErrShortBuffer { // Wait for buffer
				continue
			}
			if err == io.EOF { // Done reading
				return nil
			}
		}
		if _, err = p.stream.Write(data); err != nil {
			return err
		}
	}
	return nil
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
