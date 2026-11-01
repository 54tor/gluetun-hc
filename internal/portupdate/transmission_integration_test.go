package portupdate

import (
	"context"
	"os"
	"testing"
)

func TestIntegrationTransmissionRoundTrip(testCase *testing.T) {
	baseURL := os.Getenv("TRANSMISSION_URL")
	if baseURL == "" {
		testCase.Skip("TRANSMISSION_URL non défini")
	}
	updater, buildError := NewTransmissionUpdater(baseURL, os.Getenv("TRANSMISSION_USER"), os.Getenv("TRANSMISSION_PASSWORD"))
	if buildError != nil {
		testCase.Fatal(buildError)
	}

	const wantedPort = 54322
	if applyError := updater.Apply(context.Background(), wantedPort); applyError != nil {
		testCase.Fatal(applyError)
	}
	obtainedPort, readError := updater.CurrentPort(context.Background())
	if readError != nil || obtainedPort != wantedPort {
		testCase.Fatalf("port lu %d, erreur %v", obtainedPort, readError)
	}
}
