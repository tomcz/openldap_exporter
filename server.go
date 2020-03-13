package openldap_exporter

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version string

func GetVersion() string {
	return version
}

func StartMetricsServer(bindAddr, metricsPath string) {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.HandleFunc("/version", showVersion)

	err := http.ListenAndServe(bindAddr, mux)
	if err != nil {
		log.Fatal("http listener failed, error is:", err)
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
