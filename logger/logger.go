package logger

import (
	"io/ioutil"
	"strings"

	"player/build"
	"player/logger/formatters"
	"player/logger/hooks"

	"github.com/sirupsen/logrus"
)

// Construct a new global logger with default configuration
func init() {
	global = New(NewConfig())
}

// Glogal logger with default configuration
var global *logger

// A short-form wrapper around logrus.Fields
type F logrus.Fields

// Common logger interface
type Logger interface {
	WithError(error) Logger
	WithField(string interface{}) Logger
	WithFields(F) Logger
	Debug(string, ...interface{})
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Fatal(string, ...interface{})
	Panic(string, ...interface{})
}

// Logger
type logger struct {
	config Configurer
	entry  *logrus.Entry
	logger *logrus.Logger
}

// Sets up the logger according to configuration
func Setup() { global.Setup() }
func (l *logger) Setup() {
	l.DeleteHooks()
	l.SetLevel(l.config.Level())
	l.ConsoleOutput(l.config.ConsoleOutput())
	l.LogToFile(l.config.LogFile())
	l.SetFormat(l.config.Format())
}

// Removes all hooks from the logger, call this before each setup
func (l *logger) DeleteHooks() {
	for k, _ := range l.logger.Hooks {
		delete(l.logger.Hooks, k)
	}
}

// Set the log level of the logger
func SetLevel(lvl string) { global.SetLevel(lvl) }
func (l *logger) SetLevel(lvl string) {
	lvl = strings.ToLower(lvl)
	switch lvl {
	case "debug":
		l.logger.Level = logrus.DebugLevel
	case "warn":
		l.logger.Level = logrus.WarnLevel
	case "error":
		l.logger.Level = logrus.ErrorLevel
	default:
		l.logger.Level = logrus.InfoLevel
	}
}

// Enable or disbale console output
func ConsoleOutput(enable bool) {}
func (l *logger) ConsoleOutput(enable bool) {
	l.entry.Logger.Out = ioutil.Discard
	if enable {
		l.logger.Hooks.Add(hooks.NewConsoleHook())
	}
}

// Log to a file
func LogToFile(path string) { global.LogToFile(path) }
func (l *logger) LogToFile(path string) {
	if path != "" {
		l.logger.Hooks.Add(hooks.NewFileHook(path))
	}
}

// Set the format of the logger
func SetFormat(fmt string) { global.SetFormat(fmt) }
func (l *logger) SetFormat(fmt string) {
	switch fmt {
	case "logstash":
		l.setLogstashFormat(l.config.LogstashType())
	case "json":
		l.setJSONFormat()
	default:
		l.setTextFormat()
	}
}

// Set log format to use logstash format
func (l *logger) setLogstashFormat(typ string) {
	if typ == "" {
		typ = "scoreboard"
	}
	l.logger.Formatter = &formatters.LogstashFormatter{
		Type: typ,
	}
}

// Set log format to use json
func (l *logger) setJSONFormat() {
	l.logger.Formatter = &logrus.JSONFormatter{}
}

// Set text logger
func (l *logger) setTextFormat() {
	l.logger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}
}

// Log a field and value
func WithField(k string, v interface{}) *logger { return global.WithField(k, v) }
func (l *logger) WithField(k string, v interface{}) *logger {
	return &logger{
		config: l.config,
		entry:  l.entry.WithField(k, v),
	}
}

// Log a with multiple fields
func WithFields(fields F) *logger { return global.WithFields(fields) }
func (l *logger) WithFields(fields F) *logger {
	return &logger{
		config: l.config,
		entry:  l.entry.WithFields(logrus.Fields(fields)),
	}
}

// Log an error
func WithError(err error) *logger { return global.WithError(err) }
func (l *logger) WithError(err error) *logger {
	return &logger{
		config: l.config,
		entry:  l.entry.WithError(err),
	}
}

// Log a debug message
func Debug(msg string, v ...interface{}) { global.Debug(msg, v...) }
func (l *logger) Debug(msg string, v ...interface{}) {
	l.entry.Debugf(msg, v...)
}

// Log an info message
func Info(msg string, v ...interface{}) { global.Info(msg, v...) }
func (l *logger) Info(msg string, v ...interface{}) {
	l.entry.Infof(msg, v...)
}

// Log a warning message
func Warn(msg string, v ...interface{}) { global.Warn(msg, v...) }
func (l *logger) Warn(msg string, v ...interface{}) {
	l.entry.Warnf(msg, v...)
}

// Log an error message
func Error(msg string, v ...interface{}) { global.Error(msg, v...) }
func (l *logger) Error(msg string, v ...interface{}) {
	l.entry.Errorf(msg, v...)
}

// Log a fatal error, this causes the application to exit
func Fatal(msg string, v ...interface{}) { global.Fatal(msg, v...) }
func (l *logger) Fatal(msg string, v ...interface{}) {
	l.entry.Fatalf(msg, v...)
}

// Log a panic error, this causes the application to panic
func Panic(msg string, v ...interface{}) { global.Panic(msg, v...) }
func (l *logger) Panic(msg string, v ...interface{}) {
	l.entry.Panicf(msg, v...)
}

// Exported logger constructor, requiring a config type that
// implments the config interface
func New(config Configurer) *logger {
	log := logrus.New()
	logger := &logger{
		config: config,
		logger: log,
		entry: logrus.NewEntry(log).WithFields(logrus.Fields{
			"version":   build.Version(),
			"buildTime": build.TimeStr(),
		}),
	}
	logger.Setup()
	return logger
}

// Update the global logger to a different logger
func SetGlobalLogger(l *logger) {
	global = l
}
