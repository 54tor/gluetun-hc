package docker

import "testing"

func TestPickSharedAddressPrefersNetworkSharedWithMonitor(testCase *testing.T) {
	ownNetworkIDs := map[string]bool{"network-shared": true}
	targetAddresses := map[string]string{"network-private": "10.0.1.5", "network-shared": "10.0.2.7"}
	address, found := pickSharedAddress(ownNetworkIDs, targetAddresses)
	if !found || address != "10.0.2.7" {
		testCase.Fatalf("adresse %q, trouvée %v", address, found)
	}
}

func TestPickSharedAddressFailsWithoutCommonNetworkWhenOwnNetworksKnown(testCase *testing.T) {
	ownNetworkIDs := map[string]bool{"network-a": true}
	targetAddresses := map[string]string{"network-b": "10.0.1.5"}
	if _, found := pickSharedAddress(ownNetworkIDs, targetAddresses); found {
		testCase.Fatal("aucun réseau commun : adresse inutilisable")
	}
}

func TestPickSharedAddressFallsBackToAnyAddressWhenOwnNetworksUnknown(testCase *testing.T) {
	targetAddresses := map[string]string{"network-b": "10.0.1.5"}
	address, found := pickSharedAddress(nil, targetAddresses)
	if !found || address != "10.0.1.5" {
		testCase.Fatalf("adresse %q, trouvée %v", address, found)
	}
}
