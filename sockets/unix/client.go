package unix

import (
	"bufio"
	"net"
	"sync"

	"player/event"
	"player/logger"

	"github.com/rs/xid"
)

// A unix socket client
type Client struct {
	// Unexported Fields
	id     string
	conn   net.Conn
	server *Server
	wg     *sync.WaitGroup
	closed bool
	closeC chan bool
}

// Returns the clients ID
func (c *Client) ID() string {
	return c.id
}

// Connect to a Unix socket
func (c *Client) Connect(address string) error {
	conn, err := net.Dial("unix", address)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Read data from the socket
func (c *Client) Read() ([]byte, error) {
	buf := bufio.NewReader(c.conn)
	b, err := buf.ReadBytes('\n') // EOF on connction close
	if err != nil {
		// On read error we close the client, it's likely that
		// the client has gone away so we can't read or write from
		// it anymore
		if err := c.Close(); err != nil {
			logger.WithError(err).Error("error closing socket client on read error")
		}
	}
	return b, err
}

// Writes data to the client unix socket connection
func (c *Client) Write(b []byte) (int, error) {
	last := b[len(b)-1]
	if last != '\n' {
		b = append(b, '\n')
	}
	return c.conn.Write(b)
}

// Close the Client, closing the connection
func (c *Client) Close() error {
	event.Del(c) // Delete the client from the event hub
	if !c.closed {
		logger.Debug("close socket client")
		defer logger.Info("closed socket client")
		close(c.closeC)
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				logger.WithError(err).Error("failed to close socket client conn")
			}
		}
		if c.server != nil {
			c.server.clients.Del(c.id)
		}
		c.closed = true
		c.wg.Wait()
	}
	return nil
}

// Constructs a new Client
func NewClient() *Client {
	return &Client{
		id:     xid.New().String(),
		wg:     &sync.WaitGroup{},
		closeC: make(chan bool),
	}
}

// Constructs a new server client
func NewServerClient(server *Server, conn net.Conn) *Client {
	client := NewClient()
	client.conn = conn
	client.server = server
	return client
}
