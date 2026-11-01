package portupdate

import (
	"context"

	"github.com/autobrr/go-qbittorrent"
)

const qbittorrentTimeoutSeconds = 5

type QBittorrentUpdater struct {
	client *qbittorrent.Client
}

func NewQBittorrentUpdater(baseURL, username, password string) *QBittorrentUpdater {
	client := qbittorrent.NewClient(qbittorrent.Config{
		Host:     baseURL,
		Username: username,
		Password: password,
		Timeout:  qbittorrentTimeoutSeconds,
	})
	return &QBittorrentUpdater{client: client}
}

func (updater *QBittorrentUpdater) Apply(operationContext context.Context, port int) error {
	if loginError := updater.client.LoginCtx(operationContext); loginError != nil {
		return loginError
	}
	return updater.client.SetPreferencesCtx(operationContext, map[string]interface{}{"listen_port": port})
}

func (updater *QBittorrentUpdater) CurrentPort(operationContext context.Context) (int, error) {
	if loginError := updater.client.LoginCtx(operationContext); loginError != nil {
		return 0, loginError
	}
	preferences, readError := updater.client.GetAppPreferencesCtx(operationContext)
	if readError != nil {
		return 0, readError
	}
	return preferences.ListenPort, nil
}
