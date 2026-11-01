package gluetun

import (
	"context"
	"testing"
)

func TestSelectSourceApiUsesOnlyTheAPI(testCase *testing.T) {
	files := &fakeSource{publicIP: "from-files"}
	api := &fakeSource{publicIP: "from-api"}
	selected := SelectSource("api", files, api)
	publicIP, _ := selected.PublicIP(context.Background())
	if publicIP != "from-api" || files.publicIPCalls != 0 {
		testCase.Fatalf("IP %q, appels fichiers %d", publicIP, files.publicIPCalls)
	}
}

func TestSelectSourceFilesUsesOnlyTheFiles(testCase *testing.T) {
	files := &fakeSource{publicIP: "from-files"}
	api := &fakeSource{publicIP: "from-api"}
	selected := SelectSource("files", files, api)
	publicIP, _ := selected.PublicIP(context.Background())
	if publicIP != "from-files" || api.publicIPCalls != 0 {
		testCase.Fatalf("IP %q, appels API %d", publicIP, api.publicIPCalls)
	}
}

func TestSelectSourceAutoFallsBackToTheAPI(testCase *testing.T) {
	api := &fakeSource{publicIP: "from-api"}
	selected := SelectSource("auto", &fakeSource{failure: context.Canceled}, api)
	publicIP, selectError := selected.PublicIP(context.Background())
	if selectError != nil || publicIP != "from-api" {
		testCase.Fatalf("IP %q, erreur %v", publicIP, selectError)
	}
}
