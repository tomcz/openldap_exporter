package openldap_exporter

import (
	"context"
	"fmt"
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

	monitorReplicationFilter = "contextcsn"
	monitorReplication       = "monitorReplication"
)

type query struct {
	baseDN       string
	searchFilter string
	searchAttr   string
	metric       *prometheus.GaugeVec
	setData      func([]*ldap.Entry, *query) error
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
			setData: func(entries []*ldap.Entry, q *query) error {
				return setValue(entries, q)

			},
		}, {
			baseDN:       baseDN,
			searchFilter: objectClass(monitorCounterObject),
			searchAttr:   monitorCounter,
			metric:       monitorCounterObjectGauge,
			setData: func(entries []*ldap.Entry, q *query) error {
				return setValue(entries, q)
			},
		},
		{
			baseDN:       opsBaseDN,
			searchFilter: objectClass(monitorOperation),
			searchAttr:   monitorOpCompleted,
			metric:       monitorOperationGauge,
			setData: func(entries []*ldap.Entry, q *query) error {
				return setValue(entries, q)
			},
		},
		{
			baseDN:       opsBaseDN,
			searchFilter: objectClass(monitorOperation),
			searchAttr:   monitorOpCompleted,
			metric:       monitorOperationGauge,
			setData: func(entries []*ldap.Entry, q *query) error {
				return setValue(entries, q)
			},
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
	)
}

func help(msg ...string) string {
	return strings.Join(msg, " ")
}

func objectClass(name string) string {
	return fmt.Sprintf("(objectClass=%v)", name)
}

func setValue(entries []*ldap.Entry, q *query) error {
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
	return nil

}

type Scraper struct {
	Net      string
	Addr     string
	User     string
	Pass     string
	Tick     time.Duration
	LdapSync []string
	log      log.FieldLogger
	Sync     []string
}

func (s *Scraper) addReplicationQueries() error {
	if len(s.Sync) != 0 {
		for _, q := range s.Sync {
			queries = append(queries,
				&query{
					baseDN:       q,
					searchFilter: objectClass("*"),
					searchAttr:   "contextCSN",
					metric:       monitorReplicationGauge,
					setData: func(entries []*ldap.Entry, q *query) error {
						for _, entry := range entries {
							val := entry.GetAttributeValue(q.searchAttr)
							if val == "" {
								// not every entry will have this attribute
								continue
							}
							valueBuffer := strings.Split(val, "#")
							gt, err := time.Parse("20060102150405.999999Z", valueBuffer[0])
							if err != nil {
								return err
							}
							count, err := strconv.ParseFloat(valueBuffer[1], 64)
							if err != nil {
								return err
							}
							sid := valueBuffer[2]
							mod, err := strconv.ParseFloat(valueBuffer[3], 64)
							if err != nil {
								return err
							}
							q.metric.WithLabelValues(sid, "gt").Set(float64(gt.Unix()))
							q.metric.WithLabelValues(sid, "count").Set(count)
							q.metric.WithLabelValues(sid, "mod").Set(mod)
						}
						return nil
					},
				},
			)
		}
	}
	return nil
}

func (s *Scraper) Start(ctx context.Context) error {
	s.log = log.WithField("component", "scraper")
	s.addReplicationQueries()
	address := fmt.Sprintf("%s://%s", s.Net, s.Addr)
	s.log.WithField("addr", address).Info("starting monitor loop")
	ticker := time.NewTicker(s.Tick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.runOnce()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Scraper) runOnce() {
	result := "fail"
	if s.scrape() {
		result = "ok"
	}
	scrapeCounter.WithLabelValues(result).Inc()
}

func (s *Scraper) scrape() bool {
	conn, err := ldap.Dial(s.Net, s.Addr)
	if err != nil {
		s.log.WithError(err).Error("dial failed")
		return false
	}
	defer conn.Close()

	if s.User != "" && s.Pass != "" {
		err = conn.Bind(s.User, s.Pass)
		if err != nil {
			s.log.WithError(err).Error("bind failed")
			return false
		}
	}

	ret := true
	for _, q := range queries {
		if err := scrapeQuery(conn, q); err != nil {
			s.log.WithError(err).WithField("filter", q.searchFilter).Warn("query failed")
			ret = false
		}
	}
	return ret
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
	return q.setData(sr.Entries, q)
}
