package spotify

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"player/buffer"

	"github.com/op/go-libspotify/spotify"
)

type Spotify struct {
	Config  Configurer
	session *spotify.Session
}

func (s *Spotify) Name() string {
	return "spotify"
}

func (s *Spotify) Stream(trackID string) (io.ReadCloser, error) {
	link, err := s.session.ParseLink(trackID)
	if err != nil {
		return nil, err
	}
	if link.Type() != spotify.LinkTypeTrack {
		return nil, errors.New("not a spotify track")
	}
	track, err := link.Track()
	if err != nil {
		return nil, err
	}
	track.Wait()                            // Wait for track
	buff := buffer.SpotifyBuffer(s.session) // Create a buffer to write too
	s.session.SetAudioConsumer(buff)        // Set spotify to write to the buffer
	go buff.Buffer(track)                   // Start buffering the track
	return buff, nil
}

// Cleanly close spotify
func (s *Spotify) Close() error {
	s.session.Logout()
	s.session.Close()
	return nil
}

// Construct a new spotify streamer
func New(config Configurer) (*Spotify, error) {
	// Read Spotify App Key
	key, err := ioutil.ReadFile(config.APIKey())
	if err != nil {
		return nil, err
	}
	// Create a spotify session
	session, err := spotify.NewSession(&spotify.Config{
		ApplicationKey:               key,
		ApplicationName:              "SOON_ FM 2.0",
		CacheLocation:                os.TempDir(),
		SettingsLocation:             os.TempDir(),
		DisablePlaylistMetadataCache: true,
		InitiallyUnloadPlaylists:     true,
	})
	session.PreferredBitrate(spotify.Bitrate320k)
	creds := spotify.Credentials{
		Username: config.Username(),
		Password: config.Password(),
	}
	if err := session.Login(creds, true); err != nil {
		return nil, err
	}
	return &Spotify{
		Config:  config,
		session: session,
	}, nil
}
