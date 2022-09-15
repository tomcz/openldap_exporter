GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_TAG := $(shell git describe --tags 2>/dev/null)

LDFLAGS := -X github.com/tomcz/openldap_exporter.commit=${GITCOMMIT}
LDFLAGS := ${LDFLAGS} -X github.com/tomcz/openldap_exporter.tag=${GIT_TAG}

.PHONY: precommit
precommit: clean format lint build

.PHONY: commit
commit: clean
	GO111MODULE=on GOFLAGS='-mod=vendor' $(MAKE) build

.PHONY: clean
clean:
	rm -rf target

target:
	mkdir target

.PHONY: format
format:
ifeq (, $(shell which goimports))
	go get golang.org/x/tools/cmd/goimports
endif
	@echo "Running goimports ..."
	@goimports -w -local github.com/tomcz/openldap_exporter $(shell find . -type f -name '*.go' | grep -v '/vendor/')

.PHONY: lint
lint:
ifeq (, $(shell which staticcheck))
	go install honnef.co/go/tools/cmd/staticcheck@2021.1
endif
	@echo "Running staticcheck ..."
	@staticcheck $(shell go list ./... | grep -v /vendor/)

compile = GOOS=$1 GOARCH=amd64 go build -mod=mod -ldflags "${LDFLAGS}" -o target/openldap_exporter-$1 ./cmd/openldap_exporter

.PHONY: build-linux
build-linux: target
	$(call compile,linux)

.PHONY: build-darwin
build-darwin: target
	$(call compile,darwin)

.PHONY: build
build: target build-linux build-darwin
