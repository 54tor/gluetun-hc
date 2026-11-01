package healthserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultAddress = "127.0.0.1:8081"
	healthPath     = "/healthz"
	checkTimeout   = 3 * time.Second
)

type Checker interface {
	Healthy() bool
}

func AddressFromEnvironment(getenv func(string) string) string {
	if address := getenv("HEALTH_ADDRESS"); address != "" {
		return address
	}
	return defaultAddress
}

func NewHandler(checker Checker) http.Handler {
	multiplexer := http.NewServeMux()
	multiplexer.HandleFunc(healthPath, func(writer http.ResponseWriter, request *http.Request) {
		if checker.Healthy() {
			writer.WriteHeader(http.StatusOK)
			return
		}
		writer.WriteHeader(http.StatusServiceUnavailable)
	})
	return multiplexer
}

func Serve(serveContext context.Context, address string, checker Checker) error {
	server := &http.Server{Addr: address, Handler: NewHandler(checker), ReadHeaderTimeout: checkTimeout}
	go func() {
		<-serveContext.Done()
		server.Close()
	}()
	serveError := server.ListenAndServe()
	if errors.Is(serveError, http.ErrServerClosed) {
		return nil
	}
	return serveError
}

func Check(address string) error {
	httpClient := &http.Client{Timeout: checkTimeout}
	response, requestError := httpClient.Get("http://" + address + healthPath)
	if requestError != nil {
		return requestError
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("statut %d", response.StatusCode)
	}
	return nil
}
