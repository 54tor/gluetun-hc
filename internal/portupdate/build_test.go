package portupdate

import (
	"context"
	"net/http"
	"testing"

	"gluetun-hc/internal/config"
)

func TestFromConfigQBittorrentCanReadCurrentPort(testCase *testing.T) {
	updater, _ := FromConfig(config.PortUpdate{Mode: "qbittorrent", URL: "http://app:8080"}, http.DefaultClient, nil)
	if _, canRead := updater.(interface {
		CurrentPort(operationContext context.Context) (int, error)
	}); !canRead {
		testCase.Fatal("le profil qBittorrent doit pouvoir lire le port actuel")
	}
}

func TestFromConfigHTTPCannotReadCurrentPort(testCase *testing.T) {
	updater, _ := FromConfig(config.PortUpdate{Mode: "http", URL: "http://app/port", Method: "POST"}, http.DefaultClient, nil)
	if _, canRead := updater.(interface {
		CurrentPort(operationContext context.Context) (int, error)
	}); canRead {
		testCase.Fatal("le mode HTTP générique ne lit pas le port")
	}
}

func TestFromConfigTransmissionCanReadCurrentPort(testCase *testing.T) {
	updater, _ := FromConfig(config.PortUpdate{Mode: "transmission", URL: "http://app:9091"}, http.DefaultClient, nil)
	if _, canRead := updater.(interface {
		CurrentPort(operationContext context.Context) (int, error)
	}); !canRead {
		testCase.Fatal("le profil Transmission doit pouvoir lire le port actuel")
	}
}
