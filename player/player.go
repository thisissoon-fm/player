package player

import (
	"errors"
	"io"

	"player/logger"

	"github.com/korandiz/mpa"
	pulse "github.com/mesilliac/pulse-simple"
)

var player *Player // Global Player

var ErrUnknownStreamer = errors.New("unknown streamer")

// Package initalisation
func init() {
	var err error
	player, err = New()
	if err != nil {
		logger.WithError(err).Fatal("failed to initialise player")
	}
}

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

// Adds a streamer to the global player streamers
func AddStreamer(s Streamer) {
	player.Streamers.Add(s)
}

// Remove a streamer from the global player steamer
func DelStreamer(s string) {
	player.Streamers.Del(s)
}

// Audio Player
type Player struct {
	// Exported Fields
	Streamers Streamers // Service Streamers (google etc)
	// Unexported Fields
	stream *pulse.Stream // Audio Stream
	paused bool          // Paused State
}

// Close the pulse audio stream
func Close() error { return player.Close() }
func (p *Player) Close() error {
	logger.Debug("close player")
	defer logger.Info("closed player")
	if p.stream != nil {
		p.stream.Drain()
		p.stream.Free()
	}
	return nil
}

// Set the pause state of the player
func Pause(b bool) { player.Pause(b) }
func (p *Player) Pause(b bool) {
	p.paused = b
}

// Returns the player paused state
func Paused() { player.Paused() }
func (p *Player) Paused() bool {
	return p.paused
}

// Play a track from a service
func Play(s, t string) error { return player.Play(s, t) }
func (p *Player) Play(s, t string) error {
	f := logger.F{"service": s, "track": t}
	logger.WithFields(f).Info("play track")
	defer logger.WithFields(f).Info("finished track")
	// Reset Pause State
	p.paused = false
	// Get the streamer
	streamer := p.Streamers.Get(s)
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
		if p.paused { // If paused, don't read the buffer
			continue
		}
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
		Streamers: streamers,
		stream:    stream,
	}
	return p, nil
}
