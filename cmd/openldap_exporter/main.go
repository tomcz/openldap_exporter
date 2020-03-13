package main

import (
	"fmt"
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
	interval = "interval"
	metrics  = "metrPath"
	config   = "config"
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
		&cli.StringFlag{
			Name:  config,
			Usage: "Optional configuration from a `YAML_FILE`",
		},
	}
	commands := []*cli.Command{
		{
			Name:  "version",
			Usage: "Show the version and exit",
			Action: func(*cli.Context) error {
				fmt.Println(exporter.GetVersion())
				return nil
			},
		},
	}
	app := &cli.App{
		Name:     "openldap_exporter",
		Usage:    "Export OpenLDAP metrics to Prometheus",
		Before:   altsrc.InitInputSourceWithContext(flags, optionalYamlSourceFunc(config)),
		Flags:    flags,
		Action:   runMain,
		Commands: commands,
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
		exporter.ScrapeMetrics(c.String(ldapAddr), c.String(ldapUser), c.String(ldapPass))
	}
	return nil
}
