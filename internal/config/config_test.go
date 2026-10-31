package config

import (
	"testing"
	"time"
)

func environmentFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(testCase *testing.T) {
	loaded, err := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
	}))
	if err != nil {
		testCase.Fatal(err)
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
	loaded, err := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app:10m,other",
	}))
	if err != nil {
		testCase.Fatal(err)
	}
	if loaded.Watched[0].Name != "app" || loaded.Watched[0].Grace != 10*time.Minute {
		testCase.Fatalf("premier conteneur : %+v", loaded.Watched[0])
	}
	if loaded.Watched[1].Name != "other" || loaded.Watched[1].Grace != 5*time.Minute {
		testCase.Fatalf("second conteneur : %+v", loaded.Watched[1])
	}
}

func TestLoadRejectsMissingGluetun(testCase *testing.T) {
	_, err := Load(environmentFrom(map[string]string{"WATCHED": "app"}))
	if err == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsEmptyWatched(testCase *testing.T) {
	_, err := Load(environmentFrom(map[string]string{"GLUETUN_CONTAINER": "gluetun"}))
	if err == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestLoadRejectsBadDuration(testCase *testing.T) {
	_, err := Load(environmentFrom(map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app:abc",
	}))
	if err == nil {
		testCase.Fatal("erreur attendue")
	}
}
