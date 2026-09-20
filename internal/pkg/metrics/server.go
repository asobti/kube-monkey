package metrics

import (
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	endpointPath = "/metrics"

	// Stops a client from holding a connection open by never finishing its
	// request headers.
	readHeaderTimeout = 10 * time.Second
)

// Handler serves the recorded metrics in the Prometheus text format.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry})
}

// Serve starts the metrics endpoint in the background. The address is bound
// before returning, so an address already in use or malformed fails at startup
// instead of leaving kube-monkey running with no way to scrape it.
func Serve(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	serve(listener)
	return nil
}

func serve(listener net.Listener) {
	mux := http.NewServeMux()
	mux.Handle(endpointPath, Handler())
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			glog.Errorf("Metrics endpoint stopped serving. Error: %v", err)
		}
	}()

	glog.V(1).Infof("Serving metrics on http://%s%s", listener.Addr(), endpointPath)
}
