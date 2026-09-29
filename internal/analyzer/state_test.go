package analyzer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/wyw14/solo-039/internal/model"
)

func fillWindow(t *testing.T, state *State, feed, calibration uuid.UUID) {
	t.Helper()
	for i := int64(1); i <= 3; i++ {
		sample := model.Sample{ID: uuid.New(), FeedRevision: feed, Calibration: calibration, Value: 0.5, At: i, Valid: true}
		if !state.Add(sample) {
			t.Fatalf("sample %d rejected for the current generation", i)
		}
	}
}

// The analyzer only collected samples under the old feed revision. Asking for a
// proof stamped with a newer revision must not succeed: previous-generation
// samples cannot be lent to a new batch.
func TestProofRejectsAnotherFeedRevision(t *testing.T) {
	state := New()
	oldFeed := uuid.New()
	state.BindFeed(oldFeed)
	fillWindow(t, state, oldFeed, state.Calibration())

	if _, ok := state.Proof(uuid.New()); ok {
		t.Fatal("old-generation samples backed a proof for a newer revision")
	}

	proof, ok := state.Proof(oldFeed)
	if !ok {
		t.Fatal("same-generation normal samples failed to produce a proof")
	}
	if proof.Revision != oldFeed {
		t.Fatal("proof was stamped with a different revision")
	}
}

// Switching feed resets the sampling window, so neither the new nor the old
// revision can claim a proof until fresh samples for the new generation arrive.
func TestProofUnavailableAfterFeedSwitch(t *testing.T) {
	state := New()
	oldFeed := uuid.New()
	state.BindFeed(oldFeed)
	fillWindow(t, state, oldFeed, state.Calibration())

	newFeed := uuid.New()
	state.BindFeed(newFeed)

	if _, ok := state.Proof(newFeed); ok {
		t.Fatal("new batch received a proof without any new-generation samples")
	}
	if _, ok := state.Proof(oldFeed); ok {
		t.Fatal("old samples remained usable after the feed switch")
	}
}

// Recalibration starts a new calibration generation: earlier samples and proofs
// are invalidated, and samples stamped with the old calibration are rejected.
func TestProofUnavailableAfterRecalibration(t *testing.T) {
	state := New()
	feed := uuid.New()
	state.BindFeed(feed)
	oldCalibration := state.Calibration()
	fillWindow(t, state, feed, oldCalibration)
	if _, ok := state.Proof(feed); !ok {
		t.Fatal("same-generation proof expected before recalibration")
	}

	newCalibration := state.Calibrate()
	state.BindFeed(feed)
	if newCalibration == oldCalibration {
		t.Fatal("recalibration did not advance the calibration revision")
	}
	if _, ok := state.Proof(feed); ok {
		t.Fatal("pre-calibration samples backed a proof after recalibration")
	}
	if state.Add(model.Sample{ID: uuid.New(), FeedRevision: feed, Calibration: oldCalibration, Value: 0.5, At: 4, Valid: true}) {
		t.Fatal("a sample stamped with the old calibration was accepted")
	}
}
