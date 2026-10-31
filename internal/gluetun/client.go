package gluetun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var ErrNoForwardedPort = errors.New("aucun port forwardé")

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, httpClient: httpClient}
}

func (client *Client) PublicIP(ctx context.Context) (string, error) {
	var body struct {
		PublicIP string `json:"public_ip"`
	}
	if err := client.getJSON(ctx, "/v1/publicip/ip", &body); err != nil {
		return "", err
	}
	return body.PublicIP, nil
}

func (client *Client) ForwardedPort(ctx context.Context) (int, error) {
	var body struct {
		Port int `json:"port"`
	}
	if err := client.getJSON(ctx, "/v1/portforward", &body); err != nil {
		return 0, err
	}
	if body.Port == 0 {
		return 0, ErrNoForwardedPort
	}
	return body.Port, nil
}

func (client *Client) getJSON(ctx context.Context, path string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return err
	}
	if client.apiKey != "" {
		request.Header.Set("X-API-Key", client.apiKey)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s : statut %d", path, response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(destination)
}
