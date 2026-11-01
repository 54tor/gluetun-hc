package monitor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/gluetun"
)

const gluetunName = "gluetun"

type fakeRuntime struct {
	pingExitCodeByContainer map[string]int
	pingErrorByContainer    map[string]error
	pingedContainers        []string
	pingedTargets           []string
	restartedContainers     []string
	blockUntilCancelled     bool
	netcatExitCode          int
	netcatContainers        []string
	netcatTargets           []string
	restartFailuresLeft     map[string]int
	stoppedContainers       map[string]bool
	inspectErrorByContainer map[string]error
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{
		pingExitCodeByContainer: map[string]int{},
		pingErrorByContainer:    map[string]error{},
		restartFailuresLeft:     map[string]int{},
		stoppedContainers:       map[string]bool{},
		inspectErrorByContainer: map[string]error{},
	}
}

func (runtime *fakeRuntime) IsRunning(operationContext context.Context, container string) (bool, error) {
	if inspectError := runtime.inspectErrorByContainer[container]; inspectError != nil {
		return false, inspectError
	}
	return !runtime.stoppedContainers[container], nil
}

func (runtime *fakeRuntime) Exec(operationContext context.Context, container string, command []string) (int, error) {
	if command[0] == "nc" {
		runtime.netcatContainers = append(runtime.netcatContainers, container)
		runtime.netcatTargets = append(runtime.netcatTargets, command[len(command)-2]+":"+command[len(command)-1])
		return runtime.netcatExitCode, nil
	}
	runtime.pingedContainers = append(runtime.pingedContainers, container)
	if operationContext.Err() != nil {
		return 0, operationContext.Err()
	}
	if runtime.blockUntilCancelled {
		<-operationContext.Done()
		return 0, operationContext.Err()
	}
	runtime.pingedTargets = append(runtime.pingedTargets, command[len(command)-1])
	if pingError := runtime.pingErrorByContainer[container]; pingError != nil {
		return 0, pingError
	}
	return runtime.pingExitCodeByContainer[container], nil
}

func (runtime *fakeRuntime) Restart(operationContext context.Context, container string) error {
	runtime.restartedContainers = append(runtime.restartedContainers, container)
	if runtime.restartFailuresLeft[container] > 0 {
		runtime.restartFailuresLeft[container]--
		return errors.New("redémarrage refusé")
	}
	return nil
}

func (runtime *fakeRuntime) forgetCalls() {
	runtime.pingedContainers = nil
	runtime.pingedTargets = nil
	runtime.restartedContainers = nil
	runtime.netcatContainers = nil
	runtime.netcatTargets = nil
}

func (runtime *fakeRuntime) restartCount(container string) int {
	count := 0
	for _, restarted := range runtime.restartedContainers {
		if restarted == container {
			count++
		}
	}
	return count
}

type fakeControlAPI struct {
	publicIP      string
	forwardedPort int
	portError     error
	ipError       error
}

func (controlAPI *fakeControlAPI) PublicIP(operationContext context.Context) (string, error) {
	return controlAPI.publicIP, controlAPI.ipError
}

func (controlAPI *fakeControlAPI) ForwardedPort(operationContext context.Context) (int, error) {
	return controlAPI.forwardedPort, controlAPI.portError
}

type testBench struct {
	runtime     *fakeRuntime
	controlAPI  *fakeControlAPI
	monitor     *Monitor
	currentTime time.Time
}

func newBench(watched []config.Watched, inboundContainer string) *testBench {
	bench := &testBench{
		runtime:     newFakeRuntime(),
		controlAPI:  &fakeControlAPI{publicIP: "203.0.113.7", forwardedPort: 5914},
		currentTime: time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC),
	}
	settings := config.Config{
		GluetunContainer: gluetunName,
		GluetunGrace:     5 * time.Minute,
		Watched:          watched,
		InboundContainer: inboundContainer,
		PingTargets:      []string{"9.9.9.9", "1.1.1.1"},
		FailThreshold:    2,
		RiseThreshold:    2,
	}
	bench.monitor = New(settings, bench.runtime, bench.controlAPI,
		func() time.Time { return bench.currentTime },
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	return bench
}

