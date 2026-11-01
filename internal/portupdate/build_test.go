package portupdate

import (
	"context"
	"net/http"
	"testing"

	"gluetun-hc/internal/config"
)

func TestFromConfigQBittorrentCanReadCurrentPort(testCase *testing.T) {
	updater := FromConfig(config.PortUpdate{Mode: "qbittorrent", URL: "http://app:8080"}, http.DefaultClient)
	if _, canRead := updater.(interface {
		CurrentPort(operationContext context.Context) (int, error)
	}); !canRead {
		testCase.Fatal("le profil qBittorrent doit pouvoir lire le port actuel")
	}
}

func TestFromConfigHTTPCannotReadCurrentPort(testCase *testing.T) {
	updater := FromConfig(config.PortUpdate{Mode: "http", URL: "http://app/port", Method: "POST"}, http.DefaultClient)
	if _, canRead := updater.(interface {
		CurrentPort(operationContext context.Context) (int, error)
	}); canRead {
		testCase.Fatal("le mode HTTP générique ne lit pas le port")
	}
}
