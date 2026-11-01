package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/moby/moby/client"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/docker"
	"gluetun-hc/internal/gluetun"
	"gluetun-hc/internal/monitor"
)

const (
	controlAPITimeout  = 5 * time.Second
	inboundDialTimeout = 5 * time.Second
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	settings, loadError := config.Load(os.Getenv)
	if loadError != nil {
		logger.Error("configuration invalide", "error", loadError)
		os.Exit(1)
	}

	dockerClient, dockerError := client.New(client.FromEnv)
	if dockerError != nil {
		logger.Error("client Docker indisponible", "error", dockerError)
		os.Exit(1)
	}

	ownHostname, _ := os.Hostname()
	dockerRuntime := docker.NewRuntime(dockerClient, ownHostname)
	controlClient := gluetun.NewClient(settings.ControlURL, gluetun.Credentials{
		APIKey:   settings.ControlAPIKey,
		Username: settings.ControlUsername,
		Password: settings.ControlPassword,
	}, &http.Client{Timeout: controlAPITimeout})
	dialer := net.Dialer{Timeout: inboundDialTimeout}
	dial := func(operationContext context.Context, address string) error {
		connection, dialError := dialer.DialContext(operationContext, "tcp", address)
		if dialError != nil {
			return dialError
		}
		return connection.Close()
	}

	fileSource := gluetun.NewFileSource(dockerRuntime, settings.GluetunContainer, settings.PublicIPFile, settings.PortFile)
	portSource := gluetun.SelectSource(settings.ControlSource, fileSource, controlClient)

	runningMonitor := monitor.New(settings, dockerRuntime, portSource, dial, time.Now, logger)

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	ticker := time.NewTicker(settings.Interval)
	defer ticker.Stop()
	logger.Info("moniteur démarré", "gluetun", settings.GluetunContainer, "watched", len(settings.Watched))
	for {
		runningMonitor.Tick(shutdownContext)
		select {
		case <-shutdownContext.Done():
			logger.Info("arrêt")
			return
		case <-ticker.C:
		}
	}
}
