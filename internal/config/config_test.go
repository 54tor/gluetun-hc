package config

import (
	"testing"
	"time"
)

func environmentFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(testCase *testing.T) {
	loaded, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
	}))
	if loadError != nil {
		testCase.Fatal(loadError)
	}
	if len(loaded.PingTargets) != 2 || loaded.PingTargets[0] != "9.9.9.9" || loaded.PingTargets[1] != "1.1.1.1" {
		testCase.Fatalf("cibles de ping inattendues : %v", loaded.PingTargets)
	}
	if loaded.Interval != 10*time.Second {
		testCase.Fatalf("intervalle : %v", loaded.Interval)
	}
	if loaded.FailThreshold != 3 || loaded.RiseThreshold != 3 {
		testCase.Fatalf("seuils : %d %d", loaded.FailThreshold, loaded.RiseThreshold)
	}
	if loaded.Watched[0].Grace != 5*time.Minute || loaded.GluetunGrace != 5*time.Minute {
		testCase.Fatalf("délais de grâce : %v %v", loaded.Watched[0].Grace, loaded.GluetunGrace)
	}
}

func TestLoadPerContainerGrace(testCase *testing.T) {
	loaded, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app:10m,other",
	}))
	if loadError != nil {
		testCase.Fatal(loadError)
	}
	if loaded.Watched[0].Name != "app" || loaded.Watched[0].Grace != 10*time.Minute {
		testCase.Fatalf("premier conteneur : %+v", loaded.Watched[0])
	}
	if loaded.Watched[1].Name != "other" || loaded.Watched[1].Grace != 5*time.Minute {
		testCase.Fatalf("second conteneur : %+v", loaded.Watched[1])
	}
}

func TestLoadRejectsMissingGluetun(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{"WATCHED": "app"}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsEmptyWatched(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{"GLUETUN_CONTAINER": "gluetun"}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsBadDuration(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app:abc",
	}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsInboundContainerNotWatched(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
		"INBOUND_CONTAINER": "unknown",
	}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsNonPositiveInterval(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
		"INTERVAL":          "0s",
	}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsEmptyPingTargets(testCase *testing.T) {
	_, loadError := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
		"PING_TARGETS":      ",",
	}))
	if loadError == nil {
		testCase.Fatal("erreur attendue")
	}
}
