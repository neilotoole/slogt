package slogt_test

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/neilotoole/slogt"

	"golang.org/x/exp/slog"
)

const (
	iter  = 3
	sleep = time.Millisecond * 20
)

// TestSlog_Ugly demonstrates that testing output is ugly, because
// the slog.Logger output is not tied to the testing.T.
func TestSlog_Ugly(t *testing.T) {
	for i := 0; i < iter; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			handler := slog.NewTextHandler(os.Stdout)
			log := slog.New(handler)

			for j := 0; j < iter; j++ {
				t.Log("YAY: this is indented as expected.")
				log := log.With("count", j)
				log.Info("BOO: This, alas, is not indented.")

				// Sleep a little to allow the goroutines to interleave.
				time.Sleep(sleep)
			}
		})
	}
}

// TestSlogt_Pretty demonstrates use of slog with testing.T.
func TestSlogt_Pretty(t *testing.T) {
	for i := 0; i < iter; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			log := slogt.New(t)
			for j := 0; j < iter; j++ {
				t.Log("testing.T: this is indented as expected.")
				log.Info("slogt: hello")

				log2 := log.With("count", j)
				log2.Info("slogt: hello with attrs")

				// Sleep a little to allow the goroutines to interleave.
				time.Sleep(sleep)
			}
		})
	}
}
