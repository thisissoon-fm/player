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
	ErrPause           = errors.New("pause playing")
	ErrStop            = errors.New("stop playing")
	ErrClose           = errors.New("close player")
	ErrUnknownProvider = errors.New("unknown provider")
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
type Provider interface {
	Name() string
	Stream(track string) (io.ReadCloser, error)
}

// A store of providers
type Providers map[string]Provider

// Get a streamer by name
func (m Providers) Get(name string) Provider {
	s, ok := m[name]
	if !ok {
		return nil
	}
	return s
}

// Add streamer convenience method
func (m Providers) Add(p Provider) {
	m[p.Name()] = p
}

// Delete streamer convenience method
func (m Providers) Del(name string) {
	delete(m, name)
}

// Adds a streamer to the global player streamers
func AddProvider(p Provider) {
	player.Providers.Add(p)
}

// Remove a streamer from the global player steamer
func DelProvider(p string) {
	player.Providers.Del(p)
}

// Audio Player
type Player struct {
	// Exported Fields
	Providers Providers // Service Providers (google etc)
	// Unexported Fields
	audio     *pulse.Stream // Audio Stream
	pauseLock *sync.Mutex
	playing   bool
	paused    bool
	stopC     chan bool
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
func IsPaused() bool { return player.IsPaused() }
func (p *Player) IsPaused() bool {
	return p.paused
}

// Returns the player playing state
func IsPlaying() bool { return player.IsPlaying() }
func (p *Player) IsPlaying() bool {
	return p.playing
}

// Stops playing the current playing track if playing
func Stop() { player.Stop() }
func (p *Player) Stop() {
	if p.playing {
		p.stopC <- true
	}
}

// Play a track from a service
func Play(provider, id string) error { return player.Play(provider, id) }
func (p *Player) Play(provider, id string) error {
	f := logger.F{"provider": provider, "track": id}
	logger.WithFields(f).Info("play track")
	defer logger.WithFields(f).Info("finished track")
	// Set state
	p.playing = true
	defer func(p *Player) { p.playing = false }(p)
	p.paused = false
	defer func(p *Player) { p.paused = false }(p)
	// Get the streamer
	prvdr := p.Providers.Get(provider)
	if p == nil {
		return ErrUnknownProvider
	}
	// Get track stream
	stream, err := prvdr.Stream(id)
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
				case <-p.stopC:
					return nil
				case <-p.closeC:
					return nil
				}
			case io.ErrShortBuffer:
				continue // Wait for buffer to fill
			case io.EOF, ErrStop, ErrClose:
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
		case <-p.stopC:
			return ErrStop
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
func New() (*Player, error) {
	spec := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	audio, err := pulse.Playback("sfm", "sfm", &spec)
	if err != nil {
		return nil, err
	}
	player := &Player{
		Providers: make(Providers),
		audio:     audio,
		pauseLock: &sync.Mutex{},
		stopC:     make(chan bool),
		pauseC:    make(chan bool),
		resumeC:   make(chan bool),
		closeC:    make(chan bool),
	}
	return player, nil
}
