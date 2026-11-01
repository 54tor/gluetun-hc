package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/moby/moby/client"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/docker"
	"gluetun-hc/internal/gluetun"
	"gluetun-hc/internal/monitor"
	"gluetun-hc/internal/portupdate"
)

const controlAPITimeout = 5 * time.Second

func buildMonitor(settings config.Config, logger *slog.Logger) (*monitor.Monitor, error) {
	dockerClient, dockerError := client.New(client.FromEnv)
	if dockerError != nil {
		return nil, dockerError
	}
	ownHostname, _ := os.Hostname()
	dockerRuntime := docker.NewRuntime(dockerClient, ownHostname)

	runningMonitor := monitor.New(settings, dockerRuntime, buildPortSource(settings, dockerRuntime), time.Now, logger)
	if settings.PortUpdate.Mode != "" {
		updater := portupdate.FromConfig(settings.PortUpdate, &http.Client{Timeout: controlAPITimeout})
		runningMonitor.EnablePortSync(updater, settings.PortUpdate.Container)
	}
	return runningMonitor, nil
}

func buildPortSource(settings config.Config, dockerRuntime *docker.Runtime) gluetun.Source {
	controlClient := gluetun.NewClient(settings.ControlURL, gluetun.Credentials{
		APIKey:   settings.ControlAPIKey,
		Username: settings.ControlUsername,
		Password: settings.ControlPassword,
	}, &http.Client{Timeout: controlAPITimeout})
	fileSource := gluetun.NewFileSource(dockerRuntime, settings.GluetunContainer, settings.PublicIPFile, settings.PortFile)
	return gluetun.SelectSource(settings.ControlSource, fileSource, controlClient)
}