func (bench *testBench) tick(times int) {
	for range times {
		bench.monitor.Tick(context.Background())
	}
}

func (bench *testBench) advance(duration time.Duration) {
	bench.currentTime = bench.currentTime.Add(duration)
}

func singleApp() []config.Watched {
	return []config.Watched{{Name: "app", Grace: 5 * time.Minute}}
}

func TestHealthyEverythingNoRestart(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.tick(5)
	if len(bench.runtime.restartedContainers) != 0 {
		testCase.Fatalf("redémarrages inattendus : %v", bench.runtime.restartedContainers)
	}
}

func TestGluetunRestartedAfterFailThreshold(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(1)
	if bench.runtime.restartCount(gluetunName) != 0 {
		testCase.Fatal("redémarrage trop tôt")
	}
	bench.tick(1)
	if bench.runtime.restartCount(gluetunName) != 1 {
		testCase.Fatal("Gluetun aurait dû être redémarré")
	}
}

func TestContainersSkippedWhileGluetunFaulty(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	if slices.Contains(bench.runtime.pingedContainers, "app") {
		testCase.Fatal("aucun ping ne doit partir des conteneurs quand Gluetun est en panne")
	}
}

func TestAllWatchedRestartedOnceAfterGluetunRecovers(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.runtime.forgetCalls()

	bench.advance(6 * time.Minute)
	bench.tick(1)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un seul ping réussi ne suffit pas : RISE_THRESHOLD vaut 2")
	}
	bench.tick(1)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatalf("app doit être redémarré une fois : %v", bench.runtime.restartedContainers)
	}
	bench.tick(3)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app ne doit pas être redémarré une seconde fois")
	}
}

func TestGraceSuppressesChecks(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(2)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app aurait dû être redémarré")
	}
	bench.runtime.forgetCalls()

	bench.advance(4 * time.Minute)
	bench.tick(3)
	if slices.Contains(bench.runtime.pingedContainers, "app") {
		testCase.Fatal("app ne doit pas être testé pendant sa grâce")
	}
	bench.advance(2 * time.Minute)
	bench.tick(1)
	if !slices.Contains(bench.runtime.pingedContainers, "app") {
		testCase.Fatal("app doit être testé après sa grâce")
	}
}

func TestPerContainerGrace(testCase *testing.T) {
	watched := []config.Watched{
		{Name: "slow", Grace: 10 * time.Minute},
		{Name: "quick", Grace: 5 * time.Minute},
	}
	bench := newBench(watched, "")
	bench.runtime.pingExitCodeByContainer["slow"] = 1
	bench.runtime.pingExitCodeByContainer["quick"] = 1
	bench.tick(2)
	bench.runtime.forgetCalls()

	bench.advance(6 * time.Minute)
	bench.tick(1)
	if !slices.Contains(bench.runtime.pingedContainers, "quick") || slices.Contains(bench.runtime.pingedContainers, "slow") {
		testCase.Fatalf("après 6 min, seul quick est testé : %v", bench.runtime.pingedContainers)
	}
}

func TestContainerRestartedAfterFailThreshold(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(1)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("redémarrage trop tôt")
	}
	bench.tick(1)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app aurait dû être redémarré")
	}
}

func TestPingMissingBinaryIsNotCounted(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 127
	bench.tick(5)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un ping absent n'est pas une panne réseau")
	}
}

func TestSecondPingTargetUsedWhenFirstFails(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(1)
	if !slices.Contains(bench.runtime.pingedTargets, "9.9.9.9") || !slices.Contains(bench.runtime.pingedTargets, "1.1.1.1") {
		testCase.Fatalf("les deux cibles doivent être tentées : %v", bench.runtime.pingedTargets)
	}
}

