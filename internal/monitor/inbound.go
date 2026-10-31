package monitor

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"gluetun-hc/internal/gluetun"
)

func (monitor *Monitor) checkInbound(operationContext context.Context, watched *target, currentTime time.Time) {
	address, canCheck := monitor.inboundAddress(operationContext)
	if !canCheck {
		return
	}

	dialError := monitor.dial(operationContext, address)
	watched.inbound.Record(dialError == nil)
	if !watched.inbound.Healthy() {
		monitor.logger.Warn("connexion entrante en échec, redémarrage", "container", watched.name, "address", address)
		monitor.restart(operationContext, watched, currentTime)
	}
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
