package monitor

import (
	"sync"
	"time"
)

const staleProbeIntervals = 3

type gluetunVerdict struct {
	lock      sync.Mutex
	reachable bool
	probedAt  time.Time
}

func (verdict *gluetunVerdict) record(reachable bool, probedAt time.Time) {
	verdict.lock.Lock()
	defer verdict.lock.Unlock()
	verdict.reachable = reachable
	verdict.probedAt = probedAt
}

func (verdict *gluetunVerdict) read() (reachable bool, probedAt time.Time) {
	verdict.lock.Lock()
	defer verdict.lock.Unlock()
	return verdict.reachable, verdict.probedAt
}

func (monitor *Monitor) Healthy() bool {
	reachable, probedAt := monitor.verdict.read()
	if !reachable || probedAt.IsZero() {
		return false
	}
	maximumAge := time.Duration(staleProbeIntervals) * monitor.settings.Interval
	return monitor.now().Sub(probedAt) <= maximumAge
}
