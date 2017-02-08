package soundcloud

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Soundcloud Player
type Player struct {
	// Exported Fields
	Config Configurer
}

// constructs a steam url for the given track
func (p *Player) streamUrl(t string) *url.URL {
	v := url.Values{}
	v.Add("client_id", p.Config.ClientID())
	return &url.URL{
		Scheme:   p.Config.APIScheme(),
		Host:     p.Config.APIHost(),
		Path:     fmt.Sprintf("/tracks/%s/stream", t),
		RawQuery: v.Encode(),
	}
}

// Stream name
func (p *Player) Name() string {
	return "soundcloud"
}

// Requests the http steam from soundcloud, returning an io.Reader of
// the response body
func (p *Player) Stream(track string) (io.ReadCloser, error) {
	u := p.streamUrl(track).String()
	fmt.Println(u)
	rsp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	return rsp.Body, err
}

// Constructs a new player
func New(c Configurer) *Player {
	return &Player{
		Config: c,
	}
}
