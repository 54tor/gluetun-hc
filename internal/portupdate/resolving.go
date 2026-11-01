package portupdate

import (
	"context"

	"gluetun-hc/internal/serviceaddr"
)

type ResolvingQBittorrentUpdater struct {
	rawURL   string
	username string
	password string
	resolver serviceaddr.Resolver
}

func NewResolvingQBittorrentUpdater(rawURL, username, password string, resolver serviceaddr.Resolver) *ResolvingQBittorrentUpdater {
	return &ResolvingQBittorrentUpdater{rawURL: rawURL, username: username, password: password, resolver: resolver}
}

func (updater *ResolvingQBittorrentUpdater) resolvedUpdater(operationContext context.Context) (*QBittorrentUpdater, error) {
	resolvedURL, resolveError := serviceaddr.RewriteURL(operationContext, updater.rawURL, updater.resolver)
	if resolveError != nil {
		return nil, resolveError
	}
	return NewQBittorrentUpdater(resolvedURL, updater.username, updater.password), nil
}

func (updater *ResolvingQBittorrentUpdater) Apply(operationContext context.Context, port int) error {
	inner, resolveError := updater.resolvedUpdater(operationContext)
	if resolveError != nil {
		return resolveError
	}
	return inner.Apply(operationContext, port)
}

func (updater *ResolvingQBittorrentUpdater) CurrentPort(operationContext context.Context) (int, error) {
	inner, resolveError := updater.resolvedUpdater(operationContext)
	if resolveError != nil {
		return 0, resolveError
	}
	return inner.CurrentPort(operationContext)
}
