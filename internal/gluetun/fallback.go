package gluetun

import (
	"context"
	"errors"
)

type Source interface {
	PublicIP(operationContext context.Context) (string, error)
	ForwardedPort(operationContext context.Context) (int, error)
}

type Fallback struct {
	primary   Source
	secondary Source
}

func NewFallback(primary, secondary Source) *Fallback {
	return &Fallback{primary: primary, secondary: secondary}
}

func (fallback *Fallback) PublicIP(operationContext context.Context) (string, error) {
	publicIP, primaryError := fallback.primary.PublicIP(operationContext)
	if primaryError == nil && publicIP != "" {
		return publicIP, nil
	}
	return fallback.secondary.PublicIP(operationContext)
}

func (fallback *Fallback) ForwardedPort(operationContext context.Context) (int, error) {
	port, primaryError := fallback.primary.ForwardedPort(operationContext)
	if primaryError == nil || errors.Is(primaryError, ErrNoForwardedPort) {
		return port, primaryError
	}
	return fallback.secondary.ForwardedPort(operationContext)
}
