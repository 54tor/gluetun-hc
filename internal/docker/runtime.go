package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"

	"gluetun-hc/internal/monitor"
)

const executableNotFoundExitCode = 127

func exitCodeFromStartError(startError error) (int, bool) {
	if strings.Contains(startError.Error(), "executable file not found") {
		return executableNotFoundExitCode, true
	}
	return 0, false
}

func wrapUnavailable(dockerError error) error {
	if cerrdefs.IsNotFound(dockerError) || cerrdefs.IsConflict(dockerError) {
		return fmt.Errorf("%w : %v", monitor.ErrContainerUnavailable, dockerError)
	}
	return dockerError
}

const execPollInterval = 200 * time.Millisecond

type Runtime struct {
	dockerClient *client.Client
}

func NewRuntime(dockerClient *client.Client) *Runtime {
	return &Runtime{dockerClient: dockerClient}
}

func (runtime *Runtime) Exec(operationContext context.Context, container string, command []string) (int, error) {
	created, dockerError := runtime.dockerClient.ExecCreate(operationContext, container, client.ExecCreateOptions{Cmd: command})
	if dockerError != nil {
		return 0, wrapUnavailable(dockerError)
	}
	if _, dockerError := runtime.dockerClient.ExecStart(operationContext, created.ID, client.ExecStartOptions{Detach: true}); dockerError != nil {
		if exitCode, recognized := exitCodeFromStartError(dockerError); recognized {
			return exitCode, nil
		}
		return 0, wrapUnavailable(dockerError)
	}
	ticker := time.NewTicker(execPollInterval)
	defer ticker.Stop()
	for {
		inspected, dockerError := runtime.dockerClient.ExecInspect(operationContext, created.ID, client.ExecInspectOptions{})
		if dockerError != nil {
			return 0, dockerError
		}
		if !inspected.Running {
			return inspected.ExitCode, nil
		}
		select {
		case <-operationContext.Done():
			return 0, operationContext.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) Restart(operationContext context.Context, container string) error {
	_, dockerError := runtime.dockerClient.ContainerRestart(operationContext, container, client.ContainerRestartOptions{})
	return dockerError
}
