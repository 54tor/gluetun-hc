package gluetun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxErrorBodyBytes = 200
	maxPort           = 65535
)

var ErrNoForwardedPort = errors.New("aucun port forwardé")

type Credentials struct {
	APIKey   string
	Username string
	Password string
}

type Client struct {
	baseURL     string
	credentials Credentials
	httpClient  *http.Client
}

func NewClient(baseURL string, credentials Credentials, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), credentials: credentials, httpClient: httpClient}
}

func (client *Client) PublicIP(requestContext context.Context) (string, error) {
	var body struct {
		PublicIP string `json:"public_ip"`
	}
	if requestError := client.getJSON(requestContext, "/v1/publicip/ip", &body); requestError != nil {
		return "", requestError
	}
	return body.PublicIP, nil
}

func (client *Client) ForwardedPort(requestContext context.Context) (int, error) {
	var body struct {
		Port int `json:"port"`
	}
	if requestError := client.getJSON(requestContext, "/v1/portforward", &body); requestError != nil {
		return 0, requestError
	}
	if body.Port == 0 {
		return 0, ErrNoForwardedPort
	}
	return checkPortRange(body.Port)
}

func checkPortRange(port int) (int, error) {
	if port < 1 || port > maxPort {
		return 0, fmt.Errorf("port forwardé %d hors de la plage 1-%d", port, maxPort)
	}
	return port, nil
}

func (client *Client) getJSON(requestContext context.Context, path string, destination any) error {
	request, requestError := http.NewRequestWithContext(requestContext, http.MethodGet, client.baseURL+path, nil)
	if requestError != nil {
		return requestError
	}
	client.authenticate(request)
	response, requestError := client.httpClient.Do(request)
	if requestError != nil {
		return requestError
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		return fmt.Errorf("%s : statut %d : %s", path, response.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(response.Body).Decode(destination)
}

func (client *Client) authenticate(request *http.Request) {
	if client.credentials.APIKey != "" {
		request.Header.Set("X-API-Key", client.credentials.APIKey)
		return
	}
	if client.credentials.Username != "" {
		request.SetBasicAuth(client.credentials.Username, client.credentials.Password)
	}
}
