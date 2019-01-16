package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"app/exporter"
)

var (
	promAddr         = flag.String("promAddr", ":9330", "Bind address for prometheus HTTP metrics server")
	ldapUri          = flag.String("ldapUri", "ldap://localhost:389", "Uri of OpenLDAP server")
	ldapUser         = flag.String("ldapUser", "", "OpenLDAP bind username (optional)")
	ldapPass         = flag.String("ldapPass", "", "OpenLDAP bind password (optional)")
	ldapSkipInsecure = flag.Bool("ldapSkipInsecure", false, "OpenLDAP Skip TLS verify (default=false)")
	interval         = flag.Duration("interval", 30*time.Second, "Scrape interval")
	version          = flag.Bool("version", false, "Show version and exit")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(exporter.GetVersion())
		os.Exit(0)
	}

	log.Println("Starting prometheus HTTP metrics server on", *promAddr)
	go exporter.StartMetricsServer(*promAddr)

	log.Println("Starting OpenLDAP scraper for", *ldapUri)
	for range time.Tick(*interval) {
		exporter.ScrapeMetrics(*ldapUri, *ldapUser, *ldapPass, *ldapSkipInsecure)
	}
}
