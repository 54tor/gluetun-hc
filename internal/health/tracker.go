package health

type Tracker struct {
	failThreshold      int
	riseThreshold      int
	consecutiveFailure int
	consecutiveSuccess int
	healthy            bool
}

func NewTracker(failThreshold, riseThreshold int) *Tracker {
	return &Tracker{failThreshold: failThreshold, riseThreshold: riseThreshold, healthy: true}
}

func (tracker *Tracker) Record(succeeded bool) {
	if succeeded {
		tracker.consecutiveSuccess++
		tracker.consecutiveFailure = 0
		if tracker.consecutiveSuccess >= tracker.riseThreshold {
			tracker.healthy = true
		}
		return
	}
	tracker.consecutiveFailure++
	tracker.consecutiveSuccess = 0
	if tracker.consecutiveFailure >= tracker.failThreshold {
		tracker.healthy = false
	}
}

func (tracker *Tracker) Healthy() bool {
	return tracker.healthy
}

func (tracker *Tracker) Reset() {
	tracker.consecutiveFailure = 0
	tracker.consecutiveSuccess = 0
	tracker.healthy = true
}

func (tracker *Tracker) ConsecutiveSuccesses() int {
	return tracker.consecutiveSuccess
}
