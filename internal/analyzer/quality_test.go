package analyzer

import (
	"testing"

	"github.com/google/uuid"
)

// The proof interface must not certify a revision other than the one backing
// the current sampling window, even when the samples themselves pass quality.
func TestQualityProofRejectsAnotherGeneration(t *testing.T) {
	state := New()
	feed := uuid.New()
	state.BindFeed(feed)
	fillWindow(t, state, feed, state.Calibration())

	quality := DefaultQuality()
	if _, err := quality.Proof(state, uuid.New().String()); err == nil {
		t.Fatal("quality proof accepted a newer revision backed by old samples")
	}
	if _, err := quality.Proof(state, feed.String()); err != nil {
		t.Fatalf("same-generation quality proof failed: %v", err)
	}
}
