package run

import (
	"os"
	"os/signal"
	"syscall"

	"player/logger"
)

// os.Signal channel function
var SigChanFunc = defaultSigChanFunc

// Default os.Signal channel function
func defaultSigChanFunc() chan os.Signal {
	return make(chan os.Signal, 1)
}

// Run this method until the passed in os.Signals are triggered
// Returns the recieved signal
func UntilSignal(signals ...os.Signal) os.Signal {
	ch := SigChanFunc()
	signal.Notify(ch, signals...)
	sig := <-ch // Blocking
	return sig
}

// Run until a quit signal is recieved
func UntilQuit() os.Signal {
	signals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
	return UntilSignal(signals...)
}

// Panic Recover
func Recover() {
	if r := recover(); r != nil {
		logger.Error("panic recovery: %v", r)
	}
}
