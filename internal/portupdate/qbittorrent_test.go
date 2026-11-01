package portupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFakeQBittorrent(testCase *testing.T, receivedPorts *[]int, currentPort int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/auth/login":
			request.ParseForm()
			if request.PostForm.Get("username") != "admin" || request.PostForm.Get("password") != "secret" {
				writer.Write([]byte("Fails."))
				return
			}
			http.SetCookie(writer, &http.Cookie{Name: "SID", Value: "session", Path: "/"})
			writer.Write([]byte("Ok."))
		case "/api/v2/app/preferences":
			if cookie, cookieError := request.Cookie("SID"); cookieError != nil || cookie.Value != "session" {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			json.NewEncoder(writer).Encode(map[string]int{"listen_port": currentPort})
		case "/api/v2/app/setPreferences":
			if cookie, cookieError := request.Cookie("SID"); cookieError != nil || cookie.Value != "session" {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			request.ParseForm()
			var preferences map[string]int
			json.Unmarshal([]byte(request.PostForm.Get("json")), &preferences)
			*receivedPorts = append(*receivedPorts, preferences["listen_port"])
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	testCase.Cleanup(server.Close)
	return server
}

func TestQBittorrentUpdaterSetsListenPort(testCase *testing.T) {
	var receivedPorts []int
	server := newFakeQBittorrent(testCase, &receivedPorts, 6881)
	updater := NewQBittorrentUpdater(server.URL, "admin", "secret")
	if applyError := updater.Apply(context.Background(), 5914); applyError != nil {
		testCase.Fatal(applyError)
	}
	if len(receivedPorts) != 1 || receivedPorts[0] != 5914 {
		testCase.Fatalf("ports reçus : %v", receivedPorts)
	}
}

func TestQBittorrentUpdaterFailsWithWrongPassword(testCase *testing.T) {
	var receivedPorts []int
	server := newFakeQBittorrent(testCase, &receivedPorts, 6881)
	updater := NewQBittorrentUpdater(server.URL, "admin", "mauvais")
	if applyError := updater.Apply(context.Background(), 5914); applyError == nil || len(receivedPorts) != 0 {
		testCase.Fatalf("erreur attendue, ports reçus %v", receivedPorts)
	}
}

func TestQBittorrentUpdaterReadsCurrentPortAfterLoggingIn(testCase *testing.T) {
	var receivedPorts []int
	server := newFakeQBittorrent(testCase, &receivedPorts, 6881)
	updater := NewQBittorrentUpdater(server.URL, "admin", "secret")
	currentPort, readError := updater.CurrentPort(context.Background())
	if readError != nil || currentPort != 6881 {
		testCase.Fatalf("port %d, erreur %v", currentPort, readError)
	}
	if len(receivedPorts) != 0 {
		testCase.Fatal("la lecture ne doit rien modifier")
	}
}

func TestQBittorrentUpdaterReadFailsWithWrongPassword(testCase *testing.T) {
	var receivedPorts []int
	server := newFakeQBittorrent(testCase, &receivedPorts, 6881)
	updater := NewQBittorrentUpdater(server.URL, "admin", "mauvais")
	if _, readError := updater.CurrentPort(context.Background()); readError == nil {
		testCase.Fatal("erreur attendue")
	}
}
