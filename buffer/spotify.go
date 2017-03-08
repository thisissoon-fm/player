package buffer

import (
	"io"
	"os"
	"sync"

	"player/logger"

	"github.com/djherbis/buffer"
	"github.com/op/go-libspotify/spotify"
)

// Spotify buffer
type Spotify struct {
	file     *os.File         // Buffer temporary file
	buffer   buffer.Buffer    // Internal Buffer
	buffered int              // Amount buffered
	session  *spotify.Session // Spotify Session
	wg       sync.WaitGroup
	closeC   chan bool
}

// Read from the buffer
func (s *Spotify) Read(b []byte) (int, error) {
	// If we try and read before we have a buffer return an error
	// that we don't yet have a buffer
	if s.buffer == nil {
		return 0, io.ErrShortBuffer
	}
	// Wait for buffer to fill
	if s.buffered < 64*1024 {
		return 0, io.ErrShortBuffer
	}
	// If we are at the end of the buffer return EOF
	if s.buffer.Len() == 0 {
		return 0, io.EOF
	}
	// Read from the buffer
	return s.buffer.Read(b)
}

func (s *Spotify) Close() error {
	logger.Debug("close spotify buffer")
	defer logger.Debug("closed spotify buffer")
	close(s.closeC)
	s.wg.Wait()
	return nil
}

// Writes audio to buffer
func (s *Spotify) WriteAudio(f spotify.AudioFormat, raw []byte) int {
	select {
	case <-s.closeC:
		return 0
	default:
		i, err := s.buffer.Write(raw)
		if err != nil {
			logger.WithError(err).Error("error writting audio")
		}
		s.buffered += i
		return i
	}
}

func (s *Spotify) Buffer(track *spotify.Track) error {
	s.wg.Add(1)
	defer s.wg.Done()
	logger.Debug("start spotify buffer")
	defer logger.Debug("exit spotify buffer")
	// Buffer to memory for spotify, results in smoother playback
	buf := buffer.NewPartition(buffer.NewMemPool(128 * 1024))
	s.buffer = buf
	// Start playing - writes to buffer instead of autio out
	player := s.session.Player()
	if err := player.Prefetch(track); err != nil {
		return err
	}
	if err := player.Load(track); err != nil {
		return err
	}
	player.Play()
	defer player.Unload() // Unload the track once we are done
	select {
	case <-s.session.EndOfTrackUpdates():
		logger.Debug("end of track updates")
		return nil
	case <-s.closeC:
		return nil
	}
}

func SpotifyBuffer(session *spotify.Session) *Spotify {
	return &Spotify{
		session: session,
		closeC:  make(chan bool),
	}
}
