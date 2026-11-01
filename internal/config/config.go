package config

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultGrace         = 5 * time.Minute
	defaultInterval      = 10 * time.Second
	defaultThreshold     = 3
	defaultPingTargets   = "9.9.9.9,1.1.1.1"
	defaultPublicIPFile  = "/tmp/gluetun/ip"
	defaultPortFile      = "/tmp/gluetun/forwarded_port"
	defaultControlSource = "auto"
	defaultControlURL    = "http://gluetun:8000"
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
	ControlUsername  string
	ControlPassword  string
	PublicIPFile     string
	PortFile         string
	ControlSource    string
}

func Load(getenv func(string) string) (Config, error) {
	loaded := Config{
		GluetunContainer: getenv("GLUETUN_CONTAINER"),
		InboundContainer: getenv("INBOUND_CONTAINER"),
		ControlAPIKey:    getenv("GLUETUN_API_KEY"),
		ControlUsername:  getenv("GLUETUN_USER"),
		ControlPassword:  getenv("GLUETUN_PASSWORD"),
	}
	if loaded.GluetunContainer == "" {
		return Config{}, errors.New("GLUETUN_CONTAINER est obligatoire")
	}

	if credentialsError := validateControlCredentials(loaded); credentialsError != nil {
		return Config{}, credentialsError
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
	loaded.PublicIPFile = valueOrDefault(getenv("GLUETUN_PUBLIC_IP_FILE"), defaultPublicIPFile)
	loaded.PortFile = valueOrDefault(getenv("GLUETUN_PORT_FILE"), defaultPortFile)
	loaded.ControlSource = valueOrDefault(getenv("GLUETUN_SOURCE"), defaultControlSource)
	if !isKnownControlSource(loaded.ControlSource) {
		return Config{}, errors.New("GLUETUN_SOURCE doit valoir auto, api ou files")
	}
	loaded.ControlURL = valueOrDefault(getenv("GLUETUN_CONTROL_URL"), defaultControlURL)
	return loaded, nil
}
