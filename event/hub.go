package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"player/logger"
	"player/player"
)

// Global event hub
var hub *Hub

// Already playing a track error
var (
	ErrPlaying  = errors.New("player is playing")
	ErrPausng   = errors.New("cannot pause, not playing or already paused")
	ErrResuming = errors.New("cannot resume, not playing or not paused")
	ErrStopping = errors.New("cannot stop, not playing")
)

// Package initialiser
func init() {
	hub = New()
}

// JSON Decoder
type Decoder interface {
	Decode(v interface{}) error
}

// Client interface
type Client interface {
	ID() string
	Read() ([]byte, error)
	Write([]byte) (int, error)
	Close() error
}

// Type for holding a list of hub clients
type Clients map[string]Client

// Add client convenience method returning the client id
func (c Clients) Add(client Client) {
	c[client.ID()] = client
}

// Delete client convenience method
func (c Clients) Del(client Client) {
	delete(c, client.ID())
}

// Client events
type ClientEvent struct {
	Client Client
	Event  Event
}

// Event Hub
type Hub struct {
	// Exported Fields
	// Unexported Fields
	decoder     Decoder // JSON Decoder
	clientsLock *sync.Mutex
	clients     Clients
	eventsC     chan ClientEvent
	closeWg     *sync.WaitGroup
	closeC      chan bool
}

// Add clients to the hub
func Add(client Client) { hub.Add(client) }
func (hub *Hub) Add(client Client) {
	logger.Debug("add hub client")
	hub.clientsLock.Lock()
	hub.clients.Add(client)
	go hub.read(client)
	hub.clientsLock.Unlock()
	logger.Debug("added hub client")
}

// Delete client from Hub
func Del(client Client) { hub.Del(client) }
func (hub *Hub) Del(client Client) {
	logger.Debug("delete hub client")
	hub.clientsLock.Lock()
	hub.clients.Del(client)
	hub.clientsLock.Unlock()
	logger.Debug("deleted hub client")
}

// Process incoming events from clients
func ProcessEvents() { hub.ProcessEvents() }
func (hub *Hub) ProcessEvents() {
	logger.Debug("start process events")
	defer logger.Debug("exit process events")
	hub.closeWg.Add(1)
	defer hub.closeWg.Done()
	for {
		select {
		case <-hub.closeC:
			return
		case <-player.Playing(): // The player is playing
			go func() {
				hub.closeWg.Add(1)
				if err := hub.playingTrack(); err != nil {
					logger.WithError(err).Error("error handling playing event")
				}
				hub.closeWg.Done()
			}()
		case <-player.Stopped(): // The player has stopped
			go func() {
				hub.closeWg.Add(1)
				if err := hub.stoppedPlayer(); err != nil {
					logger.WithError(err).Error("error handling stopped event")
				}
				hub.closeWg.Done()
			}()
		case <-player.Paused(): // The player has paused
			go func() {
				hub.closeWg.Add(1)
				if err := hub.pausedPlayer(); err != nil {
					logger.WithError(err).Error("error handling paused event")
				}
				hub.closeWg.Done()
			}()
		case event := <-hub.eventsC: // Client events
			go func() {
				hub.closeWg.Add(1)
				if err := hub.handleEvent(event); err != nil {
					logger.WithError(err).Error("error handling event")
				}
				hub.closeWg.Done()
			}()
		}
	}
}

// Broadcast an event to all connected clients
func Broadcast(event Event) error { return hub.Broadcast(event) }
func (hub *Hub) Broadcast(event Event) error {
	body, err := json.Marshal(&event)
	if err != nil {
		return err
	}
	hub.clientsLock.Lock()
	for _, client := range hub.clients {
		if _, err := client.Write(body); err != nil {
			logger.WithError(err).Error("error writting to client")
		}
	}
	hub.clientsLock.Unlock()
	return nil
}

// Ready is fired when the player is ready to start playing tracks
func PlayerReady() error { return hub.PlayerReady() }
func (h *Hub) PlayerReady() error {
	event := Event{
		Type:    PlayerReadyEvent,
		Created: time.Now().UTC(),
	}
	if err := hub.Broadcast(event); err != nil {
		return err
	}
	return nil
}

// Ready is fired when the player is ready to start playing tracks
func Offline() error { return hub.Offline() }
func (h *Hub) Offline() error {
	event := Event{
		Type:    PlayerOfflineEvent,
		Created: time.Now().UTC(),
	}
	if err := hub.Broadcast(event); err != nil {
		return err
	}
	return nil
}

// Close the event hub, this will prevent any further events
// from being processed
func Close() error { return hub.Close() }
func (hub *Hub) Close() error {
	logger.Debug("close event hub")
	defer logger.Debug("closed event hub")
	close(hub.closeC)
	hub.closeWg.Wait() // Wait for coroutines to exit
	return nil
}

// Decode raw byte data into interface, defaults to json decoder
func (hub *Hub) decode(b []byte, v interface{}) error {
	decoder := hub.decoder
	if decoder == nil {
		decoder = json.NewDecoder(bytes.NewReader(b))
	}
	return decoder.Decode(v)
}

