package monitor

import (
	"context"
	"time"
)

func (monitor *Monitor) checkWatched(operationContext context.Context, watched *target, currentTime time.Time) {
	if monitor.isInGrace(watched, currentTime) {
		return
	}

	result := monitor.pingFrom(operationContext, watched.name)
	if result.isCounted() {
		watched.outbound.Record(result == probeReachable)
	}
	if !watched.outbound.Healthy() {
		monitor.logger.Warn("conteneur sans connexion sortante, redémarrage", "container", watched.name)
		monitor.restart(operationContext, watched, currentTime)
		return
	}

	if watched.name == monitor.settings.InboundContainer {
		monitor.checkInbound(operationContext, watched, currentTime)
	}
}
