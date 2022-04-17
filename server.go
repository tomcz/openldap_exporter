package openldap_exporter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	log "github.com/sirupsen/logrus"
)

var commit string
var tag string

func GetVersion() string {
	return fmt.Sprintf("%s (%s)", tag, commit)
}

type Server struct {
	server  *http.Server
	logger  log.FieldLogger
	cfgPath string
}

func NewMetricsServer(bindAddr, metricsPath, tlsConfigPath string) *Server {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.HandleFunc("/version", showVersion)
	return &Server{
		server:  &http.Server{Addr: bindAddr, Handler: mux},
		logger:  log.WithField("component", "server"),
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

func (s *Server) Start() error {
	s.logger.WithField("addr", s.server.Addr).Info("starting http listener")
	err := web.ListenAndServe(s.server, s.cfgPath, kitlog.LoggerFunc(s.adaptor))
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

func (s *Server) adaptor(kvs ...interface{}) error {
	if len(kvs) == 0 {
		return nil
	}
	if len(kvs)%2 != 0 {
		kvs = append(kvs, nil)
	}
	fields := log.Fields{}
	for i := 0; i < len(kvs); i += 2 {
		key := fmt.Sprint(kvs[i])
		fields[key] = kvs[i+1]
	}
	var msg string
	if val, ok := fields["msg"]; ok {
		delete(fields, "msg")
		msg = fmt.Sprint(val)
	}
	var level string
	if val, ok := fields["level"]; ok {
		delete(fields, "level")
		level = fmt.Sprint(val)
	}
	ll := s.logger.WithFields(fields)
	switch level {
	case "error":
		ll.Error(msg)
	case "warn":
		ll.Warn(msg)
	case "debug":
		ll.Debug(msg)
	default:
		ll.Info(msg)
	}
	return nil
}
