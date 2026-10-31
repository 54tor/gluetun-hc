package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

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

func validateControlCredentials(loaded Config) error {
	hasBasicCredentials := loaded.ControlUsername != "" || loaded.ControlPassword != ""
	if hasBasicCredentials && (loaded.ControlUsername == "" || loaded.ControlPassword == "") {
		return errors.New("GLUETUN_USER et GLUETUN_PASSWORD vont ensemble")
	}
	if hasBasicCredentials && loaded.ControlAPIKey != "" {
		return errors.New("GLUETUN_API_KEY et GLUETUN_USER/GLUETUN_PASSWORD s'excluent")
	}
	return nil
}
