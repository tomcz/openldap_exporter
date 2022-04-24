GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_TAG := $(shell git describe --tags 2>/dev/null)

LDFLAGS := -s -w -X github.com/tomcz/openldap_exporter.commit=${GITCOMMIT}
LDFLAGS := ${LDFLAGS} -X github.com/tomcz/openldap_exporter.tag=${GIT_TAG}

.PHONY: precommit
precommit: clean format lint build

.PHONY: commit
commit: clean build

.PHONY: clean
clean:
	rm -rf target

target:
	mkdir target

.PHONY: format
format:
ifeq (, $(shell which goimports))
	go install golang.org/x/tools/cmd/goimports@latest
endif
	@echo "Running goimports ..."
	@goimports -w -local github.com/tomcz/openldap_exporter $(shell find . -type f -name '*.go' | grep -v '/vendor/')

.PHONY: lint
lint:
ifeq (, $(shell which staticcheck))
	go install honnef.co/go/tools/cmd/staticcheck@latest
endif
	@echo "Running staticcheck ..."
	@staticcheck $(shell go list ./... | grep -v /vendor/)

compile = GOOS=$1 GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o target/openldap_exporter-$1 ./cmd/openldap_exporter

.PHONY: build
build: target
	$(call compile,linux)
	$(call compile,darwin)
