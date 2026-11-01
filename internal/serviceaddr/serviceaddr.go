package serviceaddr

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Resolver interface {
	ServiceAddress(operationContext context.Context, service string) (address string, isService bool, resolveError error)
}

func NewHTTPClient(timeout time.Duration, resolver Resolver) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		DialContext: func(dialContext context.Context, network, address string) (net.Conn, error) {
			resolvedAddress, resolveError := resolveDialAddress(dialContext, resolver, address)
			if resolveError != nil {
				return nil, resolveError
			}
			return dialer.DialContext(dialContext, network, resolvedAddress)
		},
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func resolveDialAddress(operationContext context.Context, resolver Resolver, address string) (string, error) {
	host, port, splitError := net.SplitHostPort(address)
	if splitError != nil || net.ParseIP(host) != nil {
		return address, nil
	}
	serviceAddress, isService, resolveError := resolver.ServiceAddress(operationContext, host)
	if resolveError != nil {
		return "", resolveError
	}
	if !isService {
		return address, nil
	}
	return net.JoinHostPort(serviceAddress, port), nil
}

func RewriteURL(operationContext context.Context, rawURL string, resolver Resolver) (string, error) {
	parsed, parseError := url.Parse(rawURL)
	if parseError != nil {
		return "", parseError
	}
	hostname := parsed.Hostname()
	if hostname == "" || net.ParseIP(hostname) != nil {
		return rawURL, nil
	}
	serviceAddress, isService, resolveError := resolver.ServiceAddress(operationContext, hostname)
	if resolveError != nil {
		return "", resolveError
	}
	if !isService {
		return rawURL, nil
	}
	if port := parsed.Port(); port != "" {
		parsed.Host = net.JoinHostPort(serviceAddress, port)
	} else {
		parsed.Host = serviceAddress
	}
	return parsed.String(), nil
}
