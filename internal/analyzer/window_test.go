package analyzer

import (
	"github.com/google/uuid"
	"github.com/wyw14/solo-039/internal/model"
	"testing"
)

func TestWindowRejectsGenerationMismatch(t *testing.T) {
	window := NewWindow()
	calibration := uuid.New()
	feedRevision := uuid.New()
	window.Reset(calibration, feedRevision)
	if window.Add(model.Sample{Calibration: uuid.New(), FeedRevision: feedRevision, Valid: true, At: 1}) {
		t.Fatal("accepted another calibration")
	}
}

func TestWindowRejectsAnotherFeedRevision(t *testing.T) {
	window := NewWindow()
	calibration := uuid.New()
	feedRevision := uuid.New()
	window.Reset(calibration, feedRevision)
	sample := model.Sample{Calibration: calibration, FeedRevision: feedRevision, Valid: true, At: 1}
	if !window.Add(sample) {
		t.Fatal("matching sample rejected")
	}
	// A feed switch starts a fresh generation: older samples must not count.
	window.Reset(calibration, uuid.New())
	if window.Complete() {
		t.Fatal("old feed samples were carried into the new generation")
	}
	if window.Add(model.Sample{Calibration: calibration, FeedRevision: feedRevision, Valid: true, At: 2}) {
		t.Fatal("accepted a sample from the previous feed revision")
	}
}
