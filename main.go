package main

import (
	"context"
	"errors"
	"fmt"
	log "log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/tomcz/gotools/errgroup"
	"github.com/tomcz/gotools/maps"
	"github.com/tomcz/gotools/quiet"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"gopkg.in/ldap.v2"
)

const (
	promAddr          = "promAddr"
	ldapNet           = "ldapNet"
	ldapAddr          = "ldapAddr"
	ldapUser          = "ldapUser"
	ldapPass          = "ldapPass"
	interval          = "interval"
	metrics           = "metrPath"
	jsonLog           = "jsonLog"
	webCfgFile        = "webCfgFile"
	config            = "config"
	replicationObject = "replicationObject"
)

var showStop bool

func main() {
	flags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    promAddr,
			Value:   ":9330",
			Usage:   "Bind address for Prometheus HTTP metrics server",
			EnvVars: []string{"PROM_ADDR"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    metrics,
			Value:   "/metrics",
			Usage:   "Path on which to expose Prometheus metrics",
			EnvVars: []string{"METRICS_PATH"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    ldapNet,
			Value:   "tcp",
			Usage:   "Network of OpenLDAP server",
			EnvVars: []string{"LDAP_NET"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    ldapAddr,
			Value:   "localhost:389",
			Usage:   "Address and port of OpenLDAP server",
			EnvVars: []string{"LDAP_ADDR"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    ldapUser,
			Usage:   "OpenLDAP bind username (optional)",
			EnvVars: []string{"LDAP_USER"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    ldapPass,
			Usage:   "OpenLDAP bind password (optional)",
			EnvVars: []string{"LDAP_PASS"},
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:    interval,
			Value:   30 * time.Second,
			Usage:   "Scrape interval",
			EnvVars: []string{"INTERVAL"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:    jsonLog,
			Value:   false,
			Usage:   "Output logs in JSON format",
			EnvVars: []string{"JSON_LOG"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    webCfgFile,
			Usage:   "Prometheus metrics web config `FILE` (optional)",
			EnvVars: []string{"WEB_CFG_FILE"},
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:  replicationObject,
			Usage: "Object to watch replication upon",
		}),
		&cli.StringFlag{
			Name:  config,
			Usage: "Optional configuration from a `YAML_FILE`",
		},
	}
	app := &cli.App{
		Name:            "openldap_exporter",
		Usage:           "Export OpenLDAP metrics to Prometheus",
		Before:          altsrc.InitInputSourceWithContext(flags, optionalYamlSourceFunc(config)),
		Version:         GetVersion(),
		HideHelpCommand: true,
		Flags:           flags,
		Action:          runMain,
	}
	if err := app.Run(os.Args); err != nil {
		log.Error("service failed", "err", err)
		os.Exit(1)
	}
	if showStop {
		log.Info("service stopped")
	}
}

func optionalYamlSourceFunc(flagFileName string) func(context *cli.Context) (altsrc.InputSourceContext, error) {
	return func(c *cli.Context) (altsrc.InputSourceContext, error) {
		filePath := c.String(flagFileName)
		if filePath != "" {
			return altsrc.NewYamlSourceFromFile(filePath)
		}
		return &altsrc.MapInputSource{}, nil
	}
}

func runMain(c *cli.Context) error {
	showStop = true

	if c.Bool(jsonLog) {
		lh := log.NewJSONHandler(os.Stderr, nil)
		log.SetDefault(log.New(lh))
	}
	log.Info("service starting")

	server := NewMetricsServer(
		c.String(promAddr),
		c.String(metrics),
		c.String(webCfgFile),
	)

	scraper := &Scraper{
		Net:  c.String(ldapNet),
		Addr: c.String(ldapAddr),
		User: c.String(ldapUser),
		Pass: c.String(ldapPass),
		Tick: c.Duration(interval),
		Sync: c.StringSlice(replicationObject),
	}

	ctx, cancel := context.WithCancel(context.Background())
	group := errgroup.New()
	group.Go(func() error {
		defer cancel()
		return server.Start()
	})
	group.Go(func() error {
		defer cancel()
		scraper.Start(ctx)
		return nil
	})
	group.Go(func() error {
		defer func() {
			cancel()
			server.Stop()
		}()
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-signalChan:
			log.Info("shutdown received")
			return nil
		case <-ctx.Done():
			return nil
		}
	})
	return group.Wait()
}

// ===============================================================
// Metrics Scraper
// ===============================================================

const (
	baseDN    = "cn=Monitor"
	opsBaseDN = "cn=Operations,cn=Monitor"

	monitorCounterObject = "monitorCounterObject"
	monitorCounter       = "monitorCounter"

	monitoredObject = "monitoredObject"
	monitoredInfo   = "monitoredInfo"

	monitorOperation   = "monitorOperation"
	monitorOpCompleted = "monitorOpCompleted"

	monitorReplicationFilter = "contextCSN"
	monitorReplication       = "monitorReplication"
)

type query struct {
	baseDN       string
	searchFilter string
	searchAttr   string
	metric       *prometheus.GaugeVec
	setData      func([]*ldap.Entry, *query)
}

var (
	monitoredObjectGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitored_object",
			Help:      help(baseDN, objectClass(monitoredObject), monitoredInfo),
		},
		[]string{"dn"},
	)
	monitorCounterObjectGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitor_counter_object",
			Help:      help(baseDN, objectClass(monitorCounterObject), monitorCounter),
		},
		[]string{"dn"},
	)
	monitorOperationGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitor_operation",
			Help:      help(opsBaseDN, objectClass(monitorOperation), monitorOpCompleted),
		},
		[]string{"dn"},
	)
	bindCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "openldap",
			Name:      "bind",
			Help:      "successful vs unsuccessful ldap bind attempts",
		},
		[]string{"result"},
	)
	dialCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "openldap",
			Name:      "dial",
			Help:      "successful vs unsuccessful ldap dial attempts",
		},
		[]string{"result"},
	)
	scrapeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "openldap",
			Name:      "scrape",
			Help:      "successful vs unsuccessful ldap scrape attempts",
		},
		[]string{"result"},
	)
	monitorReplicationGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitor_replication",
			Help:      help(baseDN, monitorReplication),
		},
		[]string{"id", "type"},
	)
	queries = []*query{
		{
			baseDN:       baseDN,
			searchFilter: objectClass(monitoredObject),
			searchAttr:   monitoredInfo,
			metric:       monitoredObjectGauge,
			setData:      setValue,
		}, {
			baseDN:       baseDN,
			searchFilter: objectClass(monitorCounterObject),
			searchAttr:   monitorCounter,
			metric:       monitorCounterObjectGauge,
			setData:      setValue,
		},
		{
			baseDN:       opsBaseDN,
			searchFilter: objectClass(monitorOperation),
			searchAttr:   monitorOpCompleted,
			metric:       monitorOperationGauge,
			setData:      setValue,
		},
		{
			baseDN:       opsBaseDN,
			searchFilter: objectClass(monitorOperation),
			searchAttr:   monitorOpCompleted,
			metric:       monitorOperationGauge,
			setData:      setValue,
		},
	}
)

