package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/mercenarysre/kongi/internal/config"
	"github.com/mercenarysre/kongi/internal/kongi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler := kongi.ProxyHandler(cfg)

	go func() {
		http.Handle(cfg.Metrics.Path, promhttp.Handler())
		log.Printf("Metrics server listening on :%d%s", cfg.Metrics.Port, cfg.Metrics.Path)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.Metrics.Port), nil))
	}()

	log.Printf("Proxy listening on :%d", cfg.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(cfg.Port), handler)
	if err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}
}
