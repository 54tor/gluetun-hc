package docker

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/moby/moby/client"
)

const integrationContainerName = "gluetun-hc-integration-check"

func startIntegrationContainer(testCase *testing.T) *Runtime {
	if os.Getenv("GLUETUN_HC_INTEGRATION") == "" {
		testCase.Skip("GLUETUN_HC_INTEGRATION non défini")
	}
	exec.Command("docker", "rm", "-f", integrationContainerName).Run()
	startCommand := exec.Command("docker", "run", "-d", "--name", integrationContainerName, "alpine:latest",
		"sh", "-c", "mkdir -p /tmp/gluetun && echo 5914 > /tmp/gluetun/forwarded_port && sleep 300")
	if output, startError := startCommand.CombinedOutput(); startError != nil {
		testCase.Fatalf("démarrage impossible : %v : %s", startError, output)
	}
	testCase.Cleanup(func() {
		exec.Command("docker", "rm", "-f", integrationContainerName).Run()
	})

	dockerClient, clientError := client.New(client.FromEnv)
	if clientError != nil {
		testCase.Fatal(clientError)
	}
	return NewRuntime(dockerClient)
}

func TestIntegrationReadFile(testCase *testing.T) {
	runtime := startIntegrationContainer(testCase)
	content, readError := runtime.ReadFile(context.Background(), integrationContainerName, "/tmp/gluetun/forwarded_port")
	if readError != nil || strings.TrimSpace(content) != "5914" {
		testCase.Fatalf("contenu %q, erreur %v", content, readError)
	}
	if _, missingError := runtime.ReadFile(context.Background(), integrationContainerName, "/tmp/gluetun/absent"); missingError == nil {
		testCase.Fatal("un fichier absent doit renvoyer une erreur")
	}
}

func TestIntegrationMissingBinaryGivesExitCode127(testCase *testing.T) {
	runtime := startIntegrationContainer(testCase)
	exitCode, execError := runtime.Exec(context.Background(), integrationContainerName, []string{"binaire-inexistant"})
	if execError != nil || exitCode != 127 {
		testCase.Fatalf("code %d, erreur %v", exitCode, execError)
	}
}

func TestIntegrationIsRunningFollowsContainerState(testCase *testing.T) {
	runtime := startIntegrationContainer(testCase)
	running, inspectError := runtime.IsRunning(context.Background(), integrationContainerName)
	if inspectError != nil || !running {
		testCase.Fatalf("doit tourner : %v %v", running, inspectError)
	}
	exec.Command("docker", "stop", "-t", "1", integrationContainerName).Run()
	running, inspectError = runtime.IsRunning(context.Background(), integrationContainerName)
	if inspectError != nil || running {
		testCase.Fatalf("doit être arrêté : %v %v", running, inspectError)
	}
}
