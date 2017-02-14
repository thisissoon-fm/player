// Google Music Player

package googlemusic

import (
	"io"

	"player/buffer"

	"github.com/korandiz/mpa"
	"github.com/krak3n/gmusic"
)

type GoogleMusicStream struct {
	buffer  *buffer.HTTP
	decoder *mpa.Reader
}

func (gms *GoogleMusicStream) Read(dst []byte) (int, error) {
	return gms.decoder.Read(dst)
}

func (gms *GoogleMusicStream) Close() error {
	return gms.buffer.Close()
}

// Login Interface
type LoginHandler interface {
	Login(username, password string) (*gmusic.GMusic, error)
}

// Func type for building LoginHandlers
type LoginHandlerFunc func(username, password string) (*gmusic.GMusic, error)

// Implements the LoginHandler interface Login method
func (f LoginHandlerFunc) Login(username, password string) (*gmusic.GMusic, error) {
	return f(username, password)
}

// Default login handler
var DefaultLoginHandler = LoginHandlerFunc(func(username, password string) (*gmusic.GMusic, error) {
	return gmusic.Login(username, password)
})

// Login Handler
var Login LoginHandler = DefaultLoginHandler

// Google Music Player
type Player struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	gmusic *gmusic.GMusic // Google Music API
}

// Stream name
func (p *Player) Name() string {
	return "googlemusic"
}

// Requests the http steam from google music, returning an io.Reader of
// the response body
func (p *Player) Stream(track string) (io.ReadCloser, error) {
	rsp, err := p.gmusic.GetStream(track)
	if err != nil {
		return nil, err
	}
	// Createa http stream buffer
	buff := buffer.HTTPBuffer(rsp)
	go buff.Buffer() // Start buffering
	gms := &GoogleMusicStream{
		buffer:  buff,
		decoder: &mpa.Reader{Decoder: &mpa.Decoder{Input: buff}},
	}
	return gms, nil
}

// Constructs a new Player
func New(c Configurer) (*Player, error) {
	gm, err := Login.Login(c.Username(), c.Password())
	if err != nil {
		return nil, err
	}
	player := &Player{
		Config: c,
		gmusic: gm,
	}
	return player, nil
}
