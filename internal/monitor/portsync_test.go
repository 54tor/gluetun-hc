package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"gluetun-hc/internal/gluetun"
)

type fakePortUpdater struct {
	appliedPorts []int
	failuresLeft int
}

func (updater *fakePortUpdater) Apply(operationContext context.Context, port int) error {
	updater.appliedPorts = append(updater.appliedPorts, port)
	if updater.failuresLeft > 0 {
		updater.failuresLeft--
		return errors.New("mise à jour refusée")
	}
	return nil
}

func newPortSyncBench() (*testBench, *fakePortUpdater) {
	bench := newBench(singleApp(), "")
	updater := &fakePortUpdater{}
	bench.monitor.EnablePortSync(updater, "app")
	return bench, updater
}

func TestPortSyncAppliesPortAtStartup(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.tick(1)
	if len(updater.appliedPorts) != 1 || updater.appliedPorts[0] != 5914 {
		testCase.Fatalf("ports appliqués : %v", updater.appliedPorts)
	}
}

func TestPortSyncDoesNotRepeatWhenPortUnchanged(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.tick(5)
	if len(updater.appliedPorts) != 1 {
		testCase.Fatalf("un seul appel attendu : %v", updater.appliedPorts)
	}
}

func TestPortSyncAppliesNewPortWhenItChanges(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.tick(1)
	bench.controlAPI.forwardedPort = 6000
	bench.tick(1)
	if len(updater.appliedPorts) != 2 || updater.appliedPorts[1] != 6000 {
		testCase.Fatalf("ports appliqués : %v", updater.appliedPorts)
	}
}

func TestPortSyncRetriesAfterFailure(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	updater.failuresLeft = 1
	bench.tick(3)
	if len(updater.appliedPorts) != 2 {
		testCase.Fatalf("un échec doit être retenté puis arrêté après succès : %v", updater.appliedPorts)
	}
}

func TestPortSyncReappliesAfterTargetRestart(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.tick(1)
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(2)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app aurait dû être redémarré")
	}
	bench.runtime.pingExitCodeByContainer["app"] = 0
	bench.advance(6 * time.Minute)
	bench.tick(1)
	if len(updater.appliedPorts) != 2 {
		testCase.Fatalf("le port doit être réappliqué après le redémarrage : %v", updater.appliedPorts)
	}
}

func TestPortSyncSkippedWithoutForwardedPort(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.controlAPI.portError = gluetun.ErrNoForwardedPort
	bench.tick(3)
	if len(updater.appliedPorts) != 0 {
		testCase.Fatal("sans port forwardé, rien à appliquer")
	}
}

func TestPortSyncSkippedWhenControlAPIDown(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.controlAPI.portError = errors.New("API injoignable")
	bench.tick(3)
	if len(updater.appliedPorts) != 0 {
		testCase.Fatal("API injoignable : rien à appliquer")
	}
}

func TestPortSyncSkippedWhileGluetunFaulty(testCase *testing.T) {
	bench, updater := newPortSyncBench()
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(3)
	if len(updater.appliedPorts) != 0 {
		testCase.Fatal("Gluetun en panne : pas de mise à jour")
	}
}

func TestPortSyncDisabledByDefault(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(3)
}

type fakeReadingUpdater struct {
	fakePortUpdater
	currentPort int
	readError   error
}

func (updater *fakeReadingUpdater) Apply(operationContext context.Context, port int) error {
	applyError := updater.fakePortUpdater.Apply(operationContext, port)
	if applyError == nil {
		updater.currentPort = port
	}
	return applyError
}

func (updater *fakeReadingUpdater) CurrentPort(operationContext context.Context) (int, error) {
	return updater.currentPort, updater.readError
}

func newReadingBench(currentPort int, inboundContainer string) (*testBench, *fakeReadingUpdater) {
	bench := newBench(singleApp(), inboundContainer)
	updater := &fakeReadingUpdater{currentPort: currentPort}
	bench.monitor.EnablePortSync(updater, "app")
	return bench, updater
}

func TestPortSyncSkipsApplyWhenCurrentPortAlreadyMatches(testCase *testing.T) {
	bench, updater := newReadingBench(5914, "")
	bench.tick(3)
	if len(updater.appliedPorts) != 0 {
		testCase.Fatalf("le port est déjà bon, aucune écriture : %v", updater.appliedPorts)
	}
}

func TestPortSyncAppliesWhenCurrentPortDiffers(testCase *testing.T) {
	bench, updater := newReadingBench(6881, "")
	bench.tick(1)
	if len(updater.appliedPorts) != 1 || updater.appliedPorts[0] != 5914 {
		testCase.Fatalf("ports appliqués : %v", updater.appliedPorts)
	}
}

func TestPortSyncDetectsDriftWithoutAnyRestart(testCase *testing.T) {
	bench, updater := newReadingBench(5914, "")
	bench.tick(2)
	updater.currentPort = 6881
	bench.tick(1)
	if len(updater.appliedPorts) != 1 {
		testCase.Fatalf("la dérive doit être corrigée : %v", updater.appliedPorts)
	}
}

func TestPortSyncReadErrorNeverApplies(testCase *testing.T) {
	bench, updater := newReadingBench(6881, "")
	updater.readError = errors.New("qBittorrent injoignable")
	bench.tick(3)
	if len(updater.appliedPorts) != 0 {
		testCase.Fatal("sans lecture, pas d'écriture à l'aveugle")
	}
}

func TestInboundNeverRestartsTargetWhileSyncPending(testCase *testing.T) {
	bench, updater := newReadingBench(6881, "app")
	updater.failuresLeft = 100
	bench.runtime.netcatExitCode = 1
	bench.tick(10)
	if bench.runtime.restartCount("app") != 0 || len(bench.runtime.netcatTargets) != 0 {
		testCase.Fatalf("pas de contrôle entrant ni de redémarrage avant application : %v", bench.runtime.restartedContainers)
	}
}

func TestInboundFailuresBeforeApplicationAreNotCounted(testCase *testing.T) {
	bench, updater := newReadingBench(6881, "app")
	updater.failuresLeft = 1
	bench.runtime.netcatExitCode = 1
	bench.tick(2)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un seul échec entrant après application ne suffit pas")
	}
	bench.tick(1)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("après application, deux échecs entrants consécutifs redémarrent la cible")
	}
}

type blockingPortUpdater struct{}

func (blockingPortUpdater) Apply(operationContext context.Context, port int) error {
	<-operationContext.Done()
	return operationContext.Err()
}

func TestPortSyncBlockedUpdaterIsCutByTimeout(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.monitor.portSyncTimeout = 10 * time.Millisecond
	bench.monitor.EnablePortSync(blockingPortUpdater{}, "app")
	finished := make(chan struct{})
	go func() {
		bench.tick(1)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		testCase.Fatal("Tick doit rendre la main malgré un updater bloqué")
	}
}
