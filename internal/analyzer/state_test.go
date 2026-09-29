package analyzer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/wyw14/solo-039/internal/model"
)

func stableSamples(calibration, revision uuid.UUID) []model.Sample {
	at := []int64{10, 20, 30}
	values := []float64{1.0, 1.01, 0.99}
	samples := make([]model.Sample, 0, len(at))
	for i := range at {
		samples = append(samples, model.Sample{
			ID:           uuid.New(),
			FeedRevision: revision,
			Calibration:  calibration,
			Value:        values[i],
			At:           at[i],
			Component:    "top",
			Valid:        true,
		})
	}
	return samples
}

func fillWindow(t *testing.T, state *State, calibration, revision uuid.UUID) {
	t.Helper()
	state.window.Reset(calibration, revision)
	for _, sample := range stableSamples(calibration, revision) {
		if !state.Add(sample) {
			t.Fatalf("sample for feed %s / calibration %s was rejected", revision, calibration)
		}
	}
}

// The analyzer only holds samples from the old feed: a proof requested with a
// newer revision must never be minted, even though the window is complete.
func TestProofRejectsForeignFeedRevision(t *testing.T) {
	state := New()
	oldRevision := uuid.New()
	calibration := state.Calibration()
	fillWindow(t, state, calibration, oldRevision)

	if _, ok := state.Proof(uuid.New()); ok {
		t.Fatal("proof minted for a feed revision the samples were not collected under")
	}

	proof, ok := state.Proof(oldRevision)
	if !ok {
		t.Fatal("same-generation proof should still be available")
	}
	if proof.Revision != oldRevision {
		t.Fatalf("proof stamped with revision %s, want %s", proof.Revision, oldRevision)
	}
}

// Recalibration starts a new generation and clears the window: samples taken
// under the old calibration cannot back a proof after the calibration advanced.
func TestProofRejectsStaleCalibration(t *testing.T) {
	state := New()
	revision := uuid.New()
	oldCalibration := state.Calibration()
	fillWindow(t, state, oldCalibration, revision)

	newCalibration := state.Calibrate()
	state.BindFeed(revision)

	if _, ok := state.Proof(revision); ok {
		t.Fatal("proof minted after the sampling calibration was superseded")
	}

	for _, sample := range stableSamples(newCalibration, revision) {
		if !state.Add(sample) {
			t.Fatal("new-calibration sample rejected after rebind")
		}
	}
	if _, ok := state.Proof(revision); !ok {
		t.Fatal("same-generation samples should produce a proof after recalibration")
	}
}

// Switching feed clears the window, and only samples of the new generation can
// complete it again.
func TestProofAfterFeedRebind(t *testing.T) {
	state := New()
	oldRevision := uuid.New()
	calibration := state.Calibration()
	fillWindow(t, state, calibration, oldRevision)

	newRevision := uuid.New()
	state.BindFeed(newRevision)
	if _, ok := state.Proof(oldRevision); ok {
		t.Fatal("old-generation proof survived the feed switch")
	}
	if _, ok := state.Proof(newRevision); ok {
		t.Fatal("proof minted before any new-generation sample was collected")
	}

	fillWindow(t, state, calibration, newRevision)
	if _, ok := state.Proof(newRevision); !ok {
		t.Fatal("same-generation samples should produce a proof after the feed switch")
	}
}

func TestProofRejectsIncompleteWindow(t *testing.T) {
	state := New()
	revision := uuid.New()
	calibration := state.Calibration()
	state.window.Reset(calibration, revision)
	samples := stableSamples(calibration, revision)
	if !state.Add(samples[0]) {
		t.Fatal("first sample rejected")
	}
	if _, ok := state.Proof(revision); ok {
		t.Fatal("proof minted from an incomplete sampling window")
	}
}
