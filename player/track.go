package player

import "io"

// A store of tracks to play in any order
type Tracks map[string]*Track

// Get a track by id
func (t Tracks) Get(id string) *Track {
	track, ok := t[id]
	if !ok {
		return nil
	}
	return track
}

// Add a track to the track store
func (t Tracks) Add(track *Track) {
	t[track.PlaylistID] = track
}

// Delete a track from the store
func (t Tracks) Del(id string) {
	delete(t, id)
}

// Represents a track we are going to play at some point
type Track struct {
	// Exported Fields
	PlaylistID string   // Unique track id
	ProviderID string   // Providers track id
	Provider   Provider // Provider of the track
	// Unexpoted Fields
	stream io.ReadCloser // Track audio stream
}

// Reads from the track buffer
func (t *Track) Read(dst []byte) (int, error) {
	if t.stream == nil {
		return 0, io.ErrShortBuffer
	}
	return t.stream.Read(dst)
}

// Close the track closes the tracks buffer
func (t *Track) Close() error {
	if t.stream != nil {
		return t.stream.Close()
	}
	return nil
}

// Loads a tracks audio stream from the provider
func (t *Track) Load() error {
	stream, err := t.Provider.Stream(t.ProviderID)
	if err != nil {
		return err
	}
	t.stream = stream
	return nil
}

// Construct a new track
func NewTrack(playlistID, providerID string, provider Provider) *Track {
	return &Track{
		PlaylistID: playlistID,
		ProviderID: providerID,
		Provider:   provider,
	}
}
