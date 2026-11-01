package portupdate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPUpdaterSendsTemplatedRequest(testCase *testing.T) {
	var receivedMethod, receivedPath, receivedBody, receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedMethod = request.Method
		receivedPath = request.URL.Path
		receivedHeader = request.Header.Get("Authorization")
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
	}))
	testCase.Cleanup(server.Close)

	updater := NewHTTPUpdater("PUT", server.URL+"/port/{{port}}", map[string]string{"Authorization": "Bearer token"}, `{"port":{{port}}}`, server.Client())
	if applyError := updater.Apply(context.Background(), 5914); applyError != nil {
		testCase.Fatal(applyError)
	}
	if receivedMethod != "PUT" || receivedPath != "/port/5914" || receivedBody != `{"port":5914}` || receivedHeader != "Bearer token" {
		testCase.Fatalf("requête inattendue : %s %s %s %s", receivedMethod, receivedPath, receivedBody, receivedHeader)
	}
}

func TestHTTPUpdaterRejectsNon2xx(testCase *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	}))
	testCase.Cleanup(server.Close)

	updater := NewHTTPUpdater("POST", server.URL, nil, "", server.Client())
	if applyError := updater.Apply(context.Background(), 5914); applyError == nil {
		testCase.Fatal("un statut 403 doit être une erreur")
	}
}
