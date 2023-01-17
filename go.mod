module github.com/mlorenzo-stratio/openldap_exporter

go 1.18

require (
	github.com/go-kit/log v0.2.0
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/exporter-toolkit v0.7.3
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli/v2 v2.4.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/ldap.v2 v2.5.1
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	golang.org/x/crypto v0.0.0-20210915214749-c084706c2272 // indirect
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/asn1-ber.v1 v1.5.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace gopkg.in/asn1-ber.v1 => github.com/go-asn1-ber/asn1-ber v1.5.4
