package exporter

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

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

	SchemeLDAPS = "ldaps"
	SchemeLDAP  = "ldap"
	SchemeLDAPI = "ldapi"
)

type query struct {
	baseDN       string
	searchFilter string
	searchAttr   string
	metric       *prometheus.GaugeVec
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

func (config *LDAPConfig) ParseAddr(addr string) error {

	var u *url.URL

	u, err := url.Parse(addr)
	if (err != nil) {
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

	if u.Scheme == SchemeLDAPS {
		config.UseTLS = true
	} else if u.Scheme == SchemeLDAP {
		config.UseTLS = false
	} else if u.Scheme == SchemeLDAPI {
		config.Protocol = "unix"
	} else {
		return errors.New(u.Scheme + " is not a scheme i understand, refusing to continue")
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
	config.TLSConfig.ServerName = config.Host

	ok := config.TLSConfig.RootCAs.AppendCertsFromPEM(cert)

	if ok == false {
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

func ScrapeMetrics(config *LDAPConfig) {
	if err := scrapeAll(config); err != nil {
		scrapeCounter.WithLabelValues("fail").Inc()
		log.Println("Scrape failed, error is:", err)
	} else {
		scrapeCounter.WithLabelValues("ok").Inc()
	}
}

func scrapeAll(config *LDAPConfig) error {

	var l *ldap.Conn
	var err error

	if config.UseTLS {
		l, err = ldap.DialTLS(config.Protocol, config.Addr, &config.TLSConfig)
	} else {
		l, err = ldap.Dial(config.Protocol, config.Addr)
		if err != nil {
			return err
		}
		if config.UseStartTLS {
			err = l.StartTLS(&config.TLSConfig)
			if err != nil {
				return err
			}
		}
	}

	if err != nil {
		return err
	}
	defer l.Close()

	if config.Username != "" && config.Password != "" {
		err = l.Bind(config.Username, config.Password)
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