// Reads messages from an attached client, decoding the raw json event
// and placing it onto the events channel for processing
func (hub *Hub) read(client Client) error {
	logger.Debug("start client read loop")
	defer logger.Debug("exit client read loop")
	hub.closeWg.Add(1)
	defer hub.closeWg.Done()
	for { // Read from the client
		raw, err := client.Read() // Blocking
		if err != nil {
			if err != io.EOF {
				logger.WithError(err).Error("unexpected event hub client read error")
			}
			return nil // Exit on any error
		}
		logger.WithField("event", string(raw)).Debug("received raw event")
		// Decode the event
		event := Event{}
		if err := hub.decode(raw, &event); err != nil {
			logger.WithError(err).Error("error decoding event json")
			continue
		}
		// Add the event to the event channel wtith the origional client attached
		hub.eventsC <- ClientEvent{client, event}
	}
}

// Handles a specific event, some events will broadcast their responses
// actions to all attachec clients, other only reply to the client
// that origionated the event, this is the case for error responses
// so errors can be surfaced back to the clients
func (hub *Hub) handleEvent(ce ClientEvent) error {
	logger.Debug("handle event")
	switch ce.Event.Type {
	case PauseEvent:
		return hub.pausePlayer(ce)
	case ResumeEvent:
		return hub.resumePlayer(ce)
	case PlayEvent:
		return hub.playTrack(ce)
	case StopEvent:
		return hub.stopTrack(ce)
	case ErrorEvent:
		return hub.eventError(ce)
	}
	return nil
}

// Pause event pauses the player, if the player is playing and not paused
// the player.Pause method will return true if the player was paused and
// false if the player was not paused
func (hub *Hub) pausePlayer(ce ClientEvent) error {
	logger.Debug("handle pause event")
	if !player.Pause() {
		// If the player is not playing or is already paused
		// error back to the origional client
		payload, err := json.Marshal(&ErrorPayload{
			Error: ErrPausng.Error(),
		})
		if err != nil {
			return err
		}
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    ErrorEvent,
				Created: time.Now().UTC(),
				Payload: json.RawMessage(payload),
			},
		}
	}
	return nil
}

// Triggered by the player pause event
func (hub *Hub) pausedPlayer() error {
	logger.Debug("handle paused event")
	event := Event{
		Type:    PausedEvent,
		Created: time.Now().UTC(),
	}
	if err := hub.Broadcast(event); err != nil {
		return err
	}
	return nil
}

// Resume event resumes the player, if the player is playing and paused
// the player.Resume method will return true if the player was resumed and
// false if the player was not resumed
func (hub *Hub) resumePlayer(ce ClientEvent) error {
	logger.Debug("handle resume event")
	if !player.Resume() {
		// If the player is not playing or is not paused
		// error back to the origional client
		payload, err := json.Marshal(&ErrorPayload{
			Error: ErrResuming.Error(),
		})
		if err != nil {
			return err
		}
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    ErrorEvent,
				Created: time.Now().UTC(),
				Payload: json.RawMessage(payload),
			},
		}
	}
	return nil
}

// The play event triggers the player to play a track from a provider
// This may error if the player is already playing a track, we cannpt
// decode the event payload or there was an error retreiving the
// player stream.
func (hub *Hub) playTrack(ce ClientEvent) error {
	logger.Debug("handle play event")
	payload := &PlayPayload{}
	if err := json.Unmarshal(ce.Event.Payload, payload); err != nil {
		return err
	}
	err := player.Play(player.LoadTrackConfig{
		ProviderName:    payload.ProviderName,
		ProviderTrackID: payload.ProviderTrackID,
		PlaylistID:      payload.PlaylistID,
	})
	if err != nil {
		payload, err := json.Marshal(&ErrorPayload{
			Error: err.Error(),
		})
		if err != nil {
			return err
		}
		body, err := json.Marshal(&Event{
			Type:    ErrorEvent,
			Created: time.Now().UTC(),
			Payload: json.RawMessage(payload),
		})
		if err != nil {
			return err
		}
		if _, err := ce.Client.Write(body); err != nil {
			return err
		}
	}
	return nil
}

// Triggered by the player playing event
func (hub *Hub) playingTrack() error {
	logger.Debug("handle playing event")
	event := Event{
		Type:    PlayingEvent,
		Created: time.Now().UTC(),
	}
	if err := hub.Broadcast(event); err != nil {
		return err
	}
	return nil
}

// The stop event will trigger the player to stop playing a track
// The player Stop method will retrun true if the stop event has
// been triggered on the player, else it will return false
func (hub *Hub) stopTrack(ce ClientEvent) error {
	logger.Debug("handle stop event")
	if !player.Stop() {
		payload, err := json.Marshal(&ErrorPayload{
			Error: ErrStopping.Error(),
		})
		if err != nil {
			return err
		}
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    ErrorEvent,
				Created: time.Now().UTC(),
				Payload: json.RawMessage(payload),
			},
		}
	}
	return nil
}

// Triggered by a player stopped event
func (hub *Hub) stoppedPlayer() error {
	logger.Debug("handle player stopped event")
	event := Event{
		Type:    StoppedEvent,
		Created: time.Now().UTC(),
	}
	if err := hub.Broadcast(event); err != nil {
		return err
	}
	return nil
}

// Write an error event to the client
func (hub *Hub) eventError(ce ClientEvent) error {
	logger.Debug("handle error event")
	body, err := json.Marshal(&ce.Event)
	if err != nil {
		return err
	}
	if _, err := ce.Client.Write(body); err != nil {
		return err
	}
	return nil
}

// Hub Constructor
func New() *Hub {
	return &Hub{
		clientsLock: &sync.Mutex{},
		clients:     make(Clients),
		eventsC:     make(chan ClientEvent),
		closeWg:     &sync.WaitGroup{},
		closeC:      make(chan bool),
	}
}