func init() {
	prometheus.MustRegister(
		monitoredObjectGauge,
		monitorCounterObjectGauge,
		monitorOperationGauge,
		monitorReplicationGauge,
		scrapeCounter,
		bindCounter,
		dialCounter,
	)
}

func help(msg ...string) string {
	return strings.Join(msg, " ")
}

func objectClass(name string) string {
	return fmt.Sprintf("(objectClass=%v)", name)
}

func setValue(entries []*ldap.Entry, q *query) {
	for _, entry := range entries {
		val := entry.GetAttributeValue(q.searchAttr)
		if val == "" {
			// not every entry will have this attribute
			continue
		}
		num, err := strconv.ParseFloat(val, 64)
		if err != nil {
			// some of these attributes are not numbers
			continue
		}
		q.metric.WithLabelValues(entry.DN).Set(num)
	}
}

type Scraper struct {
	Net      string
	Addr     string
	User     string
	Pass     string
	Tick     time.Duration
	LdapSync []string
	log      *log.Logger
	Sync     []string
}

func (s *Scraper) Start(ctx context.Context) {
	s.log = log.With("component", "scraper")
	s.addReplicationQueries()
	address := fmt.Sprintf("%s://%s", s.Net, s.Addr)
	s.log.Info("starting monitor loop", "addr", address)
	ticker := time.NewTicker(s.Tick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.scrape()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scraper) addReplicationQueries() {
	for _, q := range s.Sync {
		queries = append(queries,
			&query{
				baseDN:       q,
				searchFilter: objectClass("*"),
				searchAttr:   monitorReplicationFilter,
				metric:       monitorReplicationGauge,
				setData:      s.setReplicationValue,
			},
		)
	}
}

func (s *Scraper) setReplicationValue(entries []*ldap.Entry, q *query) {
	for _, entry := range entries {
		val := entry.GetAttributeValue(q.searchAttr)
		if val == "" {
			// not every entry will have this attribute
			continue
		}
		ll := s.log.With(
			"filter", q.searchFilter,
			"attr", q.searchAttr,
			"value", val,
		)
		valueBuffer := strings.Split(val, "#")
		gt, err := time.Parse("20060102150405.999999Z", valueBuffer[0])
		if err != nil {
			ll.Warn("unexpected gt value", "err", err)
			continue
		}
		count, err := strconv.ParseFloat(valueBuffer[1], 64)
		if err != nil {
			ll.Warn("unexpected count value", "err", err)
			continue
		}
		sid := valueBuffer[2]
		mod, err := strconv.ParseFloat(valueBuffer[3], 64)
		if err != nil {
			ll.Warn("unexpected mod value", "err", err)
			continue
		}
		q.metric.WithLabelValues(sid, "gt").Set(float64(gt.Unix()))
		q.metric.WithLabelValues(sid, "count").Set(count)
		q.metric.WithLabelValues(sid, "mod").Set(mod)
	}
}

func (s *Scraper) scrape() {
	conn, err := ldap.Dial(s.Net, s.Addr)
	if err != nil {
		s.log.Error("dial failed")
		dialCounter.WithLabelValues("fail").Inc()
		return
	}
	dialCounter.WithLabelValues("ok").Inc()
	defer conn.Close()

	if s.User != "" && s.Pass != "" {
		err = conn.Bind(s.User, s.Pass)
		if err != nil {
			s.log.Error("bind failed", "err", err)
			bindCounter.WithLabelValues("fail").Inc()
			return
		}
		bindCounter.WithLabelValues("ok").Inc()
	}

	scrapeRes := "ok"
	for _, q := range queries {
		if err = scrapeQuery(conn, q); err != nil {
			s.log.Warn("query failed", "filter", q.searchFilter, "err", err)
			scrapeRes = "fail"
		}
	}
	scrapeCounter.WithLabelValues(scrapeRes).Inc()
}

func scrapeQuery(conn *ldap.Conn, q *query) error {
	req := ldap.NewSearchRequest(
		q.baseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		q.searchFilter, []string{q.searchAttr}, nil,
	)
	sr, err := conn.Search(req)
	if err != nil {
		return err
	}
	q.setData(sr.Entries, q)
	return nil
}

// ===============================================================
// Metrics server
// ===============================================================

var commit string
var tag string

func GetVersion() string {
	return fmt.Sprintf("%s (%s)", tag, commit)
}

type Server struct {
	server  *http.Server
	logger  *log.Logger
	cfgPath string
}

func NewMetricsServer(bindAddr, metricsPath, tlsConfigPath string) *Server {
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.HandleFunc("/version", showVersion)
	return &Server{
		server:  &http.Server{Addr: bindAddr, Handler: mux},
		logger:  log.With("component", "server"),
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
	s.logger.Info("starting http listener", "addr", s.server.Addr)
	err := web.ListenAndServe(s.server, s.cfgPath, kitlog.LoggerFunc(s.adaptor))
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Stop() {
	quiet.CloseWithTimeout(s.server.Shutdown, 100*time.Millisecond)
}

func (s *Server) adaptor(kvs ...interface{}) error {
	if len(kvs) == 0 {
		return nil
	}
	if len(kvs)%2 != 0 {
		kvs = append(kvs, nil)
	}
	fields := make(map[string]any)
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
	var args []any
	for _, e := range maps.SortedEntries(fields) {
		args = append(args, e.Key, e.Val)
	}
	ll := s.logger.With(args...)
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
