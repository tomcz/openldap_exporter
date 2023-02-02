package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	exporter "github.com/tomcz/openldap_exporter"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"golang.org/x/sync/errgroup"
)

const (
	promAddr                = "promAddr"
	ldapNet                 = "ldapNet"
	ldapAddr                = "ldapAddr"
	ldapUser                = "ldapUser"
	ldapPass                = "ldapPass"
	interval                = "interval"
	metrics                 = "metrPath"
	jsonLog                 = "jsonLog"
	webCfgFile              = "webCfgFile"
	config                  = "config"
	replicationObject       = "replicationObject"
	replicationSearchFilter = "replicationSearchFilter"
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
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    webCfgFile,
			Usage:   "Prometheus metrics web config `FILE` (optional)",
			EnvVars: []string{"WEB_CFG_FILE"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:    jsonLog,
			Value:   false,
			Usage:   "Output logs in JSON format",
			EnvVars: []string{"JSON_LOG"},
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:  replicationObject,
			Usage: "Object to watch replication upon",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  replicationSearchFilter,
			Usage: "Search Filter to watch replication upon, takes precedence over replicationObject",
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
		log.WithError(err).Fatal("service failed")
	}
	log.Info("service stopped")
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
	if c.Bool(jsonLog) {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}
	log.Info("service starting")

	server := exporter.NewMetricsServer(
		c.String(promAddr),
		c.String(metrics),
		c.String(webCfgFile),
	)

	scraper := &exporter.Scraper{
		Net:              c.String(ldapNet),
		Addr:             c.String(ldapAddr),
		User:             c.String(ldapUser),
		Pass:             c.String(ldapPass),
		Tick:             c.Duration(interval),
		Sync:             c.StringSlice(replicationObject),
		SyncSearchFilter: c.String(replicationSearchFilter),
	}

	ctx, cancel := context.WithCancel(context.Background())
	var group errgroup.Group
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
