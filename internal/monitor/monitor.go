package monitor

import (
	"context"
	"log/slog"
	"time"

	"gluetun-hc/internal/config"
)

const defaultExecTimeout = 15 * time.Second

type Monitor struct {
	settings              config.Config
	runtime               Runtime
	controlAPI            ControlAPI
	dial                  Dialer
	now                   func() time.Time
	logger                *slog.Logger
	gluetunTarget         *target
	watchedTargets        []*target
	watchedRestartPending bool
	execTimeout           time.Duration
}

func New(settings config.Config, runtime Runtime, controlAPI ControlAPI, dial Dialer, now func() time.Time, logger *slog.Logger) *Monitor {
	monitor := &Monitor{
		settings:    settings,
		runtime:     runtime,
		controlAPI:  controlAPI,
		dial:        dial,
		now:         now,
		logger:      logger,
		execTimeout: defaultExecTimeout,
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
	if !monitor.gluetunAllowsChecks(operationContext, currentTime) {
		return
	}
	if monitor.watchedRestartPending {
		monitor.restartAllWatched(operationContext, currentTime)
		return
	}
	for _, watched := range monitor.watchedTargets {
		monitor.checkWatched(operationContext, watched, currentTime)
	}
}

func (monitor *Monitor) gluetunAllowsChecks(operationContext context.Context, currentTime time.Time) bool {
	reachable, counted := monitor.pingFrom(operationContext, monitor.gluetunTarget.name)
	if counted {
		monitor.gluetunTarget.outbound.Record(reachable)
	}
	if !monitor.gluetunTarget.outbound.Healthy() {
		monitor.logger.Warn("Gluetun n'a plus de connexion, redémarrage", "container", monitor.gluetunTarget.name)
		monitor.restart(operationContext, monitor.gluetunTarget, currentTime)
		monitor.watchedRestartPending = true
		return false
	}
	if counted && !reachable {
		return false
	}
	return counted || !monitor.watchedRestartPending
}

func (monitor *Monitor) restartAllWatched(operationContext context.Context, currentTime time.Time) {
	monitor.logger.Info("Gluetun est revenu, redémarrage des conteneurs surveillés")
	for _, watched := range monitor.watchedTargets {
		monitor.restart(operationContext, watched, currentTime)
	}
	monitor.watchedRestartPending = false
}

func (monitor *Monitor) restart(operationContext context.Context, restarted *target, currentTime time.Time) {
	if restartError := monitor.runtime.Restart(operationContext, restarted.name); restartError != nil {
		monitor.logger.Error("redémarrage impossible", "container", restarted.name, "error", restartError)
	}
	restarted.outbound.Reset()
	restarted.inbound.Reset()
	restarted.graceUntil = currentTime.Add(restarted.grace)
}
