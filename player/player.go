package player

import (
	"errors"
	"io"
	"sync"

	"player/logger"

	"github.com/korandiz/mpa"
	pulse "github.com/mesilliac/pulse-simple"
)

var player *Player // Global Player

var (
	ErrPause           = errors.New("paused")
	ErrClose           = errors.New("closing")
	ErrUnknownStreamer = errors.New("unknown streamer")
)

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
	audio     *pulse.Stream // Audio Stream
	pauseLock *sync.Mutex
	paused    bool
	pauseC    chan bool
	resumeC   chan bool
	closeC    chan bool
}

// Close the pulse audio stream
func Close() error { return player.Close() }
func (p *Player) Close() error {
	logger.Debug("close player")
	defer logger.Info("closed player")
	close(p.closeC) // Close the close channel
	if p.stream != nil {
		p.audio.Drain()
		p.audio.Free()
	}
	return nil
}

// Pause the player
func Pause() { player.Pause() }
func (p *Player) Pause() {
	p.pauseLock.Lock()
	if !p.IsPaused() {
		p.paused = true
		p.pauseC <- true
	}
	p.pauseLock.Unlock()
}

// Resume the player
func Resume() { player.Resume() }
func (p *Player) Resume() {
	p.pauseLock.Lock()
	if p.IsPaused() {
		p.paused = false
		p.resumeC <- true
	}
	p.pauseLock.Unlock()
}

// Returns the player paused state
func IsPaused() { player.IsPaused() }
func (p *Player) IsPaused() bool {
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
	for { // Stream the file
		if err := p.stream(decoder); err != nil {
			switch err {
			case ErrPause: // If paused, wait for resume or close
				select {
				case <-p.resumeC:
					continue
				case <-p.closeC:
					return nil
				}
			case io.ErrShortBuffer:
				continue // Wait for buffer to fill
			case io.EOF, ErrClose:
				return nil // We are done :)
			}
		}
	}
	return nil
}

// Stream routine, takes a reader
func (p *Player) stream(r io.Reader) error {
	data := make([]byte, 1024*8)
	for {
		select {
		case <-p.pauseC:
			return ErrPause
		case <-p.closeC:
			return ErrClose
		default:
			if _, err := r.Read(data); err != nil {
				return err
			}
			if _, err := p.audio.Write(data); err != nil {
				return err
			}
		}
	}
}

// Consturcts a new Player with the given steamers
func New(s ...Streamer) (*Player, error) {
	spec := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	audio, err := pulse.Playback("sfm", "sfm", &spec)
	if err != nil {
		return nil, err
	}
	streamers := make(Streamers)
	for _, streamer := range s {
		streamers.Add(streamer)
	}
	p := &Player{
		Streamers: streamers,
		audio:     audio,
		pauseLock: &sync.Mutex{},
		pauseC:    make(chan bool),
		resumeC:   make(chan bool),
		closeC:    make(chan bool),
	}
	return p, nil
}
