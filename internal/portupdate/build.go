package portupdate

import (
	"context"
	"net/http"

	"gluetun-hc/internal/config"
)

type Updater interface {
	Apply(operationContext context.Context, port int) error
}

func FromConfig(update config.PortUpdate, httpClient *http.Client) Updater {
	if update.Mode == "qbittorrent" {
		return NewQBittorrentUpdater(update.URL, update.Username, update.Password)
	}
	return NewHTTPUpdater(update.Method, update.URL, update.Headers, update.Body, httpClient)
}
