// Websocket Client
//
// Connects to the configured websocket server.
// Usage:
// ws := web.New(...)
// go ws.Connect() // Gracefull Reconnection
// defer ws.Close()
// i, err := svc.Write([]byte("hello"))
// for {
//     b, _ := ws.Read()
//     fmt.Println(string(b))
// }
//

package web

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"player/event"
	"player/logger"

	"github.com/gorilla/websocket"
	"github.com/rs/xid"
)

type message struct {
	typ int
	msg []byte
	err error
}

// Websocket connection interface
type ReadWriteCloser interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// Default dialer function
var dialer Dialer = &websocket.Dialer{}

// Implemented by websocket.Dialer
type Dialer interface {
	Dial(urlStr string, headers http.Header) (*websocket.Conn, *http.Response, error)
}

// Websocket client
type Client struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	id string
	// Connection & state
	conn      ReadWriteCloser
	connected bool
	// Received messages
	messageC chan message
	// Orchestraion
	wg       *sync.WaitGroup
	closeC   chan bool
	connectC chan bool
}

// Constructs the connection url
func (c *Client) url() string {
	u := url.URL{Scheme: "ws", Host: c.Config.Host(), Path: "/"}
	return u.String()
}

// Returns headers tp use for connecting to the server
func (c *Client) headers() http.Header {
	return http.Header{}
}

// Connect to server
func (c *Client) connect() error {
	logger.WithField("url", c.url()).Debug("connecting to websocket server")
	conn, _, err := dialer.Dial(c.url(), c.headers())
	if err != nil {
		return err
	}
	c.connected = true
	c.conn = conn
	return nil
}

// Reads messages from the websocket connection
func (c *Client) read() {
	c.wg.Add(1)
	defer c.wg.Done()
	logger.Debug("start websocket read loop")
	defer logger.Debug("exit websocket read loop")
	for c.connected {
		typ, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.connected = false
			c.conn = nil
			event.Del(c) // Remove from event hub
			defer logger.WithError(err).Error("error reading websocket server")
			select {
			case <-c.closeC:
				// Don't connect if closing
				return
			default:
				go c.Connect() // Start the connection routine
				return
			}
		}
		c.messageC <- message{typ, msg, err} // Place message on channel
	}
}

// Returns instance ID
func (c *Client) ID() string {
	return c.id
}

// Connect to the websocket server
func (c *Client) Connect() {
	c.wg.Add(1)
	defer c.wg.Done()
	logger.Debug("start websocket connect loop")
	defer logger.Debug("exit websocket service connect loop")
	c.connectC <- true // connect immediately
	for {
		select {
		case <-c.closeC:
			return
		case <-c.connectC:
			break
		case <-time.After(c.Config.Retry()):
			break
		}
		if err := c.connect(); err != nil {
			logger.WithError(err).WithFields(logger.F{
				"retry": c.Config.Retry(),
				"url":   c.url(),
			}).Error("failed connecting to websocket server")
			continue
		}
		event.Add(c) // Add to event hub
		go c.read()  // Start a read routine
		return
	}
}

// Read messages from the websocket server
func (c *Client) Read() ([]byte, error) {
	select {
	case <-c.closeC:
		return nil, io.EOF
	case message := <-c.messageC:
		return message.msg, message.err
	}
}

// Writes messages to websocket server
func (c *Client) Write(b []byte) (int, error) {
	if c.connected && c.conn != nil {
		if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
			return 0, err
		}
		return len(b), nil
	}
	logger.Warn("unable to write to websocket server")
	return 0, nil
}

// Gracefully closes the websocket connection
func (c *Client) Close() error {
	logger.Debug("close websocket client")
	defer logger.Info("closed websocket client")
	// Close the closeC
	close(c.closeC)
	// Close the websocket connection
	if c.connected && c.conn != nil {
		err := c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.WithError(err).Error("error closing connection")
		}
		if err := c.conn.Close(); err != nil {
			logger.WithError(err).Error("error closing connection")
		}
	}
	// Wait for routines to exit
	c.wg.Wait()
	return nil
}

// Constructs a new websocket Client
func New(c Configurer) *Client {
	return &Client{
		// Exported Fields
		Config: c,
		// ID
		id: xid.New().String(),
		// Read messages
		messageC: make(chan message),
		// Orechestration
		wg:       &sync.WaitGroup{},
		closeC:   make(chan bool, 1),
		connectC: make(chan bool, 1),
	}
}
