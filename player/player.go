package player

import (
	"errors"
	"io"
	"sync"

	"player/audio"
	"player/logger"
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
	player = New()
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

// Configuration to pass to player play method
type LoadTrackConfig struct {
	ProviderName    string
	ProviderTrackID string
	PlaylistID      string
}

// Audio Player
type Player struct {
	Providers Providers // Service Providers (google etc)
	// Tracks
	tracksLock *sync.Mutex
	Tracks     Tracks // Tracks loaded into the player
	// Pausing
	paused    bool
	pauseLock *sync.Mutex
	pauseC    chan bool
	pausedC   chan bool
	resumeC   chan bool
	// Playing
	playing  bool
	playingC chan bool
	// Stopped
	stopC    chan bool
	stoppedC chan bool
	// Close orchestration
	playWg *sync.WaitGroup
	closeC chan bool
}

// Close the pulse audio stream
func Close() error { return player.Close() }
func (p *Player) Close() error {
	logger.Debug("close player")
	defer logger.Info("closed player")
	close(p.closeC) // Close the close channel
	p.playWg.Wait() // Wait for play routines to exit
	return nil
}

// Pause the player
func Pause() bool { return player.Pause() }
func (p *Player) Pause() bool {
	if !p.paused && p.playing {
		logger.Debug("player not paused and playing, resume")
		p.pauseC <- true
		return true
	}
	return false
}

func Paused() <-chan bool { return player.Paused() }
func (p *Player) Paused() <-chan bool {
	return (<-chan bool)(p.pausedC)
}

// Resume the player
func Resume() bool { return player.Resume() }
func (p *Player) Resume() bool {
	if p.paused && p.playing {
		logger.Debug("player paused and playing, resume")
		p.resumeC <- true
		return true
	}
	return false
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

func Stopped() <-chan bool { return player.Stopped() }
func (p *Player) Stopped() <-chan bool {
	return (<-chan bool)(p.stoppedC)
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

// Load a track into the player
func LoadTrack(c LoadTrackConfig) (*Track, error) { return player.LoadTrack(c) }
func (p *Player) LoadTrack(c LoadTrackConfig) (*Track, error) {
	var track *Track
	track = p.Tracks.Get(c.PlaylistID)
	if track == nil {
		provider := p.Providers.Get(c.ProviderName)
		if p == nil {
			return nil, ErrUnknownProvider
		}
		track = NewTrack(c.PlaylistID, c.ProviderTrackID, provider)
		if err := track.Load(); err != nil {
			return nil, err
		}
		// Add track to player loaded tracks
		p.tracksLock.Lock()
		p.Tracks.Add(track)
		p.tracksLock.Unlock()
	}
	return track, nil
}

// Play a track from a provider
func Play(c LoadTrackConfig) error { return player.Play(c) }
func (p *Player) Play(c LoadTrackConfig) error {
	// Are we playing, if we are then we can't play something else ;)
	if p.playing {
		return ErrPlaying
	}
	// Load the track
	track, err := p.LoadTrack(c)
	if err != nil {
		return err
	}
	// Fire play goroutine
	go p.play(track)
	// Fire playing signal
	p.playingC <- true
	// Remove the track from the track store
	p.tracksLock.Lock()
	p.Tracks.Del(c.PlaylistID)
	p.tracksLock.Unlock()
	return nil
}

// Send playing signal
func Playing() <-chan bool { return player.Playing() }
func (p *Player) Playing() <-chan bool {
	return (<-chan bool)(p.playingC)
}

// Plays a track, handling pause / resume / stop events
func (p *Player) play(track io.ReadCloser) error {
	logger.Debug("start track playback")
	defer logger.Debug("exit track playback")
	// Close orchestration
	p.playWg.Add(1)
	defer p.playWg.Done()
	// Send stopped event
	defer func(p *Player) { p.stoppedC <- true }(p)
	// Set state
	p.playing = true
	defer func(p *Player) { p.playing = false }(p) // Reset player playing statr
	defer func(p *Player) { p.paused = false }(p)  // Reset player pause state
	defer track.Close()                            // Close the track
	// Get audio output
	output, err := audio.Get()
	if err != nil {
		return err
	}
	// Load cassette
	input := audio.NewInput(track, output)
	go input.Play() // Start playing the input
	defer input.Close()
	for {
		select {
		case <-input.End():
			return nil
		case <-p.pauseC:
			p.paused = true
			p.pausedC <- true
			input.Stop()
		case <-p.resumeC:
			p.paused = false
			p.playingC <- true
			input.Resume()
		case <-p.stopC:
			return nil
		case <-p.closeC:
			logger.Debug("close player")
			return nil
		}
	}
	return nil
}

// Consturcts a new Player with the given steamers
func New() *Player {
	player := &Player{
		// Providers
		Providers: make(Providers),
		// Tracks
		tracksLock: &sync.Mutex{},
		Tracks:     make(Tracks),
		// Orchestration channels
		pauseLock: &sync.Mutex{},
		stopC:     make(chan bool, 1),
		stoppedC:  make(chan bool, 1),
		pauseC:    make(chan bool, 1),
		pausedC:   make(chan bool, 1),
		resumeC:   make(chan bool, 1),
		playingC:  make(chan bool, 1),
		playWg:    &sync.WaitGroup{},
		closeC:    make(chan bool, 1),
	}
	return player
}
