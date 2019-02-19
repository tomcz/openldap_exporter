package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xaque208/openldap_exporter/code/src/app/exporter"
)

var rootCmd = &cobra.Command{
	Use:   "openldap_exporter",
	Short: "Export OpenLDAP metrics to Pometheus",
	Long:  "",
	Run:   run,
}

var (
	verbose       bool
	cfgFile       string
	listenAddress string
	ldapHost      string
	ldapPort      int
	bindDN        string
	bindPW        string
	interval      int
	caFile        string
	skipVerify    bool
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.openldap_exporter.yaml)")
	rootCmd.PersistentFlags().StringVarP(&listenAddress, "listen", "L", ":9100", "The listen address (default is :9100")
	rootCmd.PersistentFlags().IntVarP(&interval, "interval", "i", 30, "The interval at which to update the data")
	rootCmd.PersistentFlags().StringVarP(&bindDN, "bindDN", "", "", "The LDAP bind DN")
	rootCmd.PersistentFlags().StringVarP(&bindPW, "bindPW", "", "", "The LDAP bind password")
	// rootCmd.PersistentFlags().StringVarP(&ldapAddr, "host", "H", "", "The LDAP host to query")
	// viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	// viper.BindPFlag("ldapAddr", rootCmd.PersistentFlags().Lookup("ldapAddr"))
	// viper.BindPFlag("bindDN", rootCmd.PersistentFlags().Lookup("bindDN"))
	// viper.BindPFlag("bindPW", rootCmd.PersistentFlags().Lookup("bindPW"))
	// viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))

}

// initConfig reads in the config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		// Search config in home directory with name ".openldap_exporter" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".openldap_exporter")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
		cfgFile = viper.ConfigFileUsed()
		log.Error(cfgFile)
	}
}

func run(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// viper.SetDefault("nats.topic", "things")

	ldapHost = viper.GetString("ldap.host")
	ldapPort = viper.GetInt("ldap.port")
	bindDN = viper.GetString("ldap.binddn")
	bindPW = viper.GetString("ldap.bindpw")
	caFile = viper.GetString("tls.ca_file")
	skipVerify = viper.GetBool("tls.skip_verify")
	interval = viper.GetInt("interval")

	ldapAddr := fmt.Sprintf("%s:%d", ldapHost, ldapPort)

	log.Infof("Starting prometheus HTTP metrics server: %s", listenAddress)
	go exporter.StartMetricsServer(listenAddress)

	// Load CA cert
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipVerify,
		ServerName:         ldapHost,
		RootCAs:            caCertPool,
	}

	log.Infof("Starting OpenLDAP scraper: %s", ldapAddr)
	log.Debugf("   ..with credentials: %s", bindDN)
	log.Debugf("   ..with tls: %+v", tlsConfig)
	for range time.Tick(time.Duration(interval) * time.Second) {
		exporter.ScrapeMetrics(ldapAddr, bindDN, bindPW, tlsConfig)
	}
}
