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
	promAddr = flag.String("promAddr", ":9431", "Bind address for prometheus HTTP metrics server")
	ldapAddr = flag.String("ldapAddr", "localhost:389", "Address of OpenLDAP server")
	ldapUser = flag.String("ldapUser", "", "OpenLDAP bind username (optional)")
	ldapPass = flag.String("ldapPass", "", "OpenLDAP bind password (optional)")
	interval = flag.Duration("interval", 30*time.Second, "Scrape interval")
	version  = flag.Bool("version", false, "Show version and exit")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(exporter.GetVersion())
		os.Exit(0)
	}

	log.Println("Starting prometheus HTTP metrics server on", *promAddr)
	go exporter.StartMetricsServer(*promAddr)

	log.Println("Starting OpenLDAP scraper for", *ldapAddr)
	for range time.Tick(*interval) {
		exporter.ScrapeMetrics(*ldapAddr, *ldapUser, *ldapPass)
	}
}
