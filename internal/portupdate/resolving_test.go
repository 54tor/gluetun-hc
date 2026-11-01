package portupdate

import (
	"context"
	"net"
	"strings"
	"testing"
)

type fixedResolver struct {
	service string
	address string
}

func (resolver fixedResolver) ServiceAddress(operationContext context.Context, service string) (string, bool, error) {
	if service == resolver.service {
		return resolver.address, true, nil
	}
	return "", false, nil
}

func TestResolvingQBittorrentUpdaterUsesServiceAddress(testCase *testing.T) {
	var receivedPorts []int
	server := newFakeQBittorrent(testCase, &receivedPorts, 6881)
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))

	updater := NewResolvingQBittorrentUpdater("http://qbservice:"+port, "admin", "secret", fixedResolver{service: "qbservice", address: "127.0.0.1"})
	if applyError := updater.Apply(context.Background(), 5914); applyError != nil {
		testCase.Fatal(applyError)
	}
	currentPort, readError := updater.CurrentPort(context.Background())
	if readError != nil || currentPort != 6881 {
		testCase.Fatalf("port lu %d, erreur %v", currentPort, readError)
	}
	if len(receivedPorts) != 1 || receivedPorts[0] != 5914 {
		testCase.Fatalf("ports reçus : %v", receivedPorts)
	}
}
