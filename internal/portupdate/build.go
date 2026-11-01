package portupdate

import (
	"context"
	"net/http"

	"gluetun-hc/internal/config"
)

type Updater interface {
	Apply(operationContext context.Context, port int) error
}

func FromConfig(update config.PortUpdate, httpClient *http.Client) (Updater, error) {
	switch update.Mode {
	case "qbittorrent":
		return NewQBittorrentUpdater(update.URL, update.Username, update.Password), nil
	case "transmission":
		return NewTransmissionUpdater(update.URL, update.Username, update.Password)
	default:
		return NewHTTPUpdater(update.Method, update.URL, update.Headers, update.Body, httpClient), nil
	}
}
