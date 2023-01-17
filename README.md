# OpenLDAP Prometheus Exporter

This is a simple service that scrapes metrics from OpenLDAP and exports them via HTTP for Prometheus consumption.

This exporter is based on the ideas in https://github.com/jcollie/openldap_exporter, but it is written in golang to allow for simpler distribution and installation.

## Setting up OpenLDAP for monitoring

_slapd_ supports an optional LDAP monitoring interface you can use to obtain information regarding the current state of your _slapd_ instance. Documentation for this backend can be found in the OpenLDAP [backend guide](http://www.openldap.org/doc/admin24/backends.html#Monitor) and [administration guide](http://www.openldap.org/doc/admin24/monitoringslapd.html).

To enable the backend add the following to the bottom of your `slapd.conf` file:

```
database monitor
rootdn "cn=monitoring,cn=Monitor"
rootpw YOUR_MONITORING_ROOT_PASSWORD
```

Technically you don't need `rootdn` or `rootpw`, but having unauthenticated access to _slapd_ feels a little wrong.

You may need to also load the monitoring backend module if your _slapd_ installation needs to load backends as modules by adding this to your `slapd.conf`:

```
moduleload  back_monitor
```

Once you've built the exporter (see below), or downloaded the [latest release](https://github.com/mlorenzo-stratio/openldap_exporter/releases), you can install it on the same server as your _slapd_ instance, and run it as a service. You can then configure Prometheus to pull metrics from the exporter's `/metrics` endpoint on port 9330, and check to see that it is working via curl:

```
$> curl -s http://localhost:9330/metrics
...
# HELP openldap_monitor_counter_object cn=Monitor (objectClass=monitorCounterObject) monitorCounter
# TYPE openldap_monitor_counter_object gauge
openldap_monitor_counter_object{dn="cn=Bytes,cn=Statistics,cn=Monitor"} 1.857812777e+09
openldap_monitor_counter_object{dn="cn=Current,cn=Connections,cn=Monitor"} 50
openldap_monitor_counter_object{dn="cn=Entries,cn=Statistics,cn=Monitor"} 4.226632e+06
openldap_monitor_counter_object{dn="cn=Max File Descriptors,cn=Connections,cn=Monitor"} 1024
openldap_monitor_counter_object{dn="cn=PDU,cn=Statistics,cn=Monitor"} 4.446117e+06
openldap_monitor_counter_object{dn="cn=Read,cn=Waiters,cn=Monitor"} 31
openldap_monitor_counter_object{dn="cn=Referrals,cn=Statistics,cn=Monitor"} 0
openldap_monitor_counter_object{dn="cn=Total,cn=Connections,cn=Monitor"} 65383
openldap_monitor_counter_object{dn="cn=Write,cn=Waiters,cn=Monitor"} 0
# HELP openldap_monitor_operation cn=Operations,cn=Monitor (objectClass=monitorOperation) monitorOpCompleted
# TYPE openldap_monitor_operation gauge
openldap_monitor_operation{dn="cn=Abandon,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Add,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Bind,cn=Operations,cn=Monitor"} 57698
openldap_monitor_operation{dn="cn=Compare,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Delete,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Extended,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Modify,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Modrdn,cn=Operations,cn=Monitor"} 0
openldap_monitor_operation{dn="cn=Search,cn=Operations,cn=Monitor"} 161789
openldap_monitor_operation{dn="cn=Unbind,cn=Operations,cn=Monitor"} 9336
# HELP openldap_monitored_object cn=Monitor (objectClass=monitoredObject) monitoredInfo
# TYPE openldap_monitored_object gauge
openldap_monitored_object{dn="cn=Active,cn=Threads,cn=Monitor"} 1
openldap_monitored_object{dn="cn=Backload,cn=Threads,cn=Monitor"} 1
openldap_monitored_object{dn="cn=Max Pending,cn=Threads,cn=Monitor"} 0
openldap_monitored_object{dn="cn=Max,cn=Threads,cn=Monitor"} 16
openldap_monitored_object{dn="cn=Open,cn=Threads,cn=Monitor"} 8
openldap_monitored_object{dn="cn=Pending,cn=Threads,cn=Monitor"} 0
openldap_monitored_object{dn="cn=Starting,cn=Threads,cn=Monitor"} 0
openldap_monitored_object{dn="cn=Uptime,cn=Time,cn=Monitor"} 1.225737e+06
# HELP openldap_scrape successful vs unsuccessful ldap scrape attempts
# TYPE openldap_scrape counter
openldap_scrape{result="ok"} 6985
...
```

## Configuration

You can configure `openldap_exporter` using multiple configuration sources at the same time. All configuration sources are optional, if none are provided then the default values will be used.

The precedence of these configuration sources is as follows (from the highest to the lowest):

1. Command line flags
2. Environment variables
3. YAML configuration file parameters
4. Default values

```
NAME:
   openldap_exporter - Export OpenLDAP metrics to Prometheus

USAGE:
   openldap_exporter [global options] [arguments...]

VERSION:
   v2.2.0

GLOBAL OPTIONS:
   --promAddr value           Bind address for Prometheus HTTP metrics server (default: ":9330") [$PROM_ADDR]
   --metrPath value           Path on which to expose Prometheus metrics (default: "/metrics") [$METRICS_PATH]
   --ldapAddr value           Address of OpenLDAP server (default "ldap://localhost") [$LDAP_ADDR]
   --ldapUser value           OpenLDAP bind username (optional) [$LDAP_USER]
   --ldapPass value           OpenLDAP bind password (optional) [$LDAP_PASS]
   --ldapSkipInsecure         OpenLDAP Skip TLS verify (default: false) [$LDAP_SKIP_TLS_VERIFY]
   --ldapUseStartTLS          Use start TLS (optional)
   --ldapCACrt string         Path to CA certificate for LDAPS (optional)
   --interval value           Scrape interval (default: 30s) [$INTERVAL]
   --webCfgFile FILE          Prometheus metrics web config FILE (optional) [$WEB_CFG_FILE]
   --jsonLog                  Output logs in JSON format (default: false) [$JSON_LOG]
   --replicationObject value  Object to watch replication upon
   --config YAML_FILE         Optional configuration from a YAML_FILE
   --help, -h                 show help (default: false)
   --version, -v              print the version (default: false)
```

Example:

```
INTERVAL=10s /usr/sbin/openldap_exporter --promAddr ":8080" --config /etc/slapd/exporter.yaml
```

Where `exporter.yaml` looks like this:

```yaml
---
ldapUser: "cn=monitoring,cn=Monitor"
ldapPass: "sekret"
```

NOTES:

* `webCfgFile` can be used to provide authentication and TLS configuration for the [prometheus web exporter](https://github.com/prometheus/exporter-toolkit/tree/master/web).
* `ldapAddr` supports `ldaps://` (default port is `636`), `ldap://` (default port is `389`) and `ldapi://` scheme uri's. (defaults to ldap:// scheme)
using the LDAPS scheme will open a connection using TLS. Examples:
   - `ldapi:///var/run/ldapi`
   - `ldaps://ldap.host.net:666`
   - `ldap://ldap.host.net`
* Use `ldapUseStartTLS` to use StartTLS for ldap:// scheme.
* Use `ldapSkipInsecure` to skip TLS verify.
* `ldapCACrt` if the ldap server uses a custom CA certificate, add the path to the public CA Cert in PEM format

## Build

1. Install Go 1.18 from https://golang.org/
2. Build the binaries: `make build`
