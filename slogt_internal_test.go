package slogt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// newLog applies opt and returns a logger writing to buf. It exercises the
// same handler-construction path as New, minus the testing.TB.Output() sink
// (which cannot be faked because testing.TB has an unexported method).
func newLog(buf io.Writer, opt Option) *slog.Logger {
	c := &config{}
	opt(c)
	if c.newHandler == nil {
		// Mirror New: fall back to the default when opt set no handler.
		getDefault()(c)
	}
	return slog.New(c.newHandler(buf))
}

func TestText_noBogusCallsite(t *testing.T) {
	var buf bytes.Buffer
	newLog(&buf, Text()).Info("hello world")

	got := buf.String()
	// t.Output() prepends no file:line, and Text() defaults to AddSource:false,
	// so the line starts with the slog fields and carries no callsite at all.
	if !strings.HasPrefix(got, "time=") {
		t.Fatalf("text output should begin with time=, not a callsite prefix, got: %q", got)
	}
	if strings.Contains(got, "source=") {
		t.Fatalf("Text() should not emit source=, got: %q", got)
	}
	if !strings.Contains(got, "level=INFO") || !strings.Contains(got, `msg="hello world"`) {
		t.Fatalf("unexpected text output: %q", got)
	}
}

func TestJSON_format(t *testing.T) {
	var buf bytes.Buffer
	newLog(&buf, JSON()).Info("hello world")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json output is not valid JSON: %v; got: %q", err, buf.String())
	}
	if m["msg"] != "hello world" {
		t.Fatalf(`expected msg="hello world", got: %q`, buf.String())
	}
}

func TestFactory_levelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := newLog(&buf, Factory(func(w io.Writer) slog.Handler {
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelError})
	}))

	log.Info("should be filtered out")
	log.Error("should be printed")

	got := buf.String()
	if strings.Contains(got, "should be filtered out") {
		t.Fatalf("info should be filtered at LevelError, got: %q", got)
	}
	if !strings.Contains(got, "should be printed") {
		t.Fatalf("error should be printed, got: %q", got)
	}
}

func TestAddSource_correctCallsite(t *testing.T) {
	var buf bytes.Buffer
	log := newLog(&buf, Factory(func(w io.Writer) slog.Handler {
		return slog.NewTextHandler(w, &slog.HandlerOptions{AddSource: true})
	}))

	// Capture the file and line of the log.Info call below. runtime.Caller(0)
	// is one line above log.Info, so we add 1 to get the actual callsite line.
	_, file, line, _ := runtime.Caller(0)
	log.Info("real callsite")

	got := buf.String()
	if !strings.Contains(got, "source=") {
		t.Fatalf("AddSource output should contain source=, got: %q", got)
	}
	want := fmt.Sprintf("%s:%d", file, line+1)
	if !strings.Contains(got, want) {
		t.Fatalf("source should point at the caller %s, got: %q", want, got)
	}
}

func TestWithAttrs_appearsInOutput(t *testing.T) {
	var buf bytes.Buffer
	newLog(&buf, Text()).With("requestID", "abc-123").Info("handled")
	if got := buf.String(); !strings.Contains(got, "requestID=abc-123") {
		t.Fatalf("With() attrs missing from output: %q", got)
	}
}

func TestText_debugLevelEnabled(t *testing.T) {
	var buf bytes.Buffer
	newLog(&buf, Text()).Debug("debug me")
	if got := buf.String(); !strings.Contains(got, "level=DEBUG") {
		t.Fatalf("Text() should log at LevelDebug, got: %q", got)
	}
}

// TestNew_defaultPath exercises New's real t.Output() sink and the no-option
// default branch. The output can't be captured (testing.TB can't be faked),
// so this asserts that construction and logging don't panic.
func TestNew_defaultPath(t *testing.T) {
	log := New(t) // no options -> getDefault() -> Text()
	if log == nil {
		t.Fatal("New returned nil")
	}
	log.Info("smoke", "key", "value")
}

// TestSetDefault_raceSafe hammers SetDefault and getDefault concurrently; it
// is meaningful under `go test -race`, guarding the package default's access.
func TestSetDefault_raceSafe(t *testing.T) {
	t.Cleanup(func() { SetDefault(Text()) })

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			SetDefault(JSON())
			_ = getDefault()
		})
	}
	wg.Wait()
}
