package exporter

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version string

type ServerConfig struct {
	Address  string
	CertFile string
	KeyFile  string
}

func GetVersion() string {
	return version
}

func NewServerConfig() ServerConfig {

	sc := ServerConfig{}

	return sc

}

func StartMetricsServer(config ServerConfig) {
	d := http.NewServeMux()
	d.Handle("/metrics", promhttp.Handler())
	d.HandleFunc("/version", showVersion)

	var err error

	if config.CertFile != "" && config.KeyFile != "" {
		err = http.ListenAndServeTLS(config.Address, config.CertFile, config.KeyFile, d)
	} else {
		err = http.ListenAndServe(config.Address, d)
	}

	if err != nil {
		log.Fatal("Failed to start metrics server, error is:", err)
	}
}

func showVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, version)
}
