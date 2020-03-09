package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	exporter "github.com/tomcz/openldap_exporter"
)

var (
	promAddr = flag.String("promAddr", defaultEnvString("PROM_ADDR", ":9330"), "Bind address for prometheus HTTP metrics server")
	ldapAddr = flag.String("ldapAddr", defaultEnvString("LDAP_ADDR", "localhost:389"), "Address of OpenLDAP server")
	ldapUser = flag.String("ldapUser", defaultEnvString("LDAP_USER", ""), "OpenLDAP bind username (optional)")
	ldapPass = flag.String("ldapPass", defaultEnvString("LDAP_PASS", ""), "OpenLDAP bind password (optional)")
	interval = flag.Duration("interval", defaultEnvDuration("INTERVAL", 30*time.Second), "Scrape interval")
	version  = flag.Bool("version", false, "Show version and exit")
)

func defaultEnvString(envName, defValue string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return defValue
	}
	return envValue
}

func defaultEnvDuration(envName string, defValue time.Duration) time.Duration {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return defValue
	}
	parsedEnv, err := time.ParseDuration(envValue)
	if err != nil {
		log.Printf("Error parseing %s, invalid value: %s\n", envName, envValue)
	}
	return parsedEnv
}

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
