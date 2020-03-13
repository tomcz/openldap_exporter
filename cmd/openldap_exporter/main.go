package main

import (
	"fmt"
	"log"
	"os"
	"time"

	exporter "github.com/tomcz/openldap_exporter"

	"github.com/urfave/cli/v2"
)

const (
	promAddr = "promAddr"
	ldapAddr = "ldapAddr"
	ldapUser = "ldapUser"
	ldapPass = "ldapPass"
	interval = "interval"
)

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    promAddr,
			Value:   ":9330",
			Usage:   "Bind address for Prometheus HTTP metrics server",
			EnvVars: []string{"PROM_ADDR"},
		},
		&cli.StringFlag{
			Name:    ldapAddr,
			Value:   "localhost:389",
			Usage:   "Address of OpenLDAP server",
			EnvVars: []string{"LDAP_ADDR"},
		},
		&cli.StringFlag{
			Name:    ldapUser,
			Usage:   "OpenLDAP bind username (optional)",
			EnvVars: []string{"LDAP_USER"},
		},
		&cli.StringFlag{
			Name:    ldapPass,
			Usage:   "OpenLDAP bind password (optional)",
			EnvVars: []string{"LDAP_PASS"},
		},
		&cli.DurationFlag{
			Name:    interval,
			Value:   30 * time.Second,
			Usage:   "Scrape interval",
			EnvVars: []string{"INTERVAL"},
		},
	}
	app := &cli.App{
		Name:   "openldap_exporter",
		Usage:  "Export OpenLDAP metrics to Prometheus",
		Flags:  flags,
		Action: runMain,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Show the version and exit",
				Action: func(*cli.Context) error {
					fmt.Println(exporter.GetVersion())
					return nil
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func runMain(c *cli.Context) error {
	log.Println("starting Prometheus HTTP metrics server on", c.String(promAddr))
	go exporter.StartMetricsServer(c.String(promAddr))

	log.Println("starting OpenLDAP scraper for", c.String(ldapAddr))
	for range time.Tick(c.Duration(interval)) {
		exporter.ScrapeMetrics(c.String(ldapAddr), c.String(ldapUser), c.String(ldapPass))
	}
	return nil
}
