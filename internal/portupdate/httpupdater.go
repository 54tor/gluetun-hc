package portupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const portPlaceholder = "{{port}}"

type HTTPUpdater struct {
	method     string
	url        string
	headers    map[string]string
	body       string
	httpClient *http.Client
}

func NewHTTPUpdater(method, url string, headers map[string]string, body string, httpClient *http.Client) *HTTPUpdater {
	return &HTTPUpdater{method: method, url: url, headers: headers, body: body, httpClient: httpClient}
}

func (updater *HTTPUpdater) Apply(operationContext context.Context, port int) error {
	portText := strconv.Itoa(port)
	request, requestError := http.NewRequestWithContext(operationContext, updater.method,
		strings.ReplaceAll(updater.url, portPlaceholder, portText),
		strings.NewReader(strings.ReplaceAll(updater.body, portPlaceholder, portText)))
	if requestError != nil {
		return requestError
	}
	for name, value := range updater.headers {
		request.Header.Set(name, strings.ReplaceAll(value, portPlaceholder, portText))
	}

	response, doError := updater.httpClient.Do(request)
	if doError != nil {
		var urlError *url.Error
		if errors.As(doError, &urlError) {
			return fmt.Errorf("%s %s : %w", updater.method, redactedTarget(updater.url), urlError.Err)
		}
		return doError
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 200))
		return fmt.Errorf("%s %s : statut %d : %s", updater.method, redactedTarget(updater.url), response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func redactedTarget(rawURL string) string {
	parsed, parseError := url.Parse(rawURL)
	if parseError != nil || parsed.Host == "" {
		return "<url invalide>"
	}
	return parsed.Scheme + "://" + parsed.Host
}
