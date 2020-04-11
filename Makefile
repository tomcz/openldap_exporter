GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_TAG := $(shell git describe --tags 2>/dev/null)

LDFLAGS := -X github.com/tomcz/openldap_exporter.commit=${GITCOMMIT}
LDFLAGS := ${LDFLAGS} -X github.com/tomcz/openldap_exporter.tag=${GIT_TAG}

precommit: clean format build

travis: clean
	GO111MODULE=on GOFLAGS='-mod=vendor' $(MAKE) build

clean:
	rm -rf target

target:
	mkdir target

format:
	@echo "Running goimports ..."
	@goimports -w -local github.com/tomcz/openldap_exporter $(shell find . -type f -name '*.go' | grep -v '/vendor/')

compile = GOOS=$1 GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o target/openldap_exporter-$1 ./cmd/openldap_exporter

build: target
	$(call compile,linux)
	$(call compile,darwin)
