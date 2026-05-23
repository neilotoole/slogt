// Package slogt bridges Go's stdlib testing package and the log/slog
// logging library: slogt.New(t) returns a *slog.Logger whose output is
// routed to t.Output(), so log lines are correlated with the running test.
package slogt

import (
	"io"
	"log/slog"
	"sync"
	"testing"
)

var (
	defaultMu     sync.RWMutex
	defaultOption = Text()
)

// SetDefault sets the Option that New applies when it is called without a
// handler Option. The initial default is Text(). SetDefault is safe for
// concurrent use; a nil opt resets the default to Text().
func SetDefault(opt Option) {
	if opt == nil {
		opt = Text()
	}
	defaultMu.Lock()
	defaultOption = opt
	defaultMu.Unlock()
}

// getDefault returns the current default Option.
func getDefault() Option {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultOption
}

// Option is a functional option type that is used
// with New to configure the logger's underlying handler.
type Option func(c *config)

// config holds the resolved handler constructor used by New.
type config struct {
	newHandler func(w io.Writer) slog.Handler
}

// Text specifies a text handler.
//
//	log := slogt.New(t, slogt.Text())
func Text() Option {
	return func(c *config) {
		c.newHandler = func(w io.Writer) slog.Handler {
			return slog.NewTextHandler(w, &slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelDebug,
			})
		}
	}
}

// JSON specifies a JSON handler.
//
//	log := slogt.New(t, slogt.JSON())
func JSON() Option {
	return func(c *config) {
		c.newHandler = func(w io.Writer) slog.Handler {
			return slog.NewJSONHandler(w, &slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelDebug,
			})
		}
	}
}

// Factory specifies a custom factory function for
// creating the logger's underlying handler.
func Factory(fn func(w io.Writer) slog.Handler) Option {
	return func(c *config) {
		c.newHandler = fn
	}
}

// New returns a new *slog.Logger whose output is routed to t.Output()
// (added in Go 1.25). Output is correlated with the test like t.Log, but
// carries no bogus callsite prefix. Set AddSource: true on the handler to
// surface the real caller as a source= attribute.
func New(t testing.TB, opts ...Option) *slog.Logger {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}

	if c.newHandler == nil {
		// No handler set yet, use the default.
		getDefault()(c)
	}

	return slog.New(c.newHandler(t.Output()))
}
