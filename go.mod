module github.com/tomcz/openldap_exporter

go 1.16

require (
	github.com/go-kit/kit v0.10.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/exporter-toolkit v0.5.1
	github.com/sirupsen/logrus v1.6.0
	github.com/urfave/cli/v2 v2.2.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/ldap.v2 v2.5.1
)
