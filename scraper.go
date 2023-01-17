package openldap_exporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ldap.v2"
)

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

	SchemeLDAPS = "ldaps"
	SchemeLDAP  = "ldap"
	SchemeLDAPI = "ldapi"
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

func setReplicationValue(entries []*ldap.Entry, q *query) {
	for _, entry := range entries {
		val := entry.GetAttributeValue(q.searchAttr)
		if val == "" {
			// not every entry will have this attribute
			continue
		}
		fields := log.Fields{
			"filter": q.searchFilter,
			"attr":   q.searchAttr,
			"value":  val,
		}
		valueBuffer := strings.Split(val, "#")
		gt, err := time.Parse("20060102150405.999999Z", valueBuffer[0])
		if err != nil {
			log.WithFields(fields).WithError(err).Warn("unexpected gt value")
			continue
		}
		count, err := strconv.ParseFloat(valueBuffer[1], 64)
		if err != nil {
			log.WithFields(fields).WithError(err).Warn("unexpected count value")
			continue
		}
		sid := valueBuffer[2]
		mod, err := strconv.ParseFloat(valueBuffer[3], 64)
		if err != nil {
			log.WithFields(fields).WithError(err).Warn("unexpected mod value")
			continue
		}
		q.metric.WithLabelValues(sid, "gt").Set(float64(gt.Unix()))
		q.metric.WithLabelValues(sid, "count").Set(count)
		q.metric.WithLabelValues(sid, "mod").Set(mod)
	}
}

type LDAPConfig struct {
	UseTLS      bool
	UseStartTLS bool
	Scheme      string
	Addr        string
	Host        string
	Port        string
	Protocol    string
	Username    string
	Password    string
	TLSConfig   tls.Config
}

type Scraper struct {
	LDAPConfig LDAPConfig
	Tick       time.Duration
	log        log.FieldLogger
	Sync       []string
}

func (config *LDAPConfig) ProcessTLSoptions(addr string, useStartTLS bool, skipInsecure bool) error {

	var u *url.URL

	u, err := url.Parse(addr)
	if err != nil {
		// Well, so far the easy way....
		u = &url.URL{}
	}

	if u.Host == "" {
		if strings.HasPrefix(addr, SchemeLDAPI) {
			u.Scheme = SchemeLDAPI
			u.Host, _ = url.QueryUnescape(strings.Replace(addr, SchemeLDAPI+"://", "", 1))
		} else if strings.HasPrefix(addr, SchemeLDAPS) {
			u.Scheme = SchemeLDAPS
			u.Host = strings.Replace(addr, SchemeLDAPS+"://", "", 1)
		} else {
			u.Scheme = SchemeLDAP
			u.Host = strings.Replace(addr, SchemeLDAP+"://", "", 1)
		}
	}

	config.Addr = u.Host
	config.Scheme = u.Scheme
	config.Host = u.Hostname()

	r, _ := regexp.Compile(":[0-9]+")
	if u.Scheme == SchemeLDAPS {
		config.UseTLS = true
		if !r.MatchString(config.Addr) {
			config.Port = "636"
			config.Addr += ":" + config.Port
		}
	} else if u.Scheme == SchemeLDAP {
		config.UseTLS = false
		if !r.MatchString(config.Addr) {
			config.Port = "389"
			config.Addr += ":" + config.Port
		}
	} else if u.Scheme == SchemeLDAPI {
		config.Protocol = "unix"
	} else {
		return errors.New(u.Scheme + " is not a scheme i understand, refusing to continue")
	}

	config.TLSConfig.InsecureSkipVerify = skipInsecure
	config.TLSConfig.ServerName = config.Host
	if !config.UseTLS {
		// useStartTLS only relevant if not using TLS
		config.UseStartTLS = useStartTLS
	}

	return nil
}

func (config *LDAPConfig) LoadCACert(cafile string) error {

	if _, err := os.Stat(cafile); os.IsNotExist(err) {
		return errors.New("CA Certificate file does not exists")
	}

	cert, err := ioutil.ReadFile(cafile)

	if err != nil {
		return errors.New("CA Certificate file is not readable")
	}

	config.TLSConfig.RootCAs = x509.NewCertPool()

	if !config.TLSConfig.RootCAs.AppendCertsFromPEM(cert) {
		return errors.New("Could not parse CA")
	}

	return nil
}

func NewLDAPConfig() LDAPConfig {

	conf := LDAPConfig{}

	conf.Scheme = SchemeLDAP
	conf.Host = "localhost"
	conf.Port = "389"
	conf.Addr = conf.Host + ":" + conf.Port
	conf.Protocol = "tcp"
	conf.UseTLS = false
	conf.UseStartTLS = false
	conf.TLSConfig = tls.Config{}

	return conf
}

func (s *Scraper) addReplicationQueries() {
	for _, q := range s.Sync {
		queries = append(queries,
			&query{
				baseDN:       q,
				searchFilter: objectClass("*"),
				searchAttr:   monitorReplicationFilter,
				metric:       monitorReplicationGauge,
				setData:      setReplicationValue,
			},
		)
	}
}

func (s *Scraper) Start(ctx context.Context) {
	s.log = log.WithField("component", "scraper")
	s.addReplicationQueries()
	security := "None"
	if s.LDAPConfig.UseTLS {
		security = "TLS"
	} else if s.LDAPConfig.UseStartTLS {
		security = "StartTLS"
	}
	if s.LDAPConfig.TLSConfig.InsecureSkipVerify {
		security += "/InsecureSkipVerify"
	}
	address := s.LDAPConfig.Scheme + "://" + s.LDAPConfig.Addr
	s.log.WithField("addr", address).WithField("security", security).Info("starting monitor loop")
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

func (s *Scraper) scrape() {
	var conn *ldap.Conn
	var err error

	if s.LDAPConfig.UseTLS {
		conn, err = ldap.DialTLS(s.LDAPConfig.Protocol, s.LDAPConfig.Addr, &s.LDAPConfig.TLSConfig)
	} else {
		conn, err = ldap.Dial(s.LDAPConfig.Protocol, s.LDAPConfig.Addr)
		if err != nil {
			s.log.WithError(err).Error("dial failed")
			dialCounter.WithLabelValues("fail").Inc()
			return
		}

		if s.LDAPConfig.UseStartTLS {
			err = conn.StartTLS(&s.LDAPConfig.TLSConfig)
			if err != nil {
				s.log.WithError(err).Error("StartTLS failed")
				dialCounter.WithLabelValues("fail").Inc()
				return
			}
		}
	}

	if err != nil {
		s.log.WithError(err).Error("dial failed")
		dialCounter.WithLabelValues("fail").Inc()
		return
	}
	dialCounter.WithLabelValues("ok").Inc()
	defer conn.Close()

	if s.LDAPConfig.Username != "" && s.LDAPConfig.Password != "" {
		err = conn.Bind(s.LDAPConfig.Username, s.LDAPConfig.Password)
		if err != nil {
			s.log.WithError(err).Error("bind failed")
			bindCounter.WithLabelValues("fail").Inc()
			return
		}
		bindCounter.WithLabelValues("ok").Inc()
	}

	scrapeRes := "ok"
	for _, q := range queries {
		if err = scrapeQuery(conn, q); err != nil {
			s.log.WithError(err).WithField("filter", q.searchFilter).Warn("query failed")
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
