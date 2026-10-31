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

	var err error
	defaultGraceDuration, err := durationOrDefault(getenv("GRACE"), defaultGrace)
	if err != nil {
		return Config{}, fmt.Errorf("GRACE : %w", err)
	}
	if loaded.GluetunGrace, err = durationOrDefault(getenv("GLUETUN_GRACE"), defaultGraceDuration); err != nil {
		return Config{}, fmt.Errorf("GLUETUN_GRACE : %w", err)
	}
	if loaded.Interval, err = durationOrDefault(getenv("INTERVAL"), defaultInterval); err != nil {
		return Config{}, fmt.Errorf("INTERVAL : %w", err)
	}
	if loaded.FailThreshold, err = positiveIntOrDefault(getenv("FAIL_THRESHOLD"), defaultThreshold); err != nil {
		return Config{}, fmt.Errorf("FAIL_THRESHOLD : %w", err)
	}
	if loaded.RiseThreshold, err = positiveIntOrDefault(getenv("RISE_THRESHOLD"), defaultThreshold); err != nil {
		return Config{}, fmt.Errorf("RISE_THRESHOLD : %w", err)
	}
	if loaded.Watched, err = parseWatched(getenv("WATCHED"), defaultGraceDuration); err != nil {
		return Config{}, err
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
			parsedGrace, err := time.ParseDuration(rawGrace)
			if err != nil {
				return nil, fmt.Errorf("WATCHED %q : %w", entry, err)
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
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 {
		return 0, errors.New("entier strictement positif attendu")
	}
	return parsed, nil
}
