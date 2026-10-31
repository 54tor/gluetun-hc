package docker

import (
	"errors"
	"testing"
)

func TestExecutableNotFoundMapsToExitCode127(testCase *testing.T) {
	startError := errors.New(`OCI runtime exec failed: exec failed: unable to start container process: exec: "ping": executable file not found in $PATH`)
	exitCode, recognized := exitCodeFromStartError(startError)
	if !recognized || exitCode != 127 {
		testCase.Fatalf("code %d, reconnu %v", exitCode, recognized)
	}
}

func TestOtherStartErrorIsNotRecognized(testCase *testing.T) {
	_, recognized := exitCodeFromStartError(errors.New("connexion au démon perdue"))
	if recognized {
		testCase.Fatal("une autre erreur ne doit pas être convertie en code de sortie")
	}
}
