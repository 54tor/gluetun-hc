package gluetun

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type FileReader interface {
	ReadFile(operationContext context.Context, container, path string) (string, error)
}

type FileSource struct {
	reader       FileReader
	container    string
	publicIPPath string
	portPath     string
}

func NewFileSource(reader FileReader, container, publicIPPath, portPath string) *FileSource {
	return &FileSource{reader: reader, container: container, publicIPPath: publicIPPath, portPath: portPath}
}

func (source *FileSource) PublicIP(operationContext context.Context) (string, error) {
	content, readError := source.reader.ReadFile(operationContext, source.container, source.publicIPPath)
	if readError != nil {
		return "", fmt.Errorf("lecture de %s : %w", source.publicIPPath, readError)
	}
	return strings.TrimSpace(content), nil
}

func (source *FileSource) ForwardedPort(operationContext context.Context) (int, error) {
	content, readError := source.reader.ReadFile(operationContext, source.container, source.portPath)
	if readError != nil {
		return 0, fmt.Errorf("lecture de %s : %w", source.portPath, readError)
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return 0, ErrNoForwardedPort
	}
	port, parseError := strconv.Atoi(trimmed)
	if parseError != nil {
		return 0, fmt.Errorf("contenu de %s invalide : %w", source.portPath, parseError)
	}
	return port, nil
}
