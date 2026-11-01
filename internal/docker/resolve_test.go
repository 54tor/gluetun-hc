package docker

import "testing"

func TestChooseContainerNameKeepsRequestedNameWhenNoServiceMatches(testCase *testing.T) {
	chosen, chooseError := chooseContainerName("gluetun", nil)
	if chooseError != nil || chosen != "gluetun" {
		testCase.Fatalf("nom %q, erreur %v", chosen, chooseError)
	}
}

func TestChooseContainerNameUsesTheSingleServiceMatch(testCase *testing.T) {
	chosen, chooseError := chooseContainerName("gluetun", []string{"/stack-gluetun-1"})
	if chooseError != nil || chosen != "stack-gluetun-1" {
		testCase.Fatalf("nom %q, erreur %v", chosen, chooseError)
	}
}

func TestChooseContainerNameRejectsAmbiguousServiceName(testCase *testing.T) {
	_, chooseError := chooseContainerName("gluetun", []string{"/a-gluetun-1", "/b-gluetun-1"})
	if chooseError == nil {
		testCase.Fatal("deux projets avec le même service doivent être refusés")
	}
}
