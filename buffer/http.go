// Buffer a http stream to a file for reading by the player

package buffer

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"player/logger"

	"github.com/djherbis/buffer"
)

// HTTP Buffer
type HTTP struct {
	// Exported Fields
	Response *http.Response // HTTP Response Object containing the HTTP Stream
	// Unexported Fields
	file     *os.File        // Buffer temporary file
	buffer   buffer.BufferAt // Internal Buffer
	buffered int             // Amount buffered
}

// Read from the buffer
func (h *HTTP) Read(b []byte) (int, error) {
	// If we try and read before we have a buffer return an error
	// that we don't yet have a buffer
	if h.buffer == nil {
		return 0, io.ErrShortBuffer
	}
	// Wait for buffer to fill
	if h.buffered < 64*1024 {
		return 0, io.ErrShortBuffer
	}
	// If we are at the end of the buffer return EOF
	if h.buffer.Len() == 0 {
		return 0, io.EOF
	}
	// Read from the buffer
	return h.buffer.Read(b)
}

// Closes and removes the temporary buffer file
func (h *HTTP) Close() error {
	if h.file != nil {
		if err := h.file.Close(); err != nil {
			logger.WithError(err).Error("error closing buffer file")
			return err
		}
		if err := os.Remove(h.file.Name()); err != nil {
			logger.WithError(err).Error("error removing buffer file")
			return err
		}
	}
	return nil
}

// Buffer the http Response into a temporary file for reading
func (h *HTTP) Buffer() error {
	f := logger.F{"size": h.Response.ContentLength}
	logger.WithFields(f).Debug("start http buffer")
	defer logger.WithFields(f).Debug("finished buffering")
	defer h.Response.Body.Close() // Close the HTTP Response body once we are done
	// Make the buffer
	if err := h.mkBuffer(); err != nil {
		return err
	}
	var eof bool
	data := make([]byte, 1024*8) // Read response data into here
	writer := bufio.NewWriter(h.buffer)
	for {
		// Read data from response body
		rn, err := h.Response.Body.Read(data)
		if err != nil {
			switch err {
			case io.EOF:
				eof = true
			default:
				logger.WithError(err).Error("http response body read error")
				return err
			}
		}
		// Write body data to buffer
		wn, err := writer.Write(data[:rn])
		if err != nil {
			logger.WithError(err).Error("buffer write error")
			return err
		}
		h.buffered += wn
		if eof {
			return nil
		}
	}
}

// Makes a buffer to store the HTTP stream to a temporary file
func (h *HTTP) mkBuffer() error {
	logger.Debug("make buffer")
	f, err := ioutil.TempFile(os.TempDir(), "sfmplayer.buffer")
	if err != nil {
		return err
	}
	logger.WithField("path", f.Name()).Debug("tmp file created")
	b := buffer.NewFile(h.Response.ContentLength, f)
	h.buffer = b
	h.file = f
	return nil
}

// Construct a new HTTP Buffer for a HTTP Response
func HTTPBuffer(rsp *http.Response) *HTTP {
	return &HTTP{
		Response: rsp,
	}
}
