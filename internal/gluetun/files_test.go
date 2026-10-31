package gluetun

import (
	"context"
	"errors"
	"testing"
)

type fakeFileReader struct {
	contentByPath map[string]string
	readError     error
}

func (reader *fakeFileReader) ReadFile(operationContext context.Context, container, path string) (string, error) {
	if reader.readError != nil {
		return "", reader.readError
	}
	content, exists := reader.contentByPath[path]
	if !exists {
		return "", errors.New("fichier absent")
	}
	return content, nil
}

func newFileSource(contentByPath map[string]string) *FileSource {
	reader := &fakeFileReader{contentByPath: contentByPath}
	return NewFileSource(reader, "gluetun", "/tmp/gluetun/ip", "/tmp/gluetun/forwarded_port")
}

func TestFileSourcePublicIPIsTrimmed(testCase *testing.T) {
	source := newFileSource(map[string]string{"/tmp/gluetun/ip": "203.0.113.7\n"})
	publicIP, readError := source.PublicIP(context.Background())
	if readError != nil || publicIP != "203.0.113.7" {
		testCase.Fatalf("IP %q, erreur %v", publicIP, readError)
	}
}

func TestFileSourceForwardedPortParsed(testCase *testing.T) {
	source := newFileSource(map[string]string{"/tmp/gluetun/forwarded_port": "5914\n"})
	port, readError := source.ForwardedPort(context.Background())
	if readError != nil || port != 5914 {
		testCase.Fatalf("port %d, erreur %v", port, readError)
	}
}

func TestFileSourceEmptyPortFileMeansNoForwardedPort(testCase *testing.T) {
	source := newFileSource(map[string]string{"/tmp/gluetun/forwarded_port": "\n"})
	_, readError := source.ForwardedPort(context.Background())
	if !errors.Is(readError, ErrNoForwardedPort) {
		testCase.Fatalf("erreur inattendue : %v", readError)
	}
}

func TestFileSourceRejectsNonNumericPort(testCase *testing.T) {
	source := newFileSource(map[string]string{"/tmp/gluetun/forwarded_port": "abc"})
	_, readError := source.ForwardedPort(context.Background())
	if readError == nil || errors.Is(readError, ErrNoForwardedPort) {
		testCase.Fatalf("erreur de format attendue : %v", readError)
	}
}

func TestFileSourceMissingFileReturnsError(testCase *testing.T) {
	source := newFileSource(map[string]string{})
	if _, readError := source.PublicIP(context.Background()); readError == nil {
		testCase.Fatal("erreur attendue")
	}
}
