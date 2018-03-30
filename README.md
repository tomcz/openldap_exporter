# OpenLDAP Prometheus Exporter

This is a simple service that scrapes metrics from OpenLDAP and exports them via HTTP for Prometheus consumption.

This exporter is based on the ideas in https://github.com/jcollie/openldap_exporter, but it is written in golang to allow for simpler distribution and installation.

## Setting up OpenLDAP for monitoring

_slapd_ supports an optional LDAP monitoring interface you can use to obtain information regarding the current state of your _slapd_ instance. Documentation for this backend can be found in the OpenLDAP [backend guide](http://www.openldap.org/doc/admin24/backends.html#Monitor) and [administration guide](http://www.openldap.org/doc/admin24/monitoringslapd.html).

To enable the backend add the following to the bottom of your slapd.conf file:

```
database monitor
rootdn "cn=monitoring,cn=Monitor"
rootpw YOUR_MONITORING_ROOT_PASSWORD
```

Technically you don't need `rootdn` or `rootpw`, but having unauthenticated access to _slapd_ feels a little wrong.

You may need to also load the monitoring backend module if your _slapd_ installation needs to load backends as modules:

```
moduleload  back_monitor
```
