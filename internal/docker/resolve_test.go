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

func TestServiceFiltersAddProjectWhenKnown(testCase *testing.T) {
	filters := serviceFilters("gluetun", "stack-a")
	if !filters["label"]["com.docker.compose.service=gluetun"] || !filters["label"]["com.docker.compose.project=stack-a"] {
		testCase.Fatalf("filtres inattendus : %v", filters)
	}
}

func TestServiceFiltersOmitProjectWhenUnknown(testCase *testing.T) {
	filters := serviceFilters("gluetun", "")
	if len(filters["label"]) != 1 {
		testCase.Fatalf("seul le service est filtré : %v", filters)
	}
}
