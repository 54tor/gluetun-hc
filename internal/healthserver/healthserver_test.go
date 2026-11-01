package healthserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fixedChecker bool

func (checker fixedChecker) Healthy() bool { return bool(checker) }

func TestHandlerReturns200WhenHealthy(testCase *testing.T) {
	recorder := httptest.NewRecorder()
	NewHandler(fixedChecker(true)).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		testCase.Fatalf("statut %d", recorder.Code)
	}
}

func TestHandlerReturns503WhenUnhealthy(testCase *testing.T) {
	recorder := httptest.NewRecorder()
	NewHandler(fixedChecker(false)).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		testCase.Fatalf("statut %d", recorder.Code)
	}
}

func TestCheckSucceedsAgainstHealthyServer(testCase *testing.T) {
	server := httptest.NewServer(NewHandler(fixedChecker(true)))
	testCase.Cleanup(server.Close)
	if checkError := Check(strings.TrimPrefix(server.URL, "http://")); checkError != nil {
		testCase.Fatal(checkError)
	}
}

func TestCheckFailsAgainstUnhealthyServer(testCase *testing.T) {
	server := httptest.NewServer(NewHandler(fixedChecker(false)))
	testCase.Cleanup(server.Close)
	if checkError := Check(strings.TrimPrefix(server.URL, "http://")); checkError == nil {
		testCase.Fatal("erreur attendue")
	}
}

func TestAddressFromEnvironmentDefaultsAndOverrides(testCase *testing.T) {
	defaultAddress := AddressFromEnvironment(func(string) string { return "" })
	if defaultAddress != "127.0.0.1:8081" {
		testCase.Fatalf("adresse par défaut : %q", defaultAddress)
	}
	overridden := AddressFromEnvironment(func(string) string { return "127.0.0.1:9000" })
	if overridden != "127.0.0.1:9000" {
		testCase.Fatalf("adresse surchargée : %q", overridden)
	}
}
