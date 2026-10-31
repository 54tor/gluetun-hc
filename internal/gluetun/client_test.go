package gluetun

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serverReplying(testCase *testing.T, routes map[string]string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, found := routes[request.URL.Path]
		if !found {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.Header.Get("X-API-Key") != "secret" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Write([]byte(body))
	}))
	testCase.Cleanup(server.Close)
	return server
}

func TestPublicIPParsesBody(testCase *testing.T) {
	server := serverReplying(testCase, map[string]string{"/v1/publicip/ip": `{"public_ip":"203.0.113.7"}`})
	publicIP, callError := NewClient(server.URL, "secret", server.Client()).PublicIP(context.Background())
	if callError != nil || publicIP != "203.0.113.7" {
		testCase.Fatalf("IP %q, erreur %v", publicIP, callError)
	}
}

func TestForwardedPortParsesBody(testCase *testing.T) {
	server := serverReplying(testCase, map[string]string{"/v1/portforward": `{"port":5914}`})
	port, callError := NewClient(server.URL, "secret", server.Client()).ForwardedPort(context.Background())
	if callError != nil || port != 5914 {
		testCase.Fatalf("port %d, erreur %v", port, callError)
	}
}

func TestForwardedPortZeroReturnsErrNoForwardedPort(testCase *testing.T) {
	server := serverReplying(testCase, map[string]string{"/v1/portforward": `{"port":0}`})
	_, callError := NewClient(server.URL, "secret", server.Client()).ForwardedPort(context.Background())
	if !errors.Is(callError, ErrNoForwardedPort) {
		testCase.Fatalf("erreur inattendue : %v", callError)
	}
}

func TestAPIKeySent(testCase *testing.T) {
	server := serverReplying(testCase, map[string]string{"/v1/publicip/ip": `{"public_ip":"203.0.113.7"}`})
	_, callError := NewClient(server.URL, "wrong", server.Client()).PublicIP(context.Background())
	if callError == nil {
		testCase.Fatal("une clé erronée doit échouer")
	}
}

func TestNon200ReturnsError(testCase *testing.T) {
	server := serverReplying(testCase, map[string]string{})
	_, callError := NewClient(server.URL, "secret", server.Client()).PublicIP(context.Background())
	if callError == nil {
		testCase.Fatal("erreur attendue")
	}
}
