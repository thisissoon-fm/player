package player

import (
	"errors"
	"io"
	"sync"

	"player/logger"

	pulse "github.com/mesilliac/pulse-simple"
)

var player *Player // Global Player

var (
	ErrPlaying         = errors.New("cannot play, player is currently playing")
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
	playWg    *sync.WaitGroup
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
	p.playWg.Wait() // Wait for play routines to exit
	return nil
}

// Pause the player
func Pause() bool { return player.Pause() }
func (p *Player) Pause() bool {
	p.pauseLock.Lock()
	defer p.pauseLock.Unlock()
	if !p.paused && p.playing {
		p.paused = true
		p.pauseC <- true
		return true
	}
	return false
}

// Resume the player
func Resume() bool { return player.Resume() }
func (p *Player) Resume() bool {
	p.pauseLock.Lock()
	defer p.pauseLock.Unlock()
	if p.paused && p.playing {
		p.paused = false
		p.resumeC <- true
		return true
	}
	return false
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
func Stop() bool { return player.Stop() }
func (p *Player) Stop() bool {
	if p.playing {
		p.stopC <- true
		p.playWg.Wait() // Wait for play routines to exit before returning
		return true
	}
	return false
}

// Play a track from a service
func Play(provider, id string) error { return player.Play(provider, id) }
func (p *Player) Play(provider, id string) error {
	logger.WithFields(logger.F{
		"provider": provider,
		"track":    id,
	}).Info("play track")
	if p.playing {
		return ErrPlaying
	}
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
	go p.play(stream) // Fire play goroutine
	return nil
}

// Plays a track, handling pause / resume / stop events
func (p *Player) play(stream io.ReadCloser) error {
	logger.Debug("playing stream")
	defer logger.Debug("stopped playing stream")
	// Close orchestration
	p.playWg.Add(1)
	defer p.playWg.Done()
	// Set state
	p.playing = true
	p.paused = false
	defer func(p *Player) { p.playing = false }(p)
	defer func(p *Player) { p.paused = false }(p)
	// Close the stream
	defer stream.Close()
	for { // Stream the file
		if err := p.stream(stream); err != nil {
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
		playWg:    &sync.WaitGroup{},
		closeC:    make(chan bool),
	}
	return player, nil
}
