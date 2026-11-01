package portupdate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHTTPUpdaterErrorsDoNotLeakQueryString(testCase *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	}))
	closedURL := server.URL
	server.Close()

	for _, target := range []string{closedURL + "/p?token=SECRET", server.URL + "/p?token=SECRET"} {
		applyError := NewHTTPUpdater("POST", target, nil, "", http.DefaultClient).Apply(context.Background(), 5914)
		if applyError == nil || strings.Contains(applyError.Error(), "SECRET") {
			testCase.Fatalf("erreur absente ou fuyant le jeton : %v", applyError)
		}
	}
}
