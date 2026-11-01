package portupdate

import (
	"context"
	"net/http"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/serviceaddr"
)

type Updater interface {
	Apply(operationContext context.Context, port int) error
}

func FromConfig(update config.PortUpdate, httpClient *http.Client, resolver serviceaddr.Resolver) (Updater, error) {
	switch update.Mode {
	case "qbittorrent":
		if resolver == nil {
			return NewQBittorrentUpdater(update.URL, update.Username, update.Password), nil
		}
		return NewResolvingQBittorrentUpdater(update.URL, update.Username, update.Password, resolver), nil
	case "transmission":
		return NewTransmissionUpdater(update.URL, update.Username, update.Password, httpClient)
	default:
		return NewHTTPUpdater(update.Method, update.URL, update.Headers, update.Body, httpClient), nil
	}
}
