package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	loaded.PingTargets = splitList(valueOrDefault(getenv("PING_TARGETS"), defaultPingTargets))
	loaded.ControlURL = valueOrDefault(getenv("GLUETUN_CONTROL_URL"), defaultControlURL)
	return loaded, nil
}

func parseWatched(rawList string, defaultGraceDuration time.Duration) ([]Watched, error) {
	var watched []Watched
	for _, entry := range splitList(rawList) {
		name, rawGrace, hasGrace := strings.Cut(entry, ":")
		grace := defaultGraceDuration
		if hasGrace {
			parsedGrace, parseError := time.ParseDuration(rawGrace)
			if parseError != nil {
				return nil, fmt.Errorf("WATCHED %q : %w", entry, parseError)
			}
			grace = parsedGrace
		}
		watched = append(watched, Watched{Name: name, Grace: grace})
	}
	if len(watched) == 0 {
		return nil, errors.New("WATCHED doit lister au moins un conteneur")
	}
	return watched, nil
}

func isWatched(watched []Watched, name string) bool {
	for _, candidate := range watched {
		if candidate.Name == name {
			return true
		}
	}
	return false
}

func splitList(rawList string) []string {
	var items []string
	for _, item := range strings.Split(rawList, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func durationOrDefault(raw string, fallback time.Duration) (time.Duration, error) {
	if raw == "" {
		return fallback, nil
	}
	return time.ParseDuration(raw)
}

func positiveIntOrDefault(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	parsed, parseError := strconv.Atoi(raw)
	if parseError != nil || parsed < 1 {
		return 0, errors.New("entier strictement positif attendu")
	}
	return parsed, nil
}
