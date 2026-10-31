package docker

import (
	"context"
	"time"

	"github.com/moby/moby/client"
)

const execPollInterval = 200 * time.Millisecond

type Runtime struct {
	api *client.Client
}

func NewRuntime(api *client.Client) *Runtime {
	return &Runtime{api: api}
}

func (runtime *Runtime) Exec(ctx context.Context, container string, command []string) (int, error) {
	created, err := runtime.api.ExecCreate(ctx, container, client.ExecCreateOptions{Cmd: command})
	if err != nil {
		return 0, err
	}
	if _, err := runtime.api.ExecStart(ctx, created.ID, client.ExecStartOptions{Detach: true}); err != nil {
		return 0, err
	}
	ticker := time.NewTicker(execPollInterval)
	defer ticker.Stop()
	for {
		inspected, err := runtime.api.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
		if err != nil {
			return 0, err
		}
		if !inspected.Running {
			return inspected.ExitCode, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) Restart(ctx context.Context, container string) error {
	_, err := runtime.api.ContainerRestart(ctx, container, client.ContainerRestartOptions{})
	return err
}
