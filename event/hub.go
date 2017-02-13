package event

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"player/logger"
	"player/player"

	"github.com/rs/xid"
)

// Global hub
var hub *Hub

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
func (c Clients) Add(rwc ReadWriteCloser) string {
	id := xid.New().String()
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
func (h *Hub) read(rwc ReadWriteCloser) {
	defer rwc.Close()
	logger.Debug("start event hub client read")
	defer logger.Debug("exit event hub client read")
	h.closeWg.Add(1)
	defer h.closeWg.Done()
	for {
		b, err := rwc.Read()
		if err != nil {
			if err != io.EOF {
				logger.WithError(err).Error("unexpected hub read error")
			}
			return
		}
		h.eventsC <- b
	}
}

// Add a client to the hub and start reading from it
// returns the client id
func AddClient(rwc ReadWriteCloser) string { return hub.AddClient(rwc) }
func (h *Hub) AddClient(rwc ReadWriteCloser) string {
	h.clientsLock.Lock()
	id := h.clients.Add(rwc)
	go h.read(rwc)
	h.clientsLock.Unlock()
	return id
}

// Remove a client from the hub
func DelClient(id string) { hub.DelClient(id) }
func (h *Hub) DelClient(id string) {
	h.clientsLock.Lock()
	h.clients.Del(id)
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
	}
	return err
}

// Pause event handler
func (h *Hub) pause(event *Event) error {
	player.Pause()
	return nil
}

// Resume event handler
func (h *Hub) resume(event *Event) error {
	player.Resume()
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
