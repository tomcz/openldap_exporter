package openldap_exporter

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/ldap.v2"
	"os/exec"
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

	operationBind   = "OperationBind"
	operationSearch = "OperationSearch"
    operationBindTestLocal = "OperationBindTestLocal"
    operationSASLAUTHD = "OperationSASLAUTHD"
    operationBindTestForeign = "OperationBindTestForeign"

)

type query struct {
	baseDN       string
	searchFilter string
	searchAttr   string
	metric       *prometheus.GaugeVec
}

type performance struct {
	baseDN       string
	searchAttr   string
	searchFilter string
	operation    string
	metric       *prometheus.GaugeVec
}

var (
	monitoredObjectGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitored_object",
			Help:      baseDN + " " + objectClass(monitoredObject) + " " + monitoredInfo,
		},
		[]string{"dn"},
	)
	monitorCounterObjectGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitor_counter_object",
			Help:      baseDN + " " + objectClass(monitorCounterObject) + " " + monitorCounter,
		},
		[]string{"dn"},
	)
	monitorOperationGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "monitor_operation",
			Help:      opsBaseDN + " " + objectClass(monitorOperation) + " " + monitorOpCompleted,
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
	monitorPerformanceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "openldap",
			Name:      "performance",
			Help:      "Bind, Search ldap performance",
		},
		[]string{"Performance"},
	)
	queries = []query{
		{
			baseDN:       baseDN,
			searchFilter: objectClass(monitoredObject),
			searchAttr:   monitoredInfo,
			metric:       monitoredObjectGauge,
		},
		{
			baseDN:       baseDN,
			searchFilter: objectClass(monitorCounterObject),
			searchAttr:   monitorCounter,
			metric:       monitorCounterObjectGauge,
		},
		{
			baseDN:       opsBaseDN,
			searchFilter: objectClass(monitorOperation),
			searchAttr:   monitorOpCompleted,
			metric:       monitorOperationGauge,
		},
	}

	performances = []performance{
		{
			operation: operationBind,
			metric:    monitorPerformanceGauge,
		},
		{
			baseDN:       baseDN,
			searchFilter: objectClass(monitoredObject),
			searchAttr:   monitoredInfo,
			operation:    operationSearch,
			metric:       monitorPerformanceGauge,
		},
        {
            operation: operationBindTestLocal,
            metric:    monitorPerformanceGauge,
        },
        {
		    operation: operationSASLAUTHD,
		    metric: monitorPerformanceGauge,
        },
        {
		    operation: operationBindTestForeign,
		    metric: monitorPerformanceGauge,
        },
	}
)

func init() {
	prometheus.MustRegister(
		monitoredObjectGauge,
		monitorCounterObjectGauge,
		monitorOperationGauge,
		monitorPerformanceGauge,
		scrapeCounter,
	)
}

func objectClass(name string) string {
	return fmt.Sprintf("(objectClass=%v)", name)
}

func ScrapeMetrics(ldapAddr, ldapUser, ldapPass,bindTestUser,bindTestPass,bindTestAddr, saslAuthd,bindTestUserAd,bindLocalUser,bindLocalPass,searchTestFilter,searchBaseDN string) {
	if err := scrapeAll(ldapAddr, ldapUser, ldapPass,bindTestUser,bindTestPass,bindTestAddr,saslAuthd,bindTestUserAd,bindLocalUser,bindLocalPass,searchTestFilter,searchBaseDN); err != nil {
		scrapeCounter.WithLabelValues("fail").Inc()
		log.Println("scrape failed, error is:", err)
	} else {
		scrapeCounter.WithLabelValues("ok").Inc()
	}
}

