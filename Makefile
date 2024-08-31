GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_TAG := $(shell git describe --tags 2>/dev/null)

LDFLAGS := -s -w -X main.commit=${GITCOMMIT}
LDFLAGS := ${LDFLAGS} -X main.tag=${GIT_TAG}
OUTFILE ?= openldap_exporter

.PHONY: precommit
precommit: clean format lint compile

.PHONY: build
build: clean compile

.PHONY: commit
commit: clean cross-compile
	ls -lha target/

.PHONY: clean
clean:
	rm -rf target

target:
	mkdir target

.PHONY: format
format:
	goimports -w main.go

.PHONY: lint
lint:
	golangci-lint run

.PHONY: compile
compile: target
	go build -ldflags "${LDFLAGS}" -o target/${OUTFILE} main.go
	gzip -c < target/${OUTFILE} > target/${OUTFILE}.gz

.PHONY: cross-compile
cross-compile:
	OUTFILE=openldap_exporter-linux-amd64 GOOS=linux GOARCH=amd64 $(MAKE) compile
	OUTFILE=openldap_exporter-linux-nocgo CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(MAKE) compile
	OUTFILE=openldap_exporter-osx-amd64 GOOS=darwin GOARCH=amd64 $(MAKE) compile
	OUTFILE=openldap_exporter-osx-arm64 GOOS=darwin GOARCH=arm64 $(MAKE) compile
	(cd target && find . -name '*.gz' -exec sha256sum {} \;) > target/verify.sha256
