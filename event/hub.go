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

	"github.com/rs/xid"
)

// Global hub
var hub *Hub

// Already playing a track
var ErrPlaying = errors.New("player is playing")

// Initialise package with a global hub
func init() {
	hub = NewHub()
}

// Event decoder interface
type Decoder interface {
	Decode(v interface{}) error
}

// Type for holding a list of hub clients
type Clients map[string]ReadWriteCloser

// Get a client by id
func (c Clients) Get(id string) ReadWriteCloser {
	s, ok := c[id]
	if !ok {
		return nil
	}
	return s
}

// Add client convenience method returning the client id
func (c Clients) Add(id string, rwc ReadWriteCloser) string {
	c[id] = rwc
	return id
}

// Delete client convenience method
func (c Clients) Del(id string) {
	delete(c, id)
}

type Hub struct {
	// Exported Fields
	// Unexported Fields
	clientsLock *sync.Mutex     // Client lock
	clients     Clients         // Event clients
	decoder     Decoder         // Event Decoder
	eventsC     chan []byte     // Event processor channel
	closeWg     *sync.WaitGroup // Wait for internal coroutines to exit
	closeC      chan bool       // Closes internal coroutines
}

// Goroutine for reading client events
func (h *Hub) read(id string, rwc ReadWriteCloser) {
	logger.Debug("start event hub client read")
	defer logger.Debug("exit event hub client read")
	h.closeWg.Add(1)
	defer h.closeWg.Done()
	defer h.DelClient(id) // Remove the client from the hub
	for {
		b, err := rwc.Read()
		if err != nil {
			if err != io.EOF {
				logger.WithError(err).Error("unexpected hub read error")
			}
			return // Exit on any error
		}
		h.eventsC <- b
	}
}

// Add a client to the hub and start reading from it
// returns the client id
func AddClient(rwc ReadWriteCloser) string { return hub.AddClient(rwc) }
func (h *Hub) AddClient(rwc ReadWriteCloser) string {
	id := xid.New().String()
	logger.Debug("lock event hub clients")
	h.clientsLock.Lock()
	h.clients.Add(id, rwc)
	logger.Debug("unlock event hub clients")
	h.clientsLock.Unlock()
	go h.read(id, rwc)
	return id
}

// Remove a client from the hub
func DelClient(id string) { hub.DelClient(id) }
func (h *Hub) DelClient(id string) {
	logger.Debug("lock event hub clients")
	h.clientsLock.Lock()
	h.clients.Del(id)
	logger.Debug("unlock event hub clients")
	h.clientsLock.Unlock()
}

// Broadcast event to all connected clients
func Broadcast(b []byte) { hub.Broadcast(b) }
func (h *Hub) Broadcast(b []byte) {
	logger.Debug("lock event hub clients")
	h.clientsLock.Lock()
	for _, client := range h.clients {
		logger.Debug("write to client")
		if _, err := client.Write(b); err != nil {
			logger.WithError(err).Error("failed to write to client")
		}
	}
	logger.Debug("unlock event hub clients")
	h.clientsLock.Unlock()
}

// Goroutine to process events from clients
func ProcessEvents() { hub.ProcessEvents() }
func (h *Hub) ProcessEvents() {
	logger.Debug("start event hub processor")
	defer logger.Debug("exit event hub processor")
	h.closeWg.Add(1)
	defer h.closeWg.Done()
	for {
		select {
		case b := <-h.eventsC:
			go func() {
				h.closeWg.Add(1)
				defer h.closeWg.Done()
				if err := h.handle(b); err != nil {
					logger.WithFields(logger.F{
						"event": string(b),
					}).WithError(err).Error("failed to handle event")
				}
			}()
		case <-h.closeC:
			return
		}
	}
}

// Decode raw byte data into interface, defaults to json decoder
func (h *Hub) decode(b []byte, v interface{}) error {
	decoder := h.decoder
	if decoder == nil {
		decoder = json.NewDecoder(bytes.NewReader(b))
	}
	return decoder.Decode(v)
}

// Handles a received event
func (h *Hub) handle(b []byte) error {
	logger.WithField("event", string(b)).Debug("handle event")
	defer logger.WithField("event", string(b)).Debug("handled event")
	event := &Event{}
	if err := h.decode(b, event); err != nil {
		return err
	}
	var err error
	switch event.Type {
	case PauseEvent:
		err = h.pause(event)
	case ResumeEvent:
		err = h.resume(event)
	case PlayEvent:
		err = h.play(event)
	case StopEvent:
		err = h.stop(event)
	}
	return err
}

// Pause event handler
func (h *Hub) pause(event *Event) error {
	logger.Debug("pause player")
	if player.Pause() {
		defer logger.Debug("player paused")
		body, err := json.Marshal(&Event{
			Type:    PausedEvent,
			Created: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		go h.Broadcast(body)
	}
	return nil
}

// Resume event handler
func (h *Hub) resume(event *Event) error {
	logger.Debug("resume player")
	if player.Resume() {
		defer logger.Debug("resumed paused")
		body, err := json.Marshal(&Event{
			Type:    ResumedEvent,
			Created: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		go h.Broadcast(body)
	}
	return nil
}

// Play event handler
func (h *Hub) play(event *Event) error {
	if player.IsPlaying() {
		return ErrPlaying
	}
	payload := &PlayPayload{}
	if err := json.Unmarshal(event.Payload, payload); err != nil {
		return err
	}
	if err := player.Play(payload.ProviderID, payload.TrackID); err != nil {
		return err
	}
	return nil
}

// Stop event handler
func (h *Hub) stop(event *Event) error {
	player.Stop()
	return nil
}

// Closes the event hub
func Close() error { return hub.Close() }
func (h *Hub) Close() error {
	logger.Debug("close event hub")
	defer logger.Info("closed event hub")
	close(h.closeC)
	h.closeWg.Wait() // Wait for internal routtines to exit
	return nil
}

// Constructor for the Event Hub
func NewHub() *Hub {
	return &Hub{
		clientsLock: &sync.Mutex{},
		clients:     make(Clients),
		eventsC:     make(chan []byte),
		closeWg:     &sync.WaitGroup{},
		closeC:      make(chan bool),
	}
}
