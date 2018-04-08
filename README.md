[![Build Status](https://travis-ci.org/tomcz/openldap_exporter.svg?branch=master)](https://travis-ci.org/tomcz/openldap_exporter)

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

Once you've built the exporter (see below) you can install it on the same server as your _slapd_ instance, and run it as a service. You can then configure Prometheus to pull metrics from the exporter's `/metrics` endpoint on port 9330, and check to see that it is working via curl:

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

## Building the exporter

1. Install Go from https://golang.org/
2. Install `dep` from https://github.com/golang/dep
3. Clone this repository and `cd` into its root.
4. Fetch the third-party libraries: `make deps`
5. Build the binary: `make build`

## Command line configuration

The binary itself is configured via command line flags:

```
Usage of ./target/openldap_exporter:
  -interval duration
        Scrape interval (default 30s)
  -ldapAddr string
        Address of OpenLDAP server (default "localhost:389")
  -ldapPass string
        OpenLDAP bind password (optional)
  -ldapUser string
        OpenLDAP bind username (optional)
  -promAddr string
        Bind address for prometheus HTTP metrics server (default ":9330")
  -version
        Show version and exit
```
