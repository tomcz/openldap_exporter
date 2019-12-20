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

func StartMetricsServer(bindAddr string, path string) {
	d := http.NewServeMux()
	d.Handle(path, promhttp.Handler())
	d.HandleFunc("/version", showVersion)

	err := http.ListenAndServe(bindAddr, d)
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
