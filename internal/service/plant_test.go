package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func addStableSamples(t *testing.T, plant *Plant, base int64) {
	t.Helper()
	for i, value := range []float64{1.0, 1.01, 0.99} {
		if _, err := plant.AddSample(value, base+int64(i)); err != nil {
			t.Fatalf("sample %d rejected: %v", i, err)
		}
	}
}

// The analyzer only collected samples under the old feed. After switching to a
// revised feed, requesting product certification must fail instead of stamping
// the old sampling window with the new revision. New same-generation samples
// must then be able to certify the new batch.
func TestOldSamplesDoNotCertifyNewFeedRevision(t *testing.T) {
	ctx := context.Background()
	plant := NewPlant(t.TempDir())

	start, err := plant.Start(ctx, "grade-a", "feed-A")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	oldRevision := start.Revision
	base := time.Now().Unix()
	addStableSamples(t, plant, base)

	if proof, ok := plant.Analyzer.Proof(oldRevision); !ok || proof.Revision != oldRevision {
		t.Fatal("same-generation proof should be available before the feed switch")
	}

	transitioned, err := plant.Transition(ctx, "grade-b", "feed-B")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	newRevision := transitioned.Revision
	if newRevision == oldRevision {
		t.Fatal("feed switch did not advance the revision")
	}

	if _, ok := plant.Analyzer.Proof(newRevision); ok {
		t.Fatal("old-generation samples certified the new feed revision")
	}
	if plant.Analyzer.SampleCount() != 0 {
		t.Fatalf("sampling window retained %d old-generation samples after the feed switch", plant.Analyzer.SampleCount())
	}

	err = plant.RouteProduct("tank-1")
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("RouteProduct with an empty new-generation window, err = %v", err)
	}

	addStableSamples(t, plant, base+10)
	if err := plant.RouteProduct("tank-1"); err != nil {
		t.Fatalf("same-generation samples should certify the new batch: %v", err)
	}
}

// Recalibration opens a new generation too: samples taken under the previous
// calibration cannot certify the batch until fresh samples are collected.
func TestOldSamplesDoNotCertifyAfterRecalibration(t *testing.T) {
	ctx := context.Background()
	plant := NewPlant(t.TempDir())

	current, err := plant.Start(ctx, "grade-a", "feed-A")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	addStableSamples(t, plant, time.Now().Unix())

	plant.CalibrateAnalyzer()

	if _, ok := plant.Analyzer.Proof(current.Revision); ok {
		t.Fatal("pre-calibration samples certified the batch after recalibration")
	}
	if err := plant.RouteProduct("tank-1"); err == nil {
		t.Fatal("RouteProduct succeeded with only pre-calibration samples")
	}

	addStableSamples(t, plant, time.Now().Unix()+10)
	if err := plant.RouteProduct("tank-1"); err != nil {
		t.Fatalf("post-calibration samples should certify the batch: %v", err)
	}
}
