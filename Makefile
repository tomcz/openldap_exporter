GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS := -X github.com/tomcz/openldap_exporter.version=${GITCOMMIT}

precommit: clean format build

travis: clean
	GO111MODULE=on GOFLAGS='-mod=vendor' $(MAKE) build

clean:
	rm -rf target

target:
	mkdir target

format:
	go fmt ./...

compile = GOOS=$1 GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o target/openldap_exporter-$1 ./cmd/openldap_exporter

build: target
	$(call compile,linux)
	$(call compile,darwin)
