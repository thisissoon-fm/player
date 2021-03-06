package unix

import (
	"net"
	"os"
	"sync"

	"player/event"
	"player/logger"
)

// Map for storing client connections
type Clients map[string]*Client

// Convenience add client connection to map
func (c Clients) Add(client *Client) {
	c[client.id] = client
}

// Convenience delete client connection from map
func (c Clients) Del(id string) {
	delete(c, id)
}

// Socket server type
type Server struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	listener net.Listener    // Unix socket listener
	clients  Clients         // Connected clients
	wg       *sync.WaitGroup // Wait group for clean exit
	closeC   chan bool       // close channel for close orchestration
}

// Name of the event producer
func (s *Server) Name() string {
	return "unix socket server"
}

// Listens for new Unix socket client connections
// Saves accepted connections
func (s *Server) Listen() error {
	logger.Debug("start socket server listen")
	defer logger.Debug("exit socket server listen")
	s.wg.Add(1)
	defer s.wg.Done()
	addr := s.Config.Address()
	if _, err := os.Stat(addr); err == nil {
		if err := os.Remove(addr); err != nil {
			return err
		}
	}
	l, err := net.Listen("unix", addr)
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
				continue
			}
		}
		s.newClientConn(conn)
	}
	return nil
}

// Adds the client to the server and adds it to the event hub
func (s *Server) newClientConn(conn net.Conn) {
	defer logger.Debug("unix socket client connected")
	client := NewServerClient(s, conn)
	s.clients.Add(client)
	event.Add(client)
}

// Gracefully closes the socket connection, waits for the all connected
// client connections to close and listener loops
func (s *Server) Close() error {
	logger.Debug("close socket server")
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
		closeC:  make(chan bool),
		clients: make(Clients),
		wg:      &sync.WaitGroup{},
	}
}
