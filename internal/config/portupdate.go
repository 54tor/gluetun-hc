package config

import (
	"errors"
	"fmt"
	"strings"
)

const defaultPortUpdateMethod = "POST"

type PortUpdate struct {
	Mode      string
	URL       string
	Username  string
	Password  string
	Method    string
	Headers   map[string]string
	Body      string
	Container string
}

func loadPortUpdate(getenv func(string) string, inboundContainer string, watched []Watched) (PortUpdate, error) {
	update := PortUpdate{Mode: getenv("PORT_UPDATE_MODE")}
	if update.Mode == "" {
		return update, nil
	}
	if update.Mode != "qbittorrent" && update.Mode != "http" {
		return PortUpdate{}, errors.New("PORT_UPDATE_MODE doit valoir qbittorrent ou http")
	}

	update.URL = getenv("PORT_UPDATE_URL")
	if update.URL == "" {
		return PortUpdate{}, errors.New("PORT_UPDATE_URL est obligatoire quand PORT_UPDATE_MODE est défini")
	}
	update.Container = valueOrDefault(getenv("PORT_UPDATE_CONTAINER"), inboundContainer)
	if update.Container == "" || !isWatched(watched, update.Container) {
		return PortUpdate{}, errors.New("PORT_UPDATE_CONTAINER doit figurer dans WATCHED")
	}

	update.Username = getenv("PORT_UPDATE_USER")
	update.Password = getenv("PORT_UPDATE_PASSWORD")
	update.Method = valueOrDefault(getenv("PORT_UPDATE_METHOD"), defaultPortUpdateMethod)
	update.Body = getenv("PORT_UPDATE_BODY")

	headers, headersError := parseHeaders(getenv("PORT_UPDATE_HEADERS"))
	if headersError != nil {
		return PortUpdate{}, headersError
	}
	update.Headers = headers
	return update, nil
}

func parseHeaders(rawHeaders string) (map[string]string, error) {
	headers := map[string]string{}
	for _, line := range strings.Split(rawHeaders, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		name, value, found := strings.Cut(trimmed, ":")
		if !found || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("PORT_UPDATE_HEADERS : ligne invalide %q (attendu « Nom: valeur »)", trimmed)
		}
		headers[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return headers, nil
}
