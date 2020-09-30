package main

import (
	"log"
	"os"
	"time"

	exporter "github.com/tomcz/openldap_exporter"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const (
	promAddr = "promAddr"
	ldapAddr = "ldapAddr"
	ldapUser = "ldapUser"
	ldapPass = "ldapPass"
    bindLocalUser = "bindLocalUser"
    bindLocalPass = "bindLocalPass"
    bindTestUser = "bindTestUser"
    bindTestPass = "bindTestPass"
    bindTestAddr = "bindTestAddr"
    searchTestFilter = "searchTestFilter"
    searchBaseDN = "searchBaseDN"
	interval = "interval"
	metrics  = "metrPath"
	config   = "config"
    saslAuthd = "saslAuthd"
    bindTestUserAd = "bindTestUserAd"
)

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
			Name:    ldapAddr,
			Value:   "localhost:389",
			Usage:   "Address of OpenLDAP server",
			EnvVars: []string{"LDAP_ADDR"},
		}),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindTestAddr,
            Value:   "localhost:389",
            Usage:   "Address of OpenLDAP Test Bind server",
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
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindLocalUser,
            Usage:   "OpenLDAP local bind username test (optional)",
            EnvVars: []string{"BIND_LOCAL_USER"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindLocalPass,
            Usage:   "OpenLDAP local bind pass test (optional)",
            EnvVars: []string{"BIND_LOCAL_PASS"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindTestUser,
            Usage:   "OpenLDAP bind test username (optional)",
            EnvVars: []string{"BIND_TEST_USER"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindTestPass,
            Usage:   "OpenLDAP bind test password (optional)",
            EnvVars: []string{"BIND_TEST_PASS"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    saslAuthd,
            Usage:   "SASLAUTHD username (optional)",
            EnvVars: []string{"SASLAUTHD_USER"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    bindTestUserAd,
            Usage:   "OpenLDAP bind test username (optional)",
            EnvVars: []string{"BIND_TEST_USER_AD"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    searchTestFilter,
            Usage:   "Search Filter (optional)",
            EnvVars: []string{"SEARCH_FILTER"},
        }),
        altsrc.NewStringFlag(&cli.StringFlag{
            Name:    searchBaseDN,
            Usage:   "Search BaseDN (optional)",
            EnvVars: []string{"SEARCH_BASEDN"},
        }),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:    interval,
			Value:   30 * time.Second,
			Usage:   "Scrape interval",
			EnvVars: []string{"INTERVAL"},
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
		Version:         exporter.GetVersion(),
		HideHelpCommand: true,
		Flags:           flags,
		Action:          runMain,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
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
	log.Println("starting Prometheus HTTP metrics server on", c.String(promAddr))
	go exporter.StartMetricsServer(c.String(promAddr), c.String(metrics))

	log.Println("starting OpenLDAP scraper for", c.String(ldapAddr))
	for range time.Tick(c.Duration(interval)) {
		exporter.ScrapeMetrics(c.String(ldapAddr), c.String(ldapUser), c.String(ldapPass),c.String(bindTestUser),c.String(bindTestPass),c.String(bindTestAddr),c.String(saslAuthd),c.String(bindTestUserAd),c.String(bindLocalUser),c.String(bindLocalPass),c.String(searchTestFilter),c.String(searchBaseDN))
	}
	return nil
}
