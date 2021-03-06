package event

import (
	"encoding/json"
	"time"
)

const (
	PlayerReadyEvent   string = "player:ready"
	PlayerOfflineEvent string = "player:offline"
	PlayEvent          string = "player:play"
	PlayingEvent       string = "player:playing"
	StopEvent          string = "player:stop"
	StoppedEvent       string = "player:stopped"
	PauseEvent         string = "player:pause"
	PausedEvent        string = "player:paused"
	ResumeEvent        string = "player:resume"
	ResumedEvent       string = "player:resumed"
	ErrorEvent         string = "player:error"
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
	Topic   string          `json:"topic"`
	Created time.Time       `json:"created"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type PlayPayload struct {
	ProviderName    string `json:"providerID"`      // The provider name (googlemusic, soundcloud)
	ProviderTrackID string `json:"providerTrackID"` // The provider track id from the provider
	PlaylistID      string `json:"playlistID"`      // The Playlist ID from the playlist service
	UserID          string `json:"userID"`          // The user who queued the track
}

type ErrorPayload struct {
	Error string `json:"error"`
}
