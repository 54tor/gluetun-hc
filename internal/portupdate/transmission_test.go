package portupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fakeSessionIdentifier = "session-identifier"

type fakeTransmission struct {
	server        *httptest.Server
	currentPort   int64
	handshakeSeen bool
}

func newFakeTransmission(testCase *testing.T) *fakeTransmission {
	fake := &fakeTransmission{currentPort: 51413}
	fake.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, provided := request.BasicAuth()
		if !provided || username != "admin" || password != "secret" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		if request.Header.Get("X-Transmission-Session-Id") != fakeSessionIdentifier {
			fake.handshakeSeen = true
			writer.Header().Set("X-Transmission-Session-Id", fakeSessionIdentifier)
			writer.WriteHeader(http.StatusConflict)
			return
		}
		fake.answer(writer, request)
	}))
	testCase.Cleanup(fake.server.Close)
	return fake
}

func (fake *fakeTransmission) answer(writer http.ResponseWriter, request *http.Request) {
	var call struct {
		Method    string                 `json:"method"`
		Arguments map[string]interface{} `json:"arguments"`
		Tag       int                    `json:"tag"`
	}
	json.NewDecoder(request.Body).Decode(&call)
	response := map[string]interface{}{"result": "success", "tag": call.Tag, "arguments": map[string]interface{}{}}
	switch call.Method {
	case "session-get":
		response["arguments"] = map[string]interface{}{"peer-port": fake.currentPort}
	case "session-set":
		fake.currentPort = int64(call.Arguments["peer-port"].(float64))
	}
	json.NewEncoder(writer).Encode(response)
}

func TestTransmissionUpdaterSetsPeerPortAfterHandshake(testCase *testing.T) {
	fake := newFakeTransmission(testCase)
	updater, buildError := NewTransmissionUpdater(fake.server.URL, "admin", "secret")
	if buildError != nil {
		testCase.Fatal(buildError)
	}
	if applyError := updater.Apply(context.Background(), 5914); applyError != nil {
		testCase.Fatal(applyError)
	}
	if fake.currentPort != 5914 || !fake.handshakeSeen {
		testCase.Fatalf("port %d, poignée de main vue %v", fake.currentPort, fake.handshakeSeen)
	}
}

func TestTransmissionUpdaterReadsCurrentPort(testCase *testing.T) {
	fake := newFakeTransmission(testCase)
	updater, _ := NewTransmissionUpdater(fake.server.URL, "admin", "secret")
	currentPort, readError := updater.CurrentPort(context.Background())
	if readError != nil || currentPort != 51413 {
		testCase.Fatalf("port %d, erreur %v", currentPort, readError)
	}
	if fake.currentPort != 51413 {
		testCase.Fatal("la lecture ne doit rien modifier")
	}
}

func TestTransmissionUpdaterFailsWithWrongPassword(testCase *testing.T) {
	fake := newFakeTransmission(testCase)
	updater, _ := NewTransmissionUpdater(fake.server.URL, "admin", "mauvais")
	if applyError := updater.Apply(context.Background(), 5914); applyError == nil {
		testCase.Fatal("erreur attendue")
	}
	if fake.currentPort != 51413 {
		testCase.Fatal("rien ne doit être écrit avec un mauvais mot de passe")
	}
}

func TestTransmissionUpdaterAcceptsBaseURLWithoutRPCPath(testCase *testing.T) {
	endpoint := transmissionEndpoint("http://app:9091", "admin", "secret")
	if endpoint.Path != "/transmission/rpc" {
		testCase.Fatalf("chemin : %q", endpoint.Path)
	}
	custom := transmissionEndpoint("http://app:9091/custom/rpc", "", "")
	if custom.Path != "/custom/rpc" {
		testCase.Fatalf("chemin personnalisé : %q", custom.Path)
	}
}
