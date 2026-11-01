package monitor

import (
	"context"
	"errors"

	"gluetun-hc/internal/gluetun"
)

type PortUpdater interface {
	Apply(operationContext context.Context, port int) error
}

type PortReader interface {
	CurrentPort(operationContext context.Context) (int, error)
}

type portSync struct {
	updater       PortUpdater
	reader        PortReader
	container     string
	appliedPort   int
	reapplyNeeded bool
	pending       bool
}

func (monitor *Monitor) EnablePortSync(updater PortUpdater, container string) {
	reader, _ := updater.(PortReader)
	monitor.portSync = &portSync{
		updater:       updater,
		reader:        reader,
		container:     container,
		reapplyNeeded: true,
		pending:       true,
	}
}

func (monitor *Monitor) isPortSyncPending() bool {
	return monitor.portSync != nil && monitor.portSync.pending
}

func (monitor *Monitor) markPortSyncAfterRestart(restartedContainer string) {
	if monitor.portSync != nil && monitor.portSync.container == restartedContainer {
		monitor.portSync.reapplyNeeded = true
		monitor.portSync.pending = true
	}
}

func (monitor *Monitor) syncPort(operationContext context.Context) {
	if monitor.portSync == nil {
		return
	}
	forwardedPort, portError := monitor.controlAPI.ForwardedPort(operationContext)
	if errors.Is(portError, gluetun.ErrNoForwardedPort) {
		return
	}
	if portError != nil {
		monitor.logger.Warn("port forwardé indisponible, mise à jour du port sautée", "error", portError)
		return
	}

	switch monitor.portSyncState(operationContext, forwardedPort) {
	case portInSync:
		monitor.portSync.pending = false
		monitor.portSync.appliedPort = forwardedPort
	case portOutOfSync:
		monitor.portSync.pending = true
		monitor.applyPort(operationContext, forwardedPort)
	default:
		monitor.portSync.pending = true
	}
}

type portState int

const (
	portInSync portState = iota
	portOutOfSync
	portUnknown
)

func (monitor *Monitor) portSyncState(operationContext context.Context, forwardedPort int) portState {
	if monitor.portSync.reader == nil {
		portChanged := forwardedPort != monitor.portSync.appliedPort
		if !portChanged && !monitor.portSync.reapplyNeeded {
			return portInSync
		}
		return portOutOfSync
	}
	currentPort, readError := monitor.portSync.reader.CurrentPort(operationContext)
	if readError != nil {
		monitor.logger.Warn("port actuel illisible, mise à jour sautée", "container", monitor.portSync.container, "error", readError)
		return portUnknown
	}
	if currentPort == forwardedPort {
		return portInSync
	}
	return portOutOfSync
}

func (monitor *Monitor) applyPort(operationContext context.Context, forwardedPort int) {
	if applyError := monitor.portSync.updater.Apply(operationContext, forwardedPort); applyError != nil {
		monitor.logger.Warn("mise à jour du port impossible, nouvelle tentative au prochain contrôle", "port", forwardedPort, "error", applyError)
		return
	}
	monitor.logger.Info("port mis à jour", "container", monitor.portSync.container, "port", forwardedPort)
	monitor.portSync.appliedPort = forwardedPort
	monitor.portSync.reapplyNeeded = false
	monitor.portSync.pending = false
	monitor.resetInboundTracker(monitor.portSync.container)
}

func (monitor *Monitor) resetInboundTracker(container string) {
	for _, watched := range monitor.watchedTargets {
		if watched.name == container {
			watched.inbound.Reset()
		}
	}
}
