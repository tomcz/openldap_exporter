package main

import (
	"flag"
	"log"
	"time"

	"app/exporter"
)

var (
	promAddr = flag.String("promAddr", ":8001", "bind address for prometheus listener")
	ldapAddr = flag.String("ldapAddr", "localhost:389", "address of OpenLDAP server")
	ldapUser = flag.String("ldapUser", "", "LDAP bind username (optional)")
	ldapPass = flag.String("ldapPass", "", "LDAP bind password (optional)")
	interval = flag.Duration("interval", 30*time.Second, "scrape interval")
)

func main() {
	flag.Parse()

	log.Println("Starting HTTP metrics server on", *promAddr)
	go exporter.StartMetricsServer(*promAddr)

	log.Println("Starting OpenLDAP scraper for", *ldapAddr)
	for range time.Tick(*interval) {
		exporter.ScrapeMetrics(*ldapAddr, *ldapUser, *ldapPass)
	}
}
