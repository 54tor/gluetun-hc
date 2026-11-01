package portupdate

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/hekmon/transmissionrpc/v3"
)

var errNoPeerPort = errors.New("Transmission n'a pas renvoyé peer-port")

const defaultTransmissionRPCPath = "/transmission/rpc"

type TransmissionUpdater struct {
	client *transmissionrpc.Client
}

func transmissionEndpoint(rawURL, username, password string) *url.URL {
	endpoint, parseError := url.Parse(rawURL)
	if parseError != nil {
		endpoint = &url.URL{Scheme: "http", Host: rawURL}
	}
	if endpoint.Path == "" || endpoint.Path == "/" {
		endpoint.Path = defaultTransmissionRPCPath
	}
	if username != "" {
		endpoint.User = url.UserPassword(username, password)
	}
	return endpoint
}

func NewTransmissionUpdater(rawURL, username, password string, httpClient *http.Client) (*TransmissionUpdater, error) {
	var extra *transmissionrpc.Config
	if httpClient != nil {
		extra = &transmissionrpc.Config{CustomClient: httpClient}
	}
	client, clientError := transmissionrpc.New(transmissionEndpoint(rawURL, username, password), extra)
	if clientError != nil {
		return nil, clientError
	}
	return &TransmissionUpdater{client: client}, nil
}

func (updater *TransmissionUpdater) Apply(operationContext context.Context, port int) error {
	peerPort := int64(port)
	return updater.client.SessionArgumentsSet(operationContext, transmissionrpc.SessionArguments{PeerPort: &peerPort})
}

func (updater *TransmissionUpdater) CurrentPort(operationContext context.Context) (int, error) {
	session, readError := updater.client.SessionArgumentsGet(operationContext, []string{"peer-port"})
	if readError != nil {
		return 0, readError
	}
	if session.PeerPort == nil {
		return 0, errNoPeerPort
	}
	return int(*session.PeerPort), nil
}