func scrapeAll(ldapAddr, ldapUser, ldapPass,bindTestUser,bindTestPass,bindTestAddr,saslAuthd,bindTestUserAd,bindLocalUser,bindLocalPass,searchTestFilter,searchBaseDN string) error {
	l, err := ldap.Dial("tcp", ldapAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	if ldapUser != "" && ldapPass != "" {
		err = l.Bind(ldapUser, ldapPass)
		if err != nil {
			return err
		}
	}

	var errs error
	for _, q := range queries {
		if err := scrapeQuery(l, &q); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	for _, p := range performances {
		if err := scrapePerformance(l, &p,bindTestUser,bindTestPass,bindTestAddr,saslAuthd,bindTestUserAd,bindLocalUser,bindLocalPass,searchTestFilter,searchBaseDN); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}
func scrapePerformance(l *ldap.Conn, p *performance,bindTestUser,bindTestPass,bindTestAddr,saslAuthd,bindTestUserAd,bindLocalUser,bindLocalPass,searchTestFilter,searchBaseDN string) error {
	if p.operation == "OperationBind" {
		bindBefore := time.Now().UnixNano()
		err := l.Bind(bindLocalUser, bindLocalPass)
		if err != nil {
			p.metric.WithLabelValues(p.operation).Set(-1)
			return err
		} else {
			bindAfter := time.Now().UnixNano()
			diff := float64(bindAfter-bindBefore) / 1000000
			p.metric.WithLabelValues(p.operation).Set(diff)
		}
	}
	if p.operation == "OperationSearch" {
        filter := fmt.Sprintf("(%s)", ldap.EscapeFilter(searchTestFilter))
		req := ldap.NewSearchRequest(
			searchBaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			filter,
			[]string{"dn", "cn"},
            nil,
		)
		searchBefore := time.Now().UnixNano()
		_, err := l.Search(req)
		if err != nil {
		    fmt.Println(err)
			p.metric.WithLabelValues(p.operation).Set(-1)
		} else {
			searchAfter := time.Now().UnixNano()
			diff := float64(searchAfter-searchBefore) / 1000000
			p.metric.WithLabelValues(p.operation).Set(diff)
		}
	}
    if p.operation == "OperationBindTestLocal" {
        bindBefore := time.Now().UnixNano()
        err := l.Bind(bindTestUser, bindTestPass)
        if err != nil {
            p.metric.WithLabelValues(p.operation).Set(-1)
            return err
        } else {
            bindAfter := time.Now().UnixNano()
            diff := float64(bindAfter-bindBefore) / 1000000
            p.metric.WithLabelValues(p.operation).Set(diff)
        }
    }
    if p.operation == "OperationSASLAUTHD" {
        saslBefore :=  time.Now().UnixNano()
        cmd := exec.Command("testsaslauthd","-u",saslAuthd,"-p",bindTestPass)
        err := cmd.Run()
        if err != nil {
            p.metric.WithLabelValues(p.operation).Set(-1)
            return err
        } else {
            saslAfter := time.Now().UnixNano()
            diff := float64(saslAfter-saslBefore) / 1000000
            p.metric.WithLabelValues(p.operation).Set(diff)
        }
    }
    if p.operation =="OperationBindTestForeign" {
        d, err := ldap.Dial("tcp", bindTestAddr)
        if err != nil {
            return err
        }
        defer d.Close()
        bindForeignBefore := time.Now().UnixNano()
        err = d.Bind(bindTestUserAd, bindTestPass)
        if err != nil {
            p.metric.WithLabelValues(p.operation).Set(-1)
            return err
        } else {
            bindForeignAfter := time.Now().UnixNano()
            diff := float64(bindForeignAfter-bindForeignBefore) / 1000000
            p.metric.WithLabelValues(p.operation).Set(diff)
        }

    }
	return nil
}
func scrapeQuery(l *ldap.Conn, q *query) error {
	req := ldap.NewSearchRequest(
		q.baseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		q.searchFilter, []string{"dn", q.searchAttr}, nil,
	)
	sr, err := l.Search(req)
	if err != nil {
		return err
	}
	for _, entry := range sr.Entries {
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
