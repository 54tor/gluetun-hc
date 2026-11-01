package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/moby/moby/client"
)

func pickSharedAddress(ownNetworkIDs map[string]bool, targetAddresses map[string]string) (string, bool) {
	networkIDs := make([]string, 0, len(targetAddresses))
	for networkID := range targetAddresses {
		networkIDs = append(networkIDs, networkID)
	}
	sort.Strings(networkIDs)
	for _, networkID := range networkIDs {
		isShared := len(ownNetworkIDs) == 0 || ownNetworkIDs[networkID]
		if isShared && targetAddresses[networkID] != "" {
			return targetAddresses[networkID], true
		}
	}
	return "", false
}

func (runtime *Runtime) ServiceAddress(operationContext context.Context, service string) (string, bool, error) {
	matches, listError := runtime.listServiceContainers(operationContext, service)
	if listError != nil {
		return "", false, listError
	}
	switch len(matches) {
	case 0:
		return "", false, nil
	case 1:
	default:
		return "", false, fmt.Errorf("le service %q correspond à %d conteneurs, adresse ambiguë", service, len(matches))
	}

	targetAddresses := map[string]string{}
	if matches[0].NetworkSettings != nil {
		for _, endpoint := range matches[0].NetworkSettings.Networks {
			if endpoint != nil && endpoint.IPAddress.IsValid() {
				targetAddresses[endpoint.NetworkID] = endpoint.IPAddress.String()
			}
		}
	}
	address, found := pickSharedAddress(runtime.ownNetworkIDs(operationContext), targetAddresses)
	if !found {
		return "", true, fmt.Errorf("le service %q n'a aucun réseau commun avec le moniteur", service)
	}
	return address, true, nil
}

func (runtime *Runtime) ownNetworkIDs(operationContext context.Context) map[string]bool {
	networkIDs := map[string]bool{}
	if runtime.ownContainerID == "" {
		return networkIDs
	}
	listed, listError := runtime.dockerClient.ContainerList(operationContext, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("id", runtime.ownContainerID),
	})
	if listError != nil || len(listed.Items) == 0 || listed.Items[0].NetworkSettings == nil {
		return networkIDs
	}
	for _, endpoint := range listed.Items[0].NetworkSettings.Networks {
		if endpoint != nil {
			networkIDs[endpoint.NetworkID] = true
		}
	}
	return networkIDs
}
