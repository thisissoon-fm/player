package run

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSigChanFunc(t *testing.T) {
	tt := []struct {
		name     string
		sig      os.Signal
		expected os.Signal
	}{
		{
			"makes channel",
			syscall.SIGINT,
			syscall.SIGINT,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ch := defaultSigChanFunc()
			ch <- tc.sig
			sig := <-ch
			assert.Equal(t, tc.expected, sig)
		})
	}

}

func TestUntilSignal(t *testing.T) {
	tt := []struct {
		name     string
		sigs     []os.Signal
		sig      os.Signal
		ch       chan os.Signal
		expected os.Signal
	}{
		{
			"hangup",
			[]os.Signal{syscall.SIGHUP},
			syscall.SIGHUP,
			make(chan os.Signal, 1),
			syscall.SIGHUP,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defer func() { SigChanFunc = defaultSigChanFunc }()
			SigChanFunc = func() chan os.Signal {
				return tc.ch
			}
			tc.ch <- tc.sig
			s := UntilSignal(tc.sigs...)
			assert.Equal(t, tc.expected, s)
		})
	}
}

func TestUntilQuit(t *testing.T) {
	tt := []struct {
		name     string
		sig      os.Signal
		ch       chan os.Signal
		expected os.Signal
	}{
		{
			"interrupt",
			syscall.SIGINT,
			make(chan os.Signal, 1),
			syscall.SIGINT,
		},
		{
			"quit",
			syscall.SIGQUIT,
			make(chan os.Signal, 1),
			syscall.SIGQUIT,
		},
		{
			"terminate",
			syscall.SIGTERM,
			make(chan os.Signal, 1),
			syscall.SIGTERM,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defer func() { SigChanFunc = defaultSigChanFunc }()
			SigChanFunc = func() chan os.Signal {
				return tc.ch
			}
			tc.ch <- tc.sig
			s := UntilQuit()
			assert.Equal(t, tc.expected, s)
		})
	}
}
