package monitor

import (
	"testing"
	"time"
)

func TestNotHealthyBeforeFirstGluetunProbe(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	if bench.monitor.Healthy() {
		testCase.Fatal("aucune sonde encore réussie : pas sain")
	}
}

func TestHealthyAfterSuccessfulGluetunProbe(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(1)
	if !bench.monitor.Healthy() {
		testCase.Fatal("le ping de Gluetun a réussi : sain")
	}
}

func TestNotHealthyWhenLastGluetunProbeFailed(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(1)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(1)
	if bench.monitor.Healthy() {
		testCase.Fatal("le dernier ping a échoué : pas sain, même sous le seuil de redémarrage")
	}
}

func TestNotHealthyWhenLastProbeIsStale(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(1)
	bench.advance(31 * time.Second)
	if bench.monitor.Healthy() {
		testCase.Fatal("sonde trop ancienne (plus de trois intervalles) : pas sain")
	}
}

func TestNotHealthyDuringGluetunGraceAfterRestart(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.advance(time.Second)
	bench.tick(1)
	if bench.monitor.Healthy() {
		testCase.Fatal("Gluetun vient d'être redémarré, non vérifié : pas sain")
	}
	bench.advance(6 * time.Minute)
	bench.tick(1)
	if !bench.monitor.Healthy() {
		testCase.Fatal("après la grâce, un ping réussi rétablit l'état sain")
	}
}

func TestSkippedProbeKeepsPreviousVerdict(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(1)
	bench.runtime.pingErrorByContainer[gluetunName] = ErrContainerUnavailable
	bench.advance(5 * time.Second)
	bench.tick(1)
	if !bench.monitor.Healthy() {
		testCase.Fatal("une sonde sautée ne change pas le verdict récent")
	}
}
