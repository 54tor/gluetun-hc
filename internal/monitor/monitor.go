package monitor

import (
	"context"
	"log/slog"
	"time"

	"gluetun-hc/internal/config"
)

const defaultExecTimeout = 15 * time.Second

type Monitor struct {
	settings        config.Config
	runtime         Runtime
	controlAPI      ControlAPI
	now             func() time.Time
	logger          *slog.Logger
	gluetunTarget   *target
	watchedTargets  []*target
	execTimeout     time.Duration
	portSyncTimeout time.Duration
	portSync        *portSync
	verdict         gluetunVerdict
}

func New(settings config.Config, runtime Runtime, controlAPI ControlAPI, now func() time.Time, logger *slog.Logger) *Monitor {
	monitor := &Monitor{
		settings:        settings,
		runtime:         runtime,
		controlAPI:      controlAPI,
		now:             now,
		logger:          logger,
		execTimeout:     defaultExecTimeout,
		portSyncTimeout: defaultPortSyncTimeout,
	}
	monitor.gluetunTarget = monitor.newTarget(settings.GluetunContainer, settings.GluetunGrace)
	for _, watched := range settings.Watched {
		monitor.watchedTargets = append(monitor.watchedTargets, monitor.newTarget(watched.Name, watched.Grace))
	}
	return monitor
}

func (monitor *Monitor) Tick(operationContext context.Context) {
	currentTime := monitor.now()
	if monitor.isInGrace(monitor.gluetunTarget, currentTime) {
		return
	}
	result := monitor.pingFrom(operationContext, monitor.gluetunTarget.name)
	if !monitor.gluetunAllowsChecks(operationContext, result, currentTime) {
		return
	}
	if monitor.hasWatchedToRejoin() {
		monitor.rejoinWatched(operationContext, result, currentTime)
		return
	}
	monitor.syncPort(operationContext)
	for _, watched := range monitor.watchedTargets {
		monitor.checkWatched(operationContext, watched, currentTime)
	}
}

func (monitor *Monitor) gluetunAllowsChecks(operationContext context.Context, result probeResult, currentTime time.Time) bool {
	if result.isCounted() {
		monitor.gluetunTarget.outbound.Record(result == probeReachable)
		monitor.verdict.record(result == probeReachable, currentTime)
	}
	if !monitor.gluetunTarget.outbound.Healthy() {
		monitor.logger.Warn("Gluetun n'a plus de connexion, redémarrage", "container", monitor.gluetunTarget.name)
		if monitor.restart(operationContext, monitor.gluetunTarget, currentTime) {
			monitor.markWatchedToRejoin()
		}
		return false
	}
	return result == probeReachable || result == probeUnsupported
}

func (monitor *Monitor) markWatchedToRejoin() {
	for _, watched := range monitor.watchedTargets {
		watched.needsRejoin = true
	}
}

func (monitor *Monitor) hasWatchedToRejoin() bool {
	for _, watched := range monitor.watchedTargets {
		if watched.needsRejoin {
			return true
		}
	}
	return false
}

func (monitor *Monitor) rejoinWatched(operationContext context.Context, gluetunResult probeResult, currentTime time.Time) {
	tunnelIsStable := gluetunResult == probeUnsupported ||
		monitor.gluetunTarget.outbound.ConsecutiveSuccesses() >= monitor.settings.RiseThreshold
	if !tunnelIsStable {
		return
	}
	monitor.logger.Info("Gluetun est stable, redémarrage des conteneurs surveillés")
	for _, watched := range monitor.watchedTargets {
		if watched.needsRejoin && monitor.restart(operationContext, watched, currentTime) {
			watched.needsRejoin = false
		}
	}
}

func (monitor *Monitor) restart(operationContext context.Context, restarted *target, currentTime time.Time) bool {
	running, inspectError := monitor.runtime.IsRunning(operationContext, restarted.name)
	if inspectError != nil {
		monitor.logger.Error("état du conteneur inconnu, pas de redémarrage", "container", restarted.name, "error", inspectError)
		return false
	}
	if !running {
		monitor.logger.Info("conteneur arrêté, laissé à l'arrêt", "container", restarted.name)
		restarted.outbound.Reset()
		restarted.inbound.Reset()
		return true
	}
	if restartError := monitor.runtime.Restart(operationContext, restarted.name); restartError != nil {
		monitor.logger.Error("redémarrage impossible, nouvelle tentative au prochain contrôle", "container", restarted.name, "error", restartError)
		return false
	}
	restarted.outbound.Reset()
	restarted.inbound.Reset()
	restarted.graceUntil = currentTime.Add(restarted.grace)
	if restarted == monitor.gluetunTarget {
		monitor.verdict.record(false, currentTime)
	}
	monitor.markPortSyncAfterRestart(restarted.name)
	return true
}
