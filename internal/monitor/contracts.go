package monitor

import (
	"context"
	"errors"
)

var ErrContainerUnavailable = errors.New("conteneur absent ou arrêté")

type Runtime interface {
	Exec(operationContext context.Context, container string, command []string) (int, error)
	Restart(operationContext context.Context, container string) error
	IsRunning(operationContext context.Context, container string) (bool, error)
}

type ControlAPI interface {
	PublicIP(operationContext context.Context) (string, error)
	ForwardedPort(operationContext context.Context) (int, error)
}

type Dialer func(operationContext context.Context, address string) error
