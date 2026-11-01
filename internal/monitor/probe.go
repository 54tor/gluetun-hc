package monitor

import (
	"context"
	"errors"
)

const (
	binaryMissingExitCode = 127
	pingTimeoutSeconds    = "3"
)

type probeResult int

const (
	probeReachable probeResult = iota
	probeUnreachable
	probeSkipped
	probeUnsupported
)

func (result probeResult) isCounted() bool {
	return result == probeReachable || result == probeUnreachable
}

func (monitor *Monitor) pingFrom(operationContext context.Context, container string) probeResult {
	for _, pingTarget := range monitor.settings.PingTargets {
		exitCode, execError := monitor.execPing(operationContext, container, pingTarget)
		if operationContext.Err() != nil {
			return probeSkipped
		}
		if errors.Is(execError, ErrContainerUnavailable) {
			monitor.logger.Warn("conteneur absent ou arrêté, sonde sautée", "container", container)
			return probeSkipped
		}
		if execError != nil {
			monitor.logger.Warn("exec impossible", "container", container, "error", execError)
			return probeUnreachable
		}
		if exitCode == binaryMissingExitCode {
			monitor.logger.Error("ping absent de l'image, sonde ignorée", "container", container)
			return probeUnsupported
		}
		if exitCode == 0 {
			return probeReachable
		}
	}
	return probeUnreachable
}

func (monitor *Monitor) execPing(operationContext context.Context, container, pingTarget string) (int, error) {
	execContext, cancelExec := context.WithTimeout(operationContext, monitor.execTimeout)
	defer cancelExec()
	return monitor.runtime.Exec(execContext, container,
		[]string{"ping", "-c1", "-W" + pingTimeoutSeconds, pingTarget})
}
