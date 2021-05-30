package openldap_exporter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var commit string
var tag string

func GetVersion() string {
	return fmt.Sprintf("%s (%s)", tag, commit)
}

type Server struct {
	server  *http.Server
	cfgPath string
}

func (s *Server) Start() error {
	err := web.ListenAndServe(s.server, s.cfgPath, kitlog.LoggerFunc(logger))
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	s.server.Shutdown(ctx)
	cancel()
}

func NewMetricsServer(bindAddr, metricsPath, tlsConfigPath string) *Server {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.HandleFunc("/version", showVersion)
	return &Server{
		server:  &http.Server{Addr: bindAddr, Handler: mux},
		cfgPath: tlsConfigPath,
	}
}

func showVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, GetVersion())
}

func logger(kvs ...interface{}) error {
	if len(kvs) == 0 {
		return nil
	}
	if len(kvs)%2 != 0 {
		kvs = append(kvs, nil)
	}
	var buf strings.Builder
	for i := 0; i < len(kvs); i += 2 {
		if i > 0 {
			buf.WriteString(" ")
		}
		fmt.Fprintf(&buf, "%v=%v", kvs[i], kvs[i+1])
	}
	log.Println(buf.String())
	return nil
}
