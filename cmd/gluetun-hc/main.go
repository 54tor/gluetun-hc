package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gluetun-hc/internal/config"
	"gluetun-hc/internal/healthserver"
	"gluetun-hc/internal/monitor"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}
	os.Exit(runMonitor())
}

func runHealthcheck() int {
	if checkError := healthserver.Check(healthserver.AddressFromEnvironment(os.Getenv)); checkError != nil {
		os.Stderr.WriteString("pas sain : " + checkError.Error() + "\n")
		return 1
	}
	return 0
}

func runMonitor() int {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	settings, loadError := config.Load(os.Getenv)
	if loadError != nil {
		logger.Error("configuration invalide", "error", loadError)
		return 1
	}

	runningMonitor, buildError := buildMonitor(settings, logger)
	if buildError != nil {
		logger.Error("initialisation impossible", "error", buildError)
		return 1
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go serveHealth(shutdownContext, runningMonitor, logger)
	logger.Info("moniteur démarré", "gluetun", settings.GluetunContainer, "watched", len(settings.Watched))
	runLoop(shutdownContext, runningMonitor, settings.Interval)
	logger.Info("arrêt")
	return 0
}

func serveHealth(serveContext context.Context, runningMonitor *monitor.Monitor, logger *slog.Logger) {
	address := healthserver.AddressFromEnvironment(os.Getenv)
	if serveError := healthserver.Serve(serveContext, address, runningMonitor); serveError != nil {
		logger.Error("serveur de santé arrêté", "address", address, "error", serveError)
	}
}

func runLoop(loopContext context.Context, runningMonitor *monitor.Monitor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		runningMonitor.Tick(loopContext)
		select {
		case <-loopContext.Done():
			return
		case <-ticker.C:
		}
	}
}
