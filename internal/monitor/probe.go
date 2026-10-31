package monitor

import (
	"context"
	"errors"
)

const (
	pingBinaryMissingExitCode = 127
	pingTimeoutSeconds        = "3"
)

func (monitor *Monitor) pingFrom(operationContext context.Context, container string) (reachable bool, counted bool) {
	for _, pingTarget := range monitor.settings.PingTargets {
		exitCode, execError := monitor.execPing(operationContext, container, pingTarget)
		if errors.Is(execError, ErrContainerUnavailable) {
			monitor.logger.Warn("conteneur absent ou arrêté, sonde sautée", "container", container)
			return false, false
		}
		if execError != nil {
			monitor.logger.Warn("exec impossible", "container", container, "error", execError)
			return false, true
		}
		if exitCode == pingBinaryMissingExitCode {
			monitor.logger.Error("ping absent de l'image, sonde ignorée", "container", container)
			return false, false
		}
		if exitCode == 0 {
			return true, true
		}
	}
	return false, true
}

func (monitor *Monitor) execPing(operationContext context.Context, container, pingTarget string) (int, error) {
	execContext, cancelExec := context.WithTimeout(operationContext, monitor.execTimeout)
	defer cancelExec()
	return monitor.runtime.Exec(execContext, container,
		[]string{"ping", "-c1", "-W" + pingTimeoutSeconds, pingTarget})
}
