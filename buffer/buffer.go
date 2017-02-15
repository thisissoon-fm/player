package buffer

import (
	"io/ioutil"
	"os"

	"player/logger"

	"github.com/djherbis/buffer"
)

// Creates a new buffer temporary file
func Make(size int64) (*os.File, buffer.BufferAt, error) {
	logger.WithField("size", size).Debug("make new buffer")
	file, err := ioutil.TempFile(os.TempDir(), "sfmplayer.buffer")
	if err != nil {
		return nil, nil, err
	}
	logger.WithField("path", file.Name()).Debug("tmp file created")
	buff := buffer.NewFile(size, file)
	return file, buff, nil
}
