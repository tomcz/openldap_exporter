package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	exporter "github.com/tomcz/openldap_exporter"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"golang.org/x/sync/errgroup"
)

const (
	promAddr   = "promAddr"
	ldapNet    = "ldapNet"
	ldapAddr   = "ldapAddr"
	ldapUser   = "ldapUser"
	ldapPass   = "ldapPass"
	interval   = "interval"
	metrics    = "metrPath"
	webCfgFile = "webCfgFile"
	config     = "config"
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
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    webCfgFile,
			Usage:   "Prometheus metrics web config `FILE` (optional)",
			EnvVars: []string{"WEB_CFG_FILE"},
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
	server := exporter.NewMetricsServer(c.String(promAddr), c.String(metrics), c.String(webCfgFile))

	scraper := &exporter.Scraper{
		Net:  c.String(ldapNet),
		Addr: c.String(ldapAddr),
		User: c.String(ldapUser),
		Pass: c.String(ldapPass),
		Tick: c.Duration(interval),
	}

	ctx, cancel := context.WithCancel(context.Background())
	var group errgroup.Group
	group.Go(func() error {
		defer cancel()
		log.Printf("starting Prometheus HTTP metrics server on %s\n", c.String(promAddr))
		return server.Start()
	})
	group.Go(func() error {
		defer cancel()
		log.Printf("starting OpenLDAP scraper for %s://%s\n", scraper.Net, scraper.Addr)
		return scraper.Start(ctx)
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
			log.Println("shutdown received")
			return nil
		case <-ctx.Done():
			return nil
		}
	})
	return group.Wait()
}
