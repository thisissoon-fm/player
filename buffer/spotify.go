package buffer

import (
	"io"
	"os"

	"player/logger"

	"github.com/djherbis/buffer"
	"github.com/op/go-libspotify/spotify"
)

// Spotify buffer
type Spotify struct {
	file     *os.File         // Buffer temporary file
	buffer   buffer.BufferAt  // Internal Buffer
	buffered int              // Amount buffered
	session  *spotify.Session // Spotify Session
	bufferC  chan bool
	closeC   chan bool
	doneC    chan bool // Done channel
}

// Read from the buffer
func (s *Spotify) Read(b []byte) (int, error) {
	// If we try and read before we have a buffer return an error
	// that we don't yet have a buffer
	if s.buffer == nil {
		return 0, io.ErrShortBuffer
	}
	// Wait for buffer to fill
	if s.buffered < 16*1024 {
		return 0, io.ErrShortBuffer
	}
	// If we are at the end of the buffer return EOF
	if s.buffer.Len() == 0 {
		return 0, io.EOF
	}
	// Read from the buffer
	return s.buffer.Read(b)
}

// Done channel
func (s *Spotify) Done() <-chan bool {
	return (<-chan bool)(s.doneC)
}

func (s *Spotify) Close() error {
	close(s.closeC)
	<-s.bufferC // Wait for buffer method to exit
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			logger.WithError(err).Error("error closing buffer file")
			return err
		}
		if err := os.Remove(s.file.Name()); err != nil {
			logger.WithError(err).Error("error removing buffer file")
			return err
		}
	}
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
	logger.Debug("start spotify buffer")
	defer logger.Debug("exit spotify buffer")
	// Make the buffer
	file, buff, err := Make(1024 * 1024 * 104) // 100mb
	if err != nil {
		return err
	}
	s.file = file
	s.buffer = buff
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
		s.doneC <- true
		s.bufferC <- true
		return nil
	case <-s.closeC:
		s.bufferC <- true
		return nil
	}
}

func SpotifyBuffer(session *spotify.Session) *Spotify {
	return &Spotify{
		session: session,
		bufferC: make(chan bool, 1),
		doneC:   make(chan bool, 1),
		closeC:  make(chan bool),
	}
}
