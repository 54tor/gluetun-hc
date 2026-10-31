package monitor

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"time"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/gluetun"
	"gluetun-hc/internal/health"
)

const (
	pingBinaryMissingExitCode = 127
	pingTimeoutSeconds        = "3"
)

type Runtime interface {
	Exec(operationContext context.Context, container string, command []string) (int, error)
	Restart(operationContext context.Context, container string) error
}

type ControlAPI interface {
	PublicIP(operationContext context.Context) (string, error)
	ForwardedPort(operationContext context.Context) (int, error)
}

type Dialer func(operationContext context.Context, address string) error

type target struct {
	name       string
	grace      time.Duration
	outbound   *health.Tracker
	inbound    *health.Tracker
	graceUntil time.Time
}

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
}

func New(settings config.Config, runtime Runtime, controlAPI ControlAPI, dial Dialer, now func() time.Time, logger *slog.Logger) *Monitor {
	monitor := &Monitor{
		settings:   settings,
		runtime:    runtime,
		controlAPI: controlAPI,
		dial:       dial,
		now:        now,
		logger:     logger,
	}
	monitor.gluetunTarget = monitor.newTarget(settings.GluetunContainer, settings.GluetunGrace)
	for _, watched := range settings.Watched {
		monitor.watchedTargets = append(monitor.watchedTargets, monitor.newTarget(watched.Name, watched.Grace))
	}
	return monitor
}

func (monitor *Monitor) newTarget(name string, grace time.Duration) *target {
	return &target{
		name:     name,
		grace:    grace,
		outbound: health.NewTracker(monitor.settings.FailThreshold, monitor.settings.RiseThreshold),
		inbound:  health.NewTracker(monitor.settings.FailThreshold, monitor.settings.RiseThreshold),
	}
}

func (monitor *Monitor) Tick(operationContext context.Context) {
	currentTime := monitor.now()
	if currentTime.Before(monitor.gluetunTarget.graceUntil) {
		return
	}

	gluetunReachable, gluetunCounted := monitor.pingFrom(operationContext, monitor.gluetunTarget.name)
	if gluetunCounted {
		monitor.gluetunTarget.outbound.Record(gluetunReachable)
	}
	if !monitor.gluetunTarget.outbound.Healthy() {
		monitor.logger.Warn("Gluetun n'a plus de connexion, redémarrage", "container", monitor.gluetunTarget.name)
		monitor.restart(operationContext, monitor.gluetunTarget, currentTime)
		monitor.watchedRestartPending = true
		return
	}
	if gluetunCounted && !gluetunReachable {
		return
	}

	if monitor.watchedRestartPending {
		if !gluetunCounted {
			return
		}
		monitor.logger.Info("Gluetun est revenu, redémarrage des conteneurs surveillés")
		for _, watched := range monitor.watchedTargets {
			monitor.restart(operationContext, watched, currentTime)
		}
		monitor.watchedRestartPending = false
		return
	}

	for _, watched := range monitor.watchedTargets {
		monitor.checkWatched(operationContext, watched, currentTime)
	}
}

func (monitor *Monitor) checkWatched(operationContext context.Context, watched *target, currentTime time.Time) {
	if currentTime.Before(watched.graceUntil) {
		return
	}

	reachable, counted := monitor.pingFrom(operationContext, watched.name)
	if counted {
		watched.outbound.Record(reachable)
	}
	if !watched.outbound.Healthy() {
		monitor.logger.Warn("conteneur sans connexion sortante, redémarrage", "container", watched.name)
		monitor.restart(operationContext, watched, currentTime)
		return
	}

	if watched.name != monitor.settings.InboundContainer {
		return
	}
	monitor.checkInbound(operationContext, watched, currentTime)
}

func (monitor *Monitor) checkInbound(operationContext context.Context, watched *target, currentTime time.Time) {
	publicIP, ipError := monitor.controlAPI.PublicIP(operationContext)
	if ipError != nil {
		monitor.logger.Warn("IP publique indisponible, contrôle entrant sauté", "error", ipError)
		return
	}
	forwardedPort, portError := monitor.controlAPI.ForwardedPort(operationContext)
	if errors.Is(portError, gluetun.ErrNoForwardedPort) {
		return
	}
	if portError != nil {
		monitor.logger.Warn("port forwardé indisponible, contrôle entrant sauté", "error", portError)
		return
	}

	address := net.JoinHostPort(publicIP, strconv.Itoa(forwardedPort))
	dialError := monitor.dial(operationContext, address)
	watched.inbound.Record(dialError == nil)
	if !watched.inbound.Healthy() {
		monitor.logger.Warn("connexion entrante en échec, redémarrage", "container", watched.name, "address", address)
		monitor.restart(operationContext, watched, currentTime)
	}
}

func (monitor *Monitor) restart(operationContext context.Context, restarted *target, currentTime time.Time) {
	if restartError := monitor.runtime.Restart(operationContext, restarted.name); restartError != nil {
		monitor.logger.Error("redémarrage impossible", "container", restarted.name, "error", restartError)
	}
	restarted.outbound.Reset()
	restarted.inbound.Reset()
	restarted.graceUntil = currentTime.Add(restarted.grace)
}

func (monitor *Monitor) pingFrom(operationContext context.Context, container string) (reachable bool, counted bool) {
	for _, pingTarget := range monitor.settings.PingTargets {
		exitCode, execError := monitor.runtime.Exec(operationContext, container,
			[]string{"ping", "-c1", "-W" + pingTimeoutSeconds, pingTarget})
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
