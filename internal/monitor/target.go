package monitor

import (
	"time"

	"gluetun-hc/internal/health"
)

type target struct {
	name        string
	grace       time.Duration
	outbound    *health.Tracker
	inbound     *health.Tracker
	graceUntil  time.Time
	needsRejoin bool
}

func (monitor *Monitor) newTarget(name string, grace time.Duration) *target {
	return &target{
		name:     name,
		grace:    grace,
		outbound: health.NewTracker(monitor.settings.FailThreshold, monitor.settings.RiseThreshold),
		inbound:  health.NewTracker(monitor.settings.FailThreshold, monitor.settings.RiseThreshold),
	}
}

func (monitor *Monitor) isInGrace(checked *target, currentTime time.Time) bool {
	return currentTime.Before(checked.graceUntil)
}
