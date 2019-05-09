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
	promAddr 		= flag.String("promAddr", ":9330", "Bind address for prometheus HTTP metrics server")
	promCrt 		= flag.String("promCrt", "", "Path to PEM Certificate (chain) file to run metrics server in https mode (optional, required if CrtKey is used)")
	promCrtKey 		= flag.String("promCrtKey", "", "Path to PEM Certificate Key file to run metrics server in https mode (optional, required if Crt is used)")
	ldapAddr 		= flag.String("ldapAddr", "ldap://localhost:389", "Address of OpenLDAP server")
	ldapCACrt 		= flag.String("ldapCACrt", "", "Path to CA certificate for LDAPS (optional)")
	ldapUser		= flag.String("ldapUser", "", "OpenLDAP bind username (optional)")
	ldapPass 		= flag.String("ldapPass", "", "OpenLDAP bind password (optional)")
	ldapUseStartTLS = flag.Bool("ldapStartTLS", false, "Use start TLS (optional)")
	interval 		= flag.Duration("interval", 30*time.Second, "Scrape interval")
	version  		= flag.Bool("version", false, "Show version and exit")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println("Version:", exporter.GetVersion())
		os.Exit(0)
	}


	config := exporter.NewLDAPConfig()

	/** Parse ldap address **/
	err := config.ParseAddr(*ldapAddr)
	if err != nil {
		log.Println("Error parsing ldap address: ", err.Error())
		os.Exit(1)
	}

	/** Load Certificate if given, and panic on error **/
	if *ldapCACrt != "" {
		err = config.LoadCACert(*ldapCACrt)
		if err != nil {
			log.Println("Error loading CA certificate file: ", err.Error())
			os.Exit(1)
		} else {
			log.Println("Successfully loaded CA cert file:", *ldapCACrt)
		}
	}

	config.Username = *ldapUser
	config.Password = *ldapPass

	if *ldapUseStartTLS {
		config.UseStartTLS = true
	}

	serverConfig := exporter.NewServerConfig()
	serverConfig.Address = *promAddr
	serverConfig.CertFile = *promCrt
	serverConfig.KeyFile = *promCrtKey


	log.Println("Starting prometheus HTTP(s) metrics server on", serverConfig.Address)
	go exporter.StartMetricsServer(serverConfig)

	log.Println("Starting OpenLDAP scraper for", *ldapAddr)
	for range time.Tick(*interval) {
		exporter.ScrapeMetrics(&config)
	}
}
