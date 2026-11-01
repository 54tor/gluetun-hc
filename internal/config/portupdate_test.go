package config

import "testing"

func portUpdateEnvironment(extra map[string]string) func(string) string {
	values := map[string]string{
		"GLUETUN_CONTAINER": "gluetun",
		"WATCHED":           "app",
	}
	for key, value := range extra {
		values[key] = value
	}
	return environmentFrom(values)
}

func TestPortUpdateDisabledByDefault(testCase *testing.T) {
	loaded, loadError := Load(portUpdateEnvironment(nil))
	if loadError != nil || loaded.PortUpdate.Mode != "" {
		testCase.Fatalf("mode %q, erreur %v", loaded.PortUpdate.Mode, loadError)
	}
}

func TestPortUpdateQBittorrentSettings(testCase *testing.T) {
	loaded, loadError := Load(portUpdateEnvironment(map[string]string{
		"PORT_UPDATE_MODE":      "qbittorrent",
		"PORT_UPDATE_URL":       "http://app:8080",
		"PORT_UPDATE_USER":      "admin",
		"PORT_UPDATE_PASSWORD":  "secret",
		"PORT_UPDATE_CONTAINER": "app",
	}))
	if loadError != nil {
		testCase.Fatal(loadError)
	}
	update := loaded.PortUpdate
	if update.Mode != "qbittorrent" || update.URL != "http://app:8080" || update.Username != "admin" || update.Password != "secret" || update.Container != "app" {
		testCase.Fatalf("réglages inattendus : %+v", update)
	}
}

func TestPortUpdateContainerDefaultsToInboundContainer(testCase *testing.T) {
	loaded, loadError := Load(portUpdateEnvironment(map[string]string{
		"PORT_UPDATE_MODE":  "qbittorrent",
		"PORT_UPDATE_URL":   "http://app:8080",
		"INBOUND_CONTAINER": "app",
	}))
	if loadError != nil || loaded.PortUpdate.Container != "app" {
		testCase.Fatalf("conteneur %q, erreur %v", loaded.PortUpdate.Container, loadError)
	}
}

func TestPortUpdateHTTPSettings(testCase *testing.T) {
	loaded, loadError := Load(portUpdateEnvironment(map[string]string{
		"PORT_UPDATE_MODE":      "http",
		"PORT_UPDATE_URL":       "http://app/port/{{port}}",
		"PORT_UPDATE_CONTAINER": "app",
		"PORT_UPDATE_BODY":      `{"port":{{port}}}`,
		"PORT_UPDATE_HEADERS":   "Authorization: Bearer token\nContent-Type: application/json",
	}))
	if loadError != nil {
		testCase.Fatal(loadError)
	}
	update := loaded.PortUpdate
	if update.Method != "POST" || update.Body != `{"port":{{port}}}` {
		testCase.Fatalf("méthode ou corps : %+v", update)
	}
	if update.Headers["Authorization"] != "Bearer token" || update.Headers["Content-Type"] != "application/json" {
		testCase.Fatalf("en-têtes : %v", update.Headers)
	}
}

func TestPortUpdateRejectsInvalidSettings(testCase *testing.T) {
	invalidCases := map[string]map[string]string{
		"mode inconnu":            {"PORT_UPDATE_MODE": "autre", "PORT_UPDATE_URL": "http://a", "PORT_UPDATE_CONTAINER": "app"},
		"URL absente":             {"PORT_UPDATE_MODE": "qbittorrent", "PORT_UPDATE_CONTAINER": "app"},
		"conteneur absent":        {"PORT_UPDATE_MODE": "qbittorrent", "PORT_UPDATE_URL": "http://a"},
		"conteneur non surveillé": {"PORT_UPDATE_MODE": "qbittorrent", "PORT_UPDATE_URL": "http://a", "PORT_UPDATE_CONTAINER": "inconnu"},
		"en-tête invalide":        {"PORT_UPDATE_MODE": "http", "PORT_UPDATE_URL": "http://a", "PORT_UPDATE_CONTAINER": "app", "PORT_UPDATE_HEADERS": "sans-deux-points"},
	}
	for caseName, extra := range invalidCases {
		if _, loadError := Load(portUpdateEnvironment(extra)); loadError == nil {
			testCase.Fatalf("%s : erreur attendue", caseName)
		}
	}
}
