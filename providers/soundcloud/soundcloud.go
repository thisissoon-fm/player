package soundcloud

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"player/buffer"

	"github.com/korandiz/mpa"
)

type SoundCloudStream struct {
	buffer  *buffer.HTTP
	decoder *mpa.Reader
}

func (scs *SoundCloudStream) Read(dst []byte) (int, error) {
	return scs.decoder.Read(dst)
}

func (scs *SoundCloudStream) Close() error {
	return scs.buffer.Close()
}

// Soundcloud Player
type SoundCloud struct {
	// Exported Fields
	Config Configurer
	// Unexported Fields
	pw *io.PipeWriter
	pr *io.PipeReader
}

// constructs a steam url for the given track
func (sc *SoundCloud) streamUrl(t string) *url.URL {
	v := url.Values{}
	v.Add("client_id", sc.Config.ClientID())
	return &url.URL{
		Scheme:   sc.Config.APIScheme(),
		Host:     sc.Config.APIHost(),
		Path:     fmt.Sprintf("/tracks/%s/stream", t),
		RawQuery: v.Encode(),
	}
}

// Stream name
func (sc *SoundCloud) Name() string {
	return "soundcloud"
}

// Requests the http steam from soundcloud, returning an io.Reader of
// the response body
func (sc *SoundCloud) Stream(track string) (io.ReadCloser, error) {
	// Get the HTTP Stream
	rsp, err := http.Get(sc.streamUrl(track).String())
	if err != nil {
		return nil, err
	}
	// Createa http stream buffer
	buff := buffer.HTTPBuffer(rsp)
	go buff.Buffer() // Start buffering
	scs := &SoundCloudStream{
		buffer:  buff,
		decoder: &mpa.Reader{Decoder: &mpa.Decoder{Input: buff}},
	}
	return scs, nil
}

// Constructs a new player
func New(c Configurer) *SoundCloud {
	return &SoundCloud{
		Config: c,
	}
}
