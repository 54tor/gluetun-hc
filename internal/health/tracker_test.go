package health

import "testing"

func recordAll(tracker *Tracker, results ...bool) {
	for _, result := range results {
		tracker.Record(result)
	}
}

func TestStaysHealthyBelowFailThreshold(testCase *testing.T) {
	tracker := NewTracker(3, 3)
	recordAll(tracker, false, false)
	if !tracker.Healthy() {
		testCase.Fatal("doit rester sain")
	}
}

func TestTurnsUnhealthyAtFailThreshold(testCase *testing.T) {
	tracker := NewTracker(3, 3)
	recordAll(tracker, false, false, false)
	if tracker.Healthy() {
		testCase.Fatal("ne doit plus être sain")
	}
}

func TestRecoversAfterRiseThreshold(testCase *testing.T) {
	tracker := NewTracker(3, 3)
	recordAll(tracker, false, false, false, true, true)
	if tracker.Healthy() {
		testCase.Fatal("deux succès ne suffisent pas")
	}
	tracker.Record(true)
	if !tracker.Healthy() {
		testCase.Fatal("doit être sain après trois succès")
	}
}

func TestSuccessResetsFailureStreak(testCase *testing.T) {
	tracker := NewTracker(3, 3)
	recordAll(tracker, false, false, true, false, false)
	if !tracker.Healthy() {
		testCase.Fatal("la série d'échecs doit être remise à zéro")
	}
}

func TestResetRestoresHealthy(testCase *testing.T) {
	tracker := NewTracker(3, 3)
	recordAll(tracker, false, false, false)
	tracker.Reset()
	if !tracker.Healthy() {
		testCase.Fatal("Reset doit rétablir l'état sain")
	}
}