func TestInboundFailureRestartsInboundContainer(testCase *testing.T) {
	bench := newBench(singleApp(), "app")
	bench.runtime.netcatExitCode = 1
	bench.tick(2)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app aurait dû être redémarré après échecs entrants")
	}
	if len(bench.runtime.netcatTargets) == 0 || bench.runtime.netcatTargets[0] != "203.0.113.7:5914" {
		testCase.Fatalf("adresse testée : %v", bench.runtime.netcatTargets)
	}
	if bench.runtime.netcatContainers[0] != gluetunName {
		testCase.Fatalf("nc doit partir du namespace de Gluetun : %v", bench.runtime.netcatContainers)
	}
}

func TestInboundSkippedWhenNoForwardedPort(testCase *testing.T) {
	bench := newBench(singleApp(), "app")
	bench.controlAPI.portError = gluetun.ErrNoForwardedPort
	bench.runtime.netcatExitCode = 1
	bench.tick(5)
	if len(bench.runtime.netcatTargets) != 0 || bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("sans port forwardé, le contrôle entrant est sauté")
	}
}

func TestInboundSkippedWhenControlAPIDown(testCase *testing.T) {
	bench := newBench(singleApp(), "app")
	bench.controlAPI.ipError = errors.New("API injoignable")
	bench.runtime.netcatExitCode = 1
	bench.tick(5)
	if len(bench.runtime.netcatTargets) != 0 || bench.runtime.restartCount("app") != 0 || bench.runtime.restartCount(gluetunName) != 0 {
		testCase.Fatal("une API de contrôle injoignable ne déclenche aucun redémarrage")
	}
}

func TestMissingContainerNeverRestartedAndOthersStillChecked(testCase *testing.T) {
	watched := []config.Watched{
		{Name: "missing", Grace: 5 * time.Minute},
		{Name: "app", Grace: 5 * time.Minute},
	}
	bench := newBench(watched, "")
	bench.runtime.pingErrorByContainer["missing"] = ErrContainerUnavailable
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(2)
	if bench.runtime.restartCount("missing") != 0 {
		testCase.Fatal("un conteneur absent ou arrêté ne doit pas être redémarré")
	}
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("app doit être contrôlé même si un autre conteneur est absent")
	}
}

func TestInboundSkippedWhenPublicIPEmpty(testCase *testing.T) {
	bench := newBench(singleApp(), "app")
	bench.controlAPI.publicIP = ""
	bench.runtime.netcatExitCode = 1
	bench.tick(5)
	if len(bench.runtime.netcatTargets) != 0 || bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("une IP publique vide saute le contrôle entrant")
	}
}

func TestBlockedExecIsCutByTimeout(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.monitor.execTimeout = 10 * time.Millisecond
	bench.runtime.blockUntilCancelled = true
	finished := make(chan struct{})
	go func() {
		bench.tick(2)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		testCase.Fatal("Tick doit rendre la main malgré un exec bloqué")
	}
}

func TestCancelledContextNeverCountsAsFailure(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.runtime.pingExitCodeByContainer["app"] = 1
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	for range 5 {
		bench.monitor.Tick(cancelledContext)
	}
	if len(bench.runtime.restartedContainers) != 0 {
		testCase.Fatalf("un arrêt ne doit déclencher aucun redémarrage : %v", bench.runtime.restartedContainers)
	}
}

func TestFailedRestartIsRetriedWithoutGrace(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.runtime.restartFailuresLeft["app"] = 1
	bench.tick(2)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("première tentative attendue")
	}
	bench.tick(1)
	if bench.runtime.restartCount("app") != 2 {
		testCase.Fatal("un redémarrage raté doit être retenté au contrôle suivant, sans délai de grâce")
	}
	bench.runtime.forgetCalls()
	bench.tick(3)
	if len(bench.runtime.pingedContainers) != 0 && slices.Contains(bench.runtime.pingedContainers, "app") {
		testCase.Fatal("après un redémarrage réussi, la grâce s'applique")
	}
}

