package monitor

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"gluetun-hc/internal/gluetun"
)

const netcatTimeoutSeconds = "3"

func (monitor *Monitor) checkInbound(operationContext context.Context, watched *target, currentTime time.Time) {
	address, canCheck := monitor.inboundAddress(operationContext)
	if !canCheck {
		return
	}

	reachable, counted := monitor.netcatFromGluetun(operationContext, address)
	if counted {
		watched.inbound.Record(reachable)
	}
	if !watched.inbound.Healthy() {
		monitor.logger.Warn("connexion entrante en échec, redémarrage", "container", watched.name, "address", address)
		monitor.restart(operationContext, watched, currentTime)
	}
}

func (monitor *Monitor) netcatFromGluetun(operationContext context.Context, address string) (reachable bool, counted bool) {
	host, port, splitError := net.SplitHostPort(address)
	if splitError != nil {
		monitor.logger.Error("adresse entrante invalide", "address", address, "error", splitError)
		return false, false
	}
	execContext, cancelExec := context.WithTimeout(operationContext, monitor.execTimeout)
	defer cancelExec()
	exitCode, execError := monitor.runtime.Exec(execContext, monitor.settings.GluetunContainer,
		[]string{"nc", "-z", "-w", netcatTimeoutSeconds, host, port})
	if operationContext.Err() != nil || errors.Is(execError, ErrContainerUnavailable) {
		return false, false
	}
	if execError != nil {
		monitor.logger.Warn("nc impossible dans Gluetun", "error", execError)
		return false, true
	}
	if exitCode == binaryMissingExitCode {
		monitor.logger.Error("nc absent de l'image Gluetun, contrôle entrant ignoré")
		return false, false
	}
	return exitCode == 0, true
}

func (monitor *Monitor) inboundAddress(operationContext context.Context) (address string, canCheck bool) {
	publicIP, ipError := monitor.controlAPI.PublicIP(operationContext)
	if ipError != nil {
		monitor.logger.Warn("IP publique indisponible, contrôle entrant sauté", "error", ipError)
		return "", false
	}
	if publicIP == "" {
		monitor.logger.Warn("IP publique vide, contrôle entrant sauté")
		return "", false
	}

	forwardedPort, portError := monitor.controlAPI.ForwardedPort(operationContext)
	if errors.Is(portError, gluetun.ErrNoForwardedPort) {
		return "", false
	}
	if portError != nil {
		monitor.logger.Warn("port forwardé indisponible, contrôle entrant sauté", "error", portError)
		return "", false
	}
	return net.JoinHostPort(publicIP, strconv.Itoa(forwardedPort)), true
}
