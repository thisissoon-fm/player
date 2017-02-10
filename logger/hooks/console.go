// A simple logrus hook for logging errors to stderr and debug
// messages to  stdout

package hooks

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Console struct {
	stdout io.Writer
	stderr io.Writer
}

func (hook *Console) Fire(entry *logrus.Entry) error {
	var err error
	serialized, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return err
	}
	if entry.Level <= logrus.ErrorLevel {
		_, err = hook.stderr.Write(serialized)
	} else {
		_, err = hook.stdout.Write(serialized)
	}
	return err
}

// Returns the log levels support by this hook
func (hook *Console) Levels() []logrus.Level {
	return logrus.AllLevels
}

func NewConsoleHook() *Console {
	return &Console{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}
