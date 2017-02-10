package socket

import (
	"io"
	"net"
	"os"
	"sync"

	"player/event"
	"player/logger"
)

// A type that implements the event.Event interface
type Event struct {
	raw    []byte
	client *Client
}

// Implements the RawMessage method of the event.Event interface
// returning the raw event bytes
func (e Event) RawMessage() []byte {
	return e.raw
}

// Implements the ResponseWriter method of the event.Event interface
// returning the Unix socket client the event oigionated from which
// itself implements the io.Writer interface
func (e Event) ResponseWriter() io.Writer {
	return e.client
}

// Socket server type
type Server struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	events   chan event.Event // Received events from clients
	listener net.Listener     // Unix socket listener
	clients  Clients          // Connected clients
	wg       *sync.WaitGroup  // Wait group for clean exit
	closeC   chan bool        // close channel for close orchestration
}

// Name of the event producer
func (s *Server) Name() string {
	return "unix socket server"
}

// Connected client reader
func (s *Server) read(conn net.Conn) {
	defer logger.Debug("exit socket server client read routine")
	s.wg.Add(1)
	defer s.wg.Done()
	client := NewClientWithConn(s.Config, conn)
	defer client.Close()
	s.clients.Add(client)
	defer s.clients.Del(client.ID())
	for b := range client.Read() {
		s.events <- &Event{b, client}
	}
}

// Implements the event.Producer interface.
func (s *Server) Events() <-chan event.Event {
	return (<-chan event.Event)(s.events)
}

// Listens for new Unix socket client connections
// Saves accepted connections
func (s *Server) Listen() error {
	logger.Debug("start socket server listen")
	defer logger.Debug("exit socket server listen")
	s.wg.Add(1)
	defer s.wg.Done()
	l, err := net.Listen("unix", s.Config.Address())
	if err != nil {
		return err
	}
	s.listener = l
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-s.closeC:
				return nil
			default:
				logger.WithError(err).Error("failed to accept unix connection")
			}
		} else {
			logger.Debug("unix socket client connected")
			go s.read(conn)
		}
	}
	return nil
}

// Gracefully closes the socket connection, waits for the all connected
// client connections to close and listener loops
func (s *Server) Close() error {
	logger.Debug("close socket server close")
	defer logger.Info("closed socket server")
	defer os.Remove(s.Config.Address())
	if s.closeC != nil {
		close(s.closeC)
	}
	if s.listener != nil {
		s.listener.Close()
	}
	for _, client := range s.clients {
		client.Close()
	}
	s.wg.Wait()
	return nil
}

// Constructs a new Socket Server
func NewServer(c Configurer) *Server {
	return &Server{
		Config:  c,
		events:  make(chan event.Event, 10),
		closeC:  make(chan bool),
		clients: make(Clients),
		wg:      &sync.WaitGroup{},
	}
}
