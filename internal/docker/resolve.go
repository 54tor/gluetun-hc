package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/client"
)

const composeServiceLabel = "com.docker.compose.service"

func chooseContainerName(requested string, serviceMatches []string) (string, error) {
	switch len(serviceMatches) {
	case 0:
		return requested, nil
	case 1:
		return strings.TrimPrefix(serviceMatches[0], "/"), nil
	default:
		return "", fmt.Errorf("le service %q correspond à %d conteneurs, utiliser un nom de conteneur", requested, len(serviceMatches))
	}
}

func (runtime *Runtime) resolve(operationContext context.Context, requested string) (string, error) {
	listed, listError := runtime.dockerClient.ContainerList(operationContext, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("label", composeServiceLabel+"="+requested),
	})
	if listError != nil {
		return "", listError
	}
	var serviceMatches []string
	for _, summary := range listed.Items {
		if len(summary.Names) > 0 {
			serviceMatches = append(serviceMatches, summary.Names[0])
		}
	}
	return chooseContainerName(requested, serviceMatches)
}
