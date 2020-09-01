package openldap_exporter

import (
	"fmt"
	"log"
	"errors"
	"strconv"
	"regexp"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"
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
)

type query struct {
	baseDN       string
	searchFilter string
	searchAttr   string
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
)

func init() {
	prometheus.MustRegister(
		monitoredObjectGauge,
		monitorCounterObjectGauge,
		monitorOperationGauge,
		scrapeCounter,
	)
}

func objectClass(name string) string {
	return fmt.Sprintf("(objectClass=%v)", name)
}

func ScrapeMetrics(ldapAddr, ldapUser, ldapPass string) {
	if err := scrapeAll(ldapAddr, ldapUser, ldapPass); err != nil {
		scrapeCounter.WithLabelValues("fail").Inc()
		log.Println("scrape failed, error is:", err)
	} else {
		scrapeCounter.WithLabelValues("ok").Inc()
	}
}

func scrapeAll(ldapAddr, ldapUser, ldapPass string) error {
	//l, err := ldap.Dial("tcp", ldapAddr)
	l, err := scrapeDial(ldapAddr)
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
	return errs
}

func scrapeDial(ldapAddr string) (*ldap.Conn, error) {
	re := regexp.MustCompile(`^(?:(?P<ldapScheme>ldapi|ldap|ldaps):\/\/)?(?P<ldapHost>(?P<ipv4>(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])|(?P<ipv6>\[[a-z0-9\-._~%!$&'()*+,;=:]+\])|(?P<fqdn>[a-zA-Z0-9\-._~%]+))(?::(?P<ldapPort>\d+))?\/?$`)
	
	if match := re.FindStringSubmatch(ldapAddr); match != nil {
		switch scheme := match[1]; scheme {
		case "ldapi":
			unixFilePath, err := url.PathUnescape(match[2])
			if err != nil {
				return nil, err
			}

			return ldap.Dial("unix", unixFilePath)

		case "ldaps":
			// port, err := strconv.Atoi(match[6])
			// if err != nil {
			// 	port = 636
			// }
			// hostPort := fmt.Sprintf("%s:%d", match[2], port)
			// TODO: implement connection over LDAPS
			// return ldap.DialTLS("tcp", hostPost, config)

			return nil, errors.New("scraper: ldaps is not yet supported.")

		default:
			port, err := strconv.Atoi(match[6])
			if err != nil {
				port = 389
			}
			hostPort := fmt.Sprintf("%s:%d", match[2], port)

			return ldap.Dial("tcp", hostPort)

		}
	} else {
		err := errors.New("scraper: Cannot parse ldap address.")
		return nil, err
	}
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
