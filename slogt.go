package slogt

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"golang.org/x/exp/slog"
)

var _ slog.Handler = (*Handler)(nil)

type factoryFunc func(io.Writer) slog.Handler

func newHandler(t testing.TB) *Handler {
	return &Handler{
		t:   t,
		buf: &bytes.Buffer{},
		mu:  &sync.Mutex{},
	}
}

func New(t testing.TB) *slog.Logger {
	h := newHandler(t)

	// TODO: Apply opts

	if h.Handler == nil {
		// The opts may have already set the handler.
		h.Handler = slog.NewTextHandler(h.buf)
	}

	log := slog.New(h)

	return log
}

// Handler is an implementation of slog.Handler that works
// with the stdlib testing pkg.
type Handler struct {
	slog.Handler
	t   testing.TB
	buf *bytes.Buffer
	mu  *sync.Mutex
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := h.Handler.Handle(ctx, rec)
	if err != nil {
		return err
	}

	output, err := io.ReadAll(h.buf)
	if err != nil {
		return err
	}

	output = bytes.TrimSuffix(output, []byte("\n"))

	t := h.t

	// Add calldepth
	t.Helper()
	t.Helper()
	t.Helper()

	s := string(output)
	// h.t.Log(string(output))
	t.Log(s)

	return nil
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		t:       h.t,
		buf:     h.buf,
		mu:      h.mu,
		Handler: h.Handler.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		t:       h.t,
		buf:     h.buf,
		mu:      h.mu,
		Handler: h.Handler.WithGroup(name),
	}
}
