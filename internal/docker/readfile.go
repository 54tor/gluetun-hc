package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

func (runtime *Runtime) ReadFile(operationContext context.Context, requested, path string) (string, error) {
	container, resolveError := runtime.resolve(operationContext, requested)
	if resolveError != nil {
		return "", resolveError
	}
	created, dockerError := runtime.dockerClient.ExecCreate(operationContext, container, client.ExecCreateOptions{
		Cmd:          []string{"cat", path},
		AttachStdout: true,
		AttachStderr: true,
	})
	if dockerError != nil {
		return "", wrapUnavailable(dockerError)
	}

	attached, dockerError := runtime.dockerClient.ExecAttach(operationContext, created.ID, client.ExecAttachOptions{})
	if dockerError != nil {
		return "", wrapUnavailable(dockerError)
	}
	defer attached.Close()

	var standardOutput, standardError bytes.Buffer
	if _, copyError := stdcopy.StdCopy(&standardOutput, &standardError, attached.Reader); copyError != nil && copyError != io.EOF {
		return "", copyError
	}

	inspected, dockerError := runtime.dockerClient.ExecInspect(operationContext, created.ID, client.ExecInspectOptions{})
	if dockerError != nil {
		return "", dockerError
	}
	if inspected.ExitCode != 0 {
		return "", fmt.Errorf("cat %s : code %d : %s", path, inspected.ExitCode, bytes.TrimSpace(standardError.Bytes()))
	}
	return standardOutput.String(), nil
}
