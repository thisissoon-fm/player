package socket

import (
	"bufio"
	"net"
	"sync"

	"player/logger"

	"github.com/rs/xid"
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

// A unix socket client
type Client struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	id     string
	conn   net.Conn
	wg     *sync.WaitGroup
	closed bool
	closeC chan bool
}

// Returns the clients ID
func (c *Client) ID() string {
	return c.id
}

// Connect to a Unix socket
func (c *Client) Connect() error {
	conn, err := net.Dial("unix", c.Config.Address())
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Reads data from the socket connection
func (c *Client) read(ch chan []byte) {
	logger.Debug("start socket client read loop")
	defer logger.Debug("exit socket client read loop")
	c.wg.Add(1)
	defer c.wg.Done()
	defer close(ch)
	buf := bufio.NewReader(c.conn)
	for {
		b, err := buf.ReadBytes('\n') // EOF on connction close
		if err != nil {
			return
		}
		ch <- b
	}
}

// Read data from the socket
func (c *Client) Read() <-chan []byte {
	ch := make(chan []byte)
	go c.read(ch) // Start the read loop
	return (<-chan []byte)(ch)
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
	logger.Debug("close socket client")
	defer logger.Info("closed socket client")
	if !c.closed {
		close(c.closeC)
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				logger.WithError(err).Error("failed to close socket client conn")
			}
		}
		c.closed = true
		c.wg.Wait()
	}
	return nil
}

// Constructs a new Client
func NewClient(c Configurer) *Client {
	return &Client{
		Config: c,
		id:     xid.New().String(),
		wg:     &sync.WaitGroup{},
		closeC: make(chan bool),
	}
}

// Constructs a new client with an already open connection
func NewClientWithConn(c Configurer, conn net.Conn) *Client {
	client := NewClient(c)
	client.conn = conn
	return client
}
