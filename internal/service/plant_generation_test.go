package service

import (
	"context"
	"strings"
	"testing"
)

func stableSamples(t *testing.T, plant *Plant) {
	t.Helper()
	for at := int64(1); at <= 3; at++ {
		if _, err := plant.AddSample(0.5, at); err != nil {
			t.Fatalf("sample %d rejected for the active batch: %v", at, err)
		}
	}
}

// After a feed/batch switch, samples collected for the previous batch must not
// be lent to the new revision: routing has to fail until fresh samples for the
// new generation are collected.
func TestPlantProofRejectsSamplesFromPreviousBatch(t *testing.T) {
	plant := NewPlant(t.TempDir())
	ctx := context.Background()
	if _, err := plant.Start(ctx, "baseline", "primary"); err != nil {
		t.Fatal(err)
	}
	stableSamples(t, plant)

	if _, err := plant.Transition(ctx, "premium", "secondary"); err != nil {
		t.Fatal(err)
	}
	if err := plant.RouteProduct("T-202"); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("new batch accepted a proof backed by old-feed samples: %v", err)
	}

	// Same-generation samples still certify the batch.
	stableSamples(t, plant)
	if err := plant.RouteProduct("T-202"); err != nil {
		t.Fatalf("same-generation samples failed to produce a routable proof: %v", err)
	}
}

// Recalibration starts a new analyzer generation; pre-calibration samples must
// not back a proof afterwards.
func TestPlantRecalibrationInvalidatesSamples(t *testing.T) {
	plant := NewPlant(t.TempDir())
	ctx := context.Background()
	if _, err := plant.Start(ctx, "baseline", "primary"); err != nil {
		t.Fatal(err)
	}
	stableSamples(t, plant)

	plant.CalibrateAnalyzer()
	if err := plant.RouteProduct("T-202"); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("pre-calibration samples backed a proof after recalibration: %v", err)
	}

	stableSamples(t, plant)
	if err := plant.RouteProduct("T-202"); err != nil {
		t.Fatalf("post-calibration samples failed to produce a proof: %v", err)
	}
}
