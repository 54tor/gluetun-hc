package gluetun

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeSource struct {
	publicIP      string
	port          int
	failure       error
	publicIPCalls int
	portCalls     int
}

func (source *fakeSource) PublicIP(operationContext context.Context) (string, error) {
	source.publicIPCalls++
	return source.publicIP, source.failure
}

func (source *fakeSource) ForwardedPort(operationContext context.Context) (int, error) {
	source.portCalls++
	return source.port, source.failure
}

func TestFallbackUsesPrimaryWhenItWorks(testCase *testing.T) {
	primary := &fakeSource{port: 5914}
	secondary := &fakeSource{port: 1}
	port, readError := NewFallback(primary, secondary).ForwardedPort(context.Background())
	if readError != nil || port != 5914 || secondary.portCalls != 0 {
		testCase.Fatalf("port %d, erreur %v, appels secondaires %d", port, readError, secondary.portCalls)
	}
}

func TestFallbackUsesSecondaryWhenPrimaryFails(testCase *testing.T) {
	primary := &fakeSource{failure: errors.New("fichier absent")}
	secondary := &fakeSource{publicIP: "203.0.113.7"}
	publicIP, readError := NewFallback(primary, secondary).PublicIP(context.Background())
	if readError != nil || publicIP != "203.0.113.7" {
		testCase.Fatalf("IP %q, erreur %v", publicIP, readError)
	}
}

func TestFallbackKeepsNoForwardedPortAnswer(testCase *testing.T) {
	primary := &fakeSource{failure: ErrNoForwardedPort}
	secondary := &fakeSource{port: 1}
	_, readError := NewFallback(primary, secondary).ForwardedPort(context.Background())
	if !errors.Is(readError, ErrNoForwardedPort) || secondary.portCalls != 0 {
		testCase.Fatalf("erreur %v, appels secondaires %d", readError, secondary.portCalls)
	}
}

func TestFallbackReportsBothErrorsWhenBothFail(testCase *testing.T) {
	primary := &fakeSource{failure: errors.New("fichier absent")}
	secondary := &fakeSource{failure: errors.New("statut 401")}
	_, readError := NewFallback(primary, secondary).ForwardedPort(context.Background())
	if readError == nil {
		testCase.Fatal("erreur attendue")
	}
	message := readError.Error()
	if !strings.Contains(message, "fichier absent") || !strings.Contains(message, "statut 401") {
		testCase.Fatalf("les deux causes doivent apparaître : %q", message)
	}
}
