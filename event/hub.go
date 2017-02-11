package event

import (
	"sync"

	"player/logger"

	"github.com/rs/xid"
)

// Global hub
var hub *Hub

// Initialise package with a global hub
func init() {
	hub = NewHub()
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
	clientsLock *sync.Mutex
	clients     Clients
	eventsC     chan []byte
	closeWg     *sync.WaitGroup // Wait for internal coroutines to exit
	closeC      chan bool       // Closes internal coroutines
}

// Goroutine for reading client events
func (h *Hub) read(rwc ReadWriteCloser) {
	logger.Debug("start event hub client read")
	defer logger.Debug("exit event hub client read")
	h.closeWg.Add(1)
	defer h.closeWg.Done()
	for {
		select {
		case b := <-rwc.Read():
			h.eventsC <- b
		case <-h.closeC:
			return
		}
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
			logger.Debug(string(b))
		case <-h.closeC:
			return
		}
	}
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
