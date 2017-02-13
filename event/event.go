package event

import (
	"encoding/json"
	"time"
)

const (
	PlayEvent    string = "player:play"
	PlayingEvent string = "player:playing"
	StopEvent    string = "player:stop"
	StopedEvent  string = "player:stoped"
	PauseEvent   string = "player:pause"
	PausedEvent  string = "player:paused"
	ResumeEvent  string = "player:resume"
	ResumedEvent string = "player:resumed"
	ErrorEvent   string = "player:error"
)

type Reader interface {
	Read() ([]byte, error)
}

type Writer interface {
	Write(b []byte) (int, error)
}

type Closer interface {
	Close() error
}

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

type Event struct {
	Type    string          `json:"type"`
	Created time.Time       `json:"created"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type PlayPayload struct {
	ProviderID string `json:"providerID"` // The provider ID name (googlemusic, soundcloud)
	TrackID    string `json:"trackID"`    // The track id from the provider
	PlaylistID string `json:"playlistID"` // The Playlist ID from the playlist service
}

type ErrorPayload struct {
	Error string `json:"error"`
}