func TestFailedWatchedRestartAfterGluetunIsRetriedForThatContainerOnly(testCase *testing.T) {
	watched := []config.Watched{
		{Name: "first", Grace: 5 * time.Minute},
		{Name: "second", Grace: 5 * time.Minute},
	}
	bench := newBench(watched, "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.runtime.restartFailuresLeft["first"] = 1
	bench.advance(6 * time.Minute)
	bench.tick(2)
	bench.tick(1)
	if bench.runtime.restartCount("first") != 2 || bench.runtime.restartCount("second") != 1 {
		testCase.Fatalf("first doit être retenté seul : %v", bench.runtime.restartedContainers)
	}
}

func TestGluetunMissingBlocksWatchedChecks(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingErrorByContainer[gluetunName] = ErrContainerUnavailable
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.tick(5)
	if slices.Contains(bench.runtime.pingedContainers, "app") || len(bench.runtime.restartedContainers) != 0 {
		testCase.Fatal("Gluetun absent : aucun contrôle ni redémarrage des conteneurs")
	}
}

func TestGluetunWithoutPingStillRejoinsWatched(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 127
	bench.advance(6 * time.Minute)
	bench.tick(1)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("sans ping dans Gluetun, la preuve est impossible : on raccroche quand même")
	}
}

func TestGluetunFlapResetsRiseStreak(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.advance(6 * time.Minute)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.tick(1)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(1)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.tick(1)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un rebond du tunnel remet la série à zéro")
	}
	bench.tick(1)
	if bench.runtime.restartCount("app") != 1 {
		testCase.Fatal("deux succès consécutifs doivent déclencher le raccrochage")
	}
}

func TestManuallyStoppedContainerNotRestartedOnFailure(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.runtime.stoppedContainers["app"] = true
	bench.tick(5)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un conteneur arrêté à la main ne doit jamais être relancé")
	}
}

func TestManuallyStoppedContainerNotRejoinedAfterGluetunRestart(testCase *testing.T) {
	watched := []config.Watched{
		{Name: "stopped", Grace: 5 * time.Minute},
		{Name: "running", Grace: 5 * time.Minute},
	}
	bench := newBench(watched, "")
	bench.runtime.stoppedContainers["stopped"] = true
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.tick(2)
	bench.runtime.pingExitCodeByContainer[gluetunName] = 0
	bench.advance(6 * time.Minute)
	bench.tick(3)
	if bench.runtime.restartCount("stopped") != 0 {
		testCase.Fatal("le conteneur arrêté ne doit pas être relancé par le raccrochage")
	}
	if bench.runtime.restartCount("running") != 1 {
		testCase.Fatal("le conteneur en marche doit être raccroché")
	}
}

func TestManuallyStoppedGluetunNotRestarted(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer[gluetunName] = 1
	bench.runtime.stoppedContainers[gluetunName] = true
	bench.tick(5)
	if bench.runtime.restartCount(gluetunName) != 0 {
		testCase.Fatal("Gluetun arrêté à la main ne doit pas être relancé")
	}
}

func TestRestartSkippedWhenStateUnknown(testCase *testing.T) {
	bench := newBench(singleApp(), "")
	bench.runtime.pingExitCodeByContainer["app"] = 1
	bench.runtime.inspectErrorByContainer["app"] = errors.New("démon injoignable")
	bench.tick(5)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("état inconnu : pas de redémarrage")
	}
}

func TestInboundNetcatMissingIsNotCounted(testCase *testing.T) {
	bench := newBench(singleApp(), "app")
	bench.runtime.netcatExitCode = 127
	bench.tick(5)
	if bench.runtime.restartCount("app") != 0 {
		testCase.Fatal("un nc absent n'est pas une panne entrante")
	}
}
