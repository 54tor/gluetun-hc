package serviceaddr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeResolver struct {
	addressByService map[string]string
	failure          error
}

func (resolver *fakeResolver) ServiceAddress(operationContext context.Context, service string) (string, bool, error) {
	if resolver.failure != nil {
		return "", false, resolver.failure
	}
	address, isService := resolver.addressByService[service]
	return address, isService, nil
}

func TestHTTPClientDialsResolvedAddressForServiceNames(testCase *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(request.Host))
	}))
	testCase.Cleanup(server.Close)
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))

	resolver := &fakeResolver{addressByService: map[string]string{"gluetun": "127.0.0.1"}}
	httpClient := NewHTTPClient(2*time.Second, resolver)
	response, requestError := httpClient.Get("http://gluetun:" + port + "/")
	if requestError != nil {
		testCase.Fatal(requestError)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		testCase.Fatalf("statut %d", response.StatusCode)
	}
}

func TestHTTPClientLeavesIPAddressesUntouched(testCase *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {}))
	testCase.Cleanup(server.Close)

	resolver := &fakeResolver{failure: errors.New("ne doit pas être appelé pour une IP")}
	httpClient := NewHTTPClient(2*time.Second, resolver)
	response, requestError := httpClient.Get(server.URL)
	if requestError != nil {
		testCase.Fatal(requestError)
	}
	response.Body.Close()
}

func TestHTTPClientFailsWhenServiceIsAmbiguous(testCase *testing.T) {
	resolver := &fakeResolver{failure: errors.New("service ambigu")}
	httpClient := NewHTTPClient(2*time.Second, resolver)
	if _, requestError := httpClient.Get("http://gluetun:8080/"); requestError == nil {
		testCase.Fatal("une ambiguïté ne doit jamais retomber sur le DNS partagé")
	}
}

func TestRewriteURLReplacesServiceHostKeepingPortAndPath(testCase *testing.T) {
	resolver := &fakeResolver{addressByService: map[string]string{"gluetun": "10.0.2.7"}}
	rewritten, rewriteError := RewriteURL(context.Background(), "http://gluetun:8080/api", resolver)
	if rewriteError != nil || rewritten != "http://10.0.2.7:8080/api" {
		testCase.Fatalf("URL %q, erreur %v", rewritten, rewriteError)
	}
}

func TestRewriteURLKeepsNonServiceHosts(testCase *testing.T) {
	resolver := &fakeResolver{addressByService: map[string]string{}}
	rewritten, rewriteError := RewriteURL(context.Background(), "http://example.com:8080/api", resolver)
	if rewriteError != nil {
		testCase.Fatal(rewriteError)
	}
	parsed, _ := url.Parse(rewritten)
	if parsed.Host != "example.com:8080" {
		testCase.Fatalf("hôte modifié : %q", parsed.Host)
	}
}
