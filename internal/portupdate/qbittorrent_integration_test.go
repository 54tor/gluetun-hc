package portupdate

import (
	"context"
	"os"
	"testing"
)

func TestIntegrationQBittorrentRoundTrip(testCase *testing.T) {
	baseURL := os.Getenv("QBITTORRENT_URL")
	if baseURL == "" {
		testCase.Skip("QBITTORRENT_URL non défini")
	}
	updater := NewQBittorrentUpdater(baseURL, os.Getenv("QBITTORRENT_USER"), os.Getenv("QBITTORRENT_PASSWORD"))

	const wantedPort = 54321
	if applyError := updater.Apply(context.Background(), wantedPort); applyError != nil {
		testCase.Fatal(applyError)
	}
	obtainedPort, readError := updater.CurrentPort(context.Background())
	if readError != nil || obtainedPort != wantedPort {
		testCase.Fatalf("port lu %d, erreur %v", obtainedPort, readError)
	}
}
