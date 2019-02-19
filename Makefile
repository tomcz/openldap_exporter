GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
BASE_DIR := $(shell git rev-parse --show-toplevel 2>/dev/null)
LDFLAGS := -X exporter.version=${GITCOMMIT}
GO_PATH := ${BASE_DIR}/code

precommit: clean format lint build

deps:
	dep ensure

clean:
	rm -Rf build/

format:
	go fmt

lint:
	go vet

compile = GOOS=$1 GOARCH=amd64 \
	go build -ldflags "${LDFLAGS}" \
	-o build/openldap_exporter-$1


build:
	$(call compile,linux)
	$(call compile,darwin)

travis: clean deps
	$(call compile,linux)
