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
		case event := <-hub.eventsC:
			go hub.handleEvent(event)
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
func (hub *Hub) handleEvent(ce ClientEvent) {
	logger.Debug("handle event")
	hub.closeWg.Add(1)
	defer hub.closeWg.Done()
	switch ce.Event.Type {
	case PauseEvent:
		if err := hub.pausePlayer(ce); err != nil {
			logger.WithError(err).Error("error pausing player")
		}
	case PausedEvent:
		if err := hub.pausedPlayer(ce); err != nil {
			logger.WithError(err).Error("error sending paused event")
		}
	case ResumeEvent:
		if err := hub.resumePlayer(ce); err != nil {
			logger.WithError(err).Error("error resuming player")
		}
	case ResumedEvent:
		if err := hub.resumedPlayer(ce); err != nil {
			logger.WithError(err).Error("error sending resumed event")
		}
	case PlayEvent:
		if err := hub.playTrack(ce); err != nil {
			logger.WithError(err).Error("error playing track")
		}
	case PlayingEvent:
		if err := hub.playingTrack(ce); err != nil {
			logger.WithError(err).Error("error sending playing event")
		}
	case StopEvent:
		if err := hub.stopTrack(ce); err != nil {
			logger.WithError(err).Error("error stopping player")
		}
	case StoppedEvent:
		if err := hub.stoppedPlayer(ce); err != nil {
			logger.WithError(err).Error("error sending stopped event")
		}
	case ErrorEvent:
		if err := hub.eventError(ce); err != nil {
			logger.WithError(err).Error("unable to write error to client")
		}
	}
}

// Pause event pauses the player, if the player is playing and not paused
// the player.Pause method will return true if the player was paused and
// false if the player was not paused
func (hub *Hub) pausePlayer(ce ClientEvent) error {
	logger.Debug("handle pause event")
	if player.Pause() {
		// The player was paused place a new paused event
		// onto the events channel
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    PausedEvent,
				Created: time.Now().UTC(),
			},
		}
	} else {
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

// Triggered by the pause handler on a successful pause, this needs
// to be broadcast to all clients so UI's can update their state
func (hub *Hub) pausedPlayer(ce ClientEvent) error {
	logger.Debug("handle paused event")
	if err := hub.Broadcast(ce.Event); err != nil {
		payload, err := json.Marshal(&ErrorPayload{
			Error: err.Error(),
		})
		if err != nil {
			return err
		}
		body, err := json.Marshal(Event{
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

// Resume event resumes the player, if the player is playing and paused
// the player.Resume method will return true if the player was resumed and
// false if the player was not resumed
func (hub *Hub) resumePlayer(ce ClientEvent) error {
	logger.Debug("handle resume event")
	if player.Resume() {
		// The player was resumed place a new resumed event
		// onto the events channel
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    ResumedEvent,
				Created: time.Now().UTC(),
			},
		}
	} else {
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

// Triggered by the resume handler on a successful resume, this needs
// to be broadcast to all clients so UI's can update their state
func (hub *Hub) resumedPlayer(ce ClientEvent) error {
	logger.Debug("handle resumed event")
	if err := hub.Broadcast(ce.Event); err != nil {
		payload, err := json.Marshal(&ErrorPayload{
			Error: err.Error(),
		})
		if err != nil {
			return err
		}
		body, err := json.Marshal(Event{
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

// The play event triggers the player to play a track from a provider
// This may error if the player is already playing a track, we cannpt
// decode the event payload or there was an error retreiving the
// player stream. On success a playing event will be published.
func (hub *Hub) playTrack(ce ClientEvent) error {
	logger.Debug("handle play event")
	payload := &PlayPayload{}
	if err := json.Unmarshal(ce.Event.Payload, payload); err != nil {
		return err
	}
	// player.Play is a blocking method
	if err := player.Play(payload.ProviderID, payload.TrackID); err != nil {
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
	} else {
		// The play is now playing, dispatch the playing event
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    PlayingEvent,
				Created: time.Now().UTC(),
				Payload: ce.Event.Payload, // We can just send the same payload back
			},
		}
	}
	return nil
}

// Triggered by the play event handler, this handler broadcasts a playing event
// once the player starts playing a track
func (hub *Hub) playingTrack(ce ClientEvent) error {
	logger.Debug("handle playing event")
	if err := hub.Broadcast(ce.Event); err != nil {
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

// The stop event will trigger the player to stop playing a track
// The player Stop method will retrun true if the stop event has
// been triggered on the player, else it will return false
func (hub *Hub) stopTrack(ce ClientEvent) error {
	logger.Debug("handle stop event")
	if player.Stop() {
		// The player was stopped place a new stopped event
		// onto the events channel
		hub.eventsC <- ClientEvent{
			Client: ce.Client,
			Event: Event{
				Type:    StoppedEvent,
				Created: time.Now().UTC(),
			},
		}
	} else {
		// If the player is not playing origional client
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

// Triggered by the stop handler on a successful stop, this needs
// to be broadcast to all clients so UI's can update their state
func (hub *Hub) stoppedPlayer(ce ClientEvent) error {
	logger.Debug("handle stopped event")
	if err := hub.Broadcast(ce.Event); err != nil {
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
