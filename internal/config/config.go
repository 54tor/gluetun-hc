package config

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultGrace       = 5 * time.Minute
	defaultInterval    = 10 * time.Second
	defaultThreshold   = 3
	defaultPingTargets = "9.9.9.9,1.1.1.1"
	defaultControlURL  = "http://gluetun:8000"
)

type Watched struct {
	Name  string
	Grace time.Duration
}

type Config struct {
	GluetunContainer string
	GluetunGrace     time.Duration
	Watched          []Watched
	InboundContainer string
	PingTargets      []string
	Interval         time.Duration
	FailThreshold    int
	RiseThreshold    int
	ControlURL       string
	ControlAPIKey    string
}

func Load(getenv func(string) string) (Config, error) {
	loaded := Config{
		GluetunContainer: getenv("GLUETUN_CONTAINER"),
		InboundContainer: getenv("INBOUND_CONTAINER"),
		ControlAPIKey:    getenv("GLUETUN_API_KEY"),
	}
	if loaded.GluetunContainer == "" {
		return Config{}, errors.New("GLUETUN_CONTAINER est obligatoire")
	}

	var parseError error
	defaultGraceDuration, parseError := durationOrDefault(getenv("GRACE"), defaultGrace)
	if parseError != nil {
		return Config{}, fmt.Errorf("GRACE : %w", parseError)
	}
	if loaded.GluetunGrace, parseError = durationOrDefault(getenv("GLUETUN_GRACE"), defaultGraceDuration); parseError != nil {
		return Config{}, fmt.Errorf("GLUETUN_GRACE : %w", parseError)
	}
	if loaded.Interval, parseError = durationOrDefault(getenv("INTERVAL"), defaultInterval); parseError != nil {
		return Config{}, fmt.Errorf("INTERVAL : %w", parseError)
	}
	if loaded.FailThreshold, parseError = positiveIntOrDefault(getenv("FAIL_THRESHOLD"), defaultThreshold); parseError != nil {
		return Config{}, fmt.Errorf("FAIL_THRESHOLD : %w", parseError)
	}
	if loaded.RiseThreshold, parseError = positiveIntOrDefault(getenv("RISE_THRESHOLD"), defaultThreshold); parseError != nil {
		return Config{}, fmt.Errorf("RISE_THRESHOLD : %w", parseError)
	}
	if loaded.Watched, parseError = parseWatched(getenv("WATCHED"), defaultGraceDuration); parseError != nil {
		return Config{}, parseError
	}

	if loaded.InboundContainer != "" && !isWatched(loaded.Watched, loaded.InboundContainer) {
		return Config{}, errors.New("INBOUND_CONTAINER doit figurer dans WATCHED")
	}

	if loaded.Interval <= 0 {
		return Config{}, errors.New("INTERVAL doit être strictement positif")
	}

	loaded.PingTargets = splitList(valueOrDefault(getenv("PING_TARGETS"), defaultPingTargets))
	if len(loaded.PingTargets) == 0 {
		return Config{}, errors.New("PING_TARGETS doit lister au moins une cible")
	}
	loaded.ControlURL = valueOrDefault(getenv("GLUETUN_CONTROL_URL"), defaultControlURL)
	return loaded, nil
}
