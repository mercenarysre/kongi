package kongi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mercenarysre/kongi/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"path", "method"},
	)
	ResponseCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_responses_total", Help: "Total HTTP responses"},
		[]string{"path", "method", "status_code", "outcome"},
	)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Request duration"},
		[]string{"path", "method"},
	)
	RetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_retries_total", Help: "Retry attempts"},
		[]string{"path", "method"},
	)
	TimeoutsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_timeouts_total", Help: "Timed-out requests"},
		[]string{"path", "method"},
	)
)

func init() {
	prometheus.MustRegister(RequestCounter, ResponseCounter, RequestDuration, RetriesTotal, TimeoutsTotal)
}

func ProxyHandler(cfg config.ProxyConfig) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		var route *config.PathConfig
		for i := range cfg.Paths {
			if cfg.Paths[i].Path == req.URL.Path {
				route = &cfg.Paths[i]
				break
			}
		}

		if route == nil {
			http.Error(rw, "path not found", http.StatusNotFound)
			return
		}

		if req.Method != route.Method {
			ResponseCounter.WithLabelValues(route.Path, req.Method, "405", "error").Inc()
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		RequestCounter.WithLabelValues(route.Path, req.Method).Inc()

		target := &url.URL{
			Scheme: "http",
			Host:   "localhost:" + strconv.Itoa(route.ForwardPort),
			Path:   "/",
		}

		client := &http.Client{Timeout: route.Timeout}
		attempt := 0

		for {
			// clone request for origin server
			cloned := req.Clone(req.Context())
			cloned.URL = target
			cloned.Host = target.Host
			cloned.RequestURI = ""

			resp, err := client.Do(cloned)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					TimeoutsTotal.WithLabelValues(route.Path, req.Method).Inc()
					http.Error(rw, "request	timeout", http.StatusGatewayTimeout)
					ResponseCounter.WithLabelValues(route.Path, req.Method, "504", "error").Inc()
					return
				}

				http.Error(rw, "Internal Server Error", http.StatusBadGateway)
				ResponseCounter.WithLabelValues(route.Path, req.Method, "502", "error").Inc()
				return
			}

			// Retry on 5xx
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				attempt++

				if attempt <= route.MaxRetry {
					RetriesTotal.WithLabelValues(route.Path, req.Method).Inc()
					continue
				}

				http.Error(rw, "Internal Server Error", http.StatusBadGateway)
				ResponseCounter.WithLabelValues(route.Path, req.Method, strconv.Itoa(resp.StatusCode), "error").Inc()
				return
			}

			RequestDuration.WithLabelValues(route.Path, req.Method).Observe(time.Since(start).Seconds())
			ResponseCounter.WithLabelValues(route.Path, req.Method, strconv.Itoa(resp.StatusCode), "success").Inc()

			for k, vals := range resp.Header {
				for _, v := range vals {
					rw.Header().Add(k, v)
				}
			}

			rw.WriteHeader(resp.StatusCode)
			io.Copy(rw, resp.Body)
			resp.Body.Close()
			return
		}
	})
}
