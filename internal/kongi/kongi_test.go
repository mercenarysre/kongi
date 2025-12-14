package kongi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/mercenarysre/kongi/internal/config"
)

func TestProxy_ForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer backend.Close()

	port, _ := strconv.Atoi(backend.URL[len("http://localhost:"):])

	cfg := config.ProxyConfig{
		Paths: []config.PathConfig{
			{
				Path:        "/foo",
				Method:      "GET",
				ForwardPort: port,
				MaxRetry:    3,
				Timeout:     time.Second,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	if string(body) != "hello" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestProxy_PathNotFound(t *testing.T) {
	cfg := config.ProxyConfig{}

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestProxy_MethodNotAllowed(t *testing.T) {
	cfg := config.ProxyConfig{
		Paths: []config.PathConfig{
			{
				Path:   "/foo",
				Method: "GET",
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/foo", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestProxy_Timeout(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	port, _ := strconv.Atoi(backend.URL[len("http://localhost:"):])

	cfg := config.ProxyConfig{
		Paths: []config.PathConfig{
			{
				Path:        "/foo",
				Method:      "GET",
				ForwardPort: port,
				MaxRetry:    0,
				Timeout:     50 * time.Millisecond,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", rr.Code)
	}
}

func TestProxy_RetriesOn5xx(t *testing.T) {
	attempts := 0

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 4 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	port, _ := strconv.Atoi(backend.URL[len("http://localhost:"):])

	cfg := config.ProxyConfig{
		Paths: []config.PathConfig{
			{
				Path:        "/foo",
				Method:      "GET",
				ForwardPort: port,
				MaxRetry:    3,
				Timeout:     time.Second,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if attempts != 4 {
		t.Fatalf("expected 4 attempts, got %d", attempts)
	}
}
