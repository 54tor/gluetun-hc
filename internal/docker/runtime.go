package docker

import (
	"context"
	"time"

	"github.com/moby/moby/client"
)

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
		return 0, dockerError
	}
	if _, dockerError := runtime.dockerClient.ExecStart(operationContext, created.ID, client.ExecStartOptions{Detach: true}); dockerError != nil {
		return 0, dockerError
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
