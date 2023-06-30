ARG GO_VERSION=1.20
ARG VCS_REF
ARG BUILD_DATE
ARG GIT_COMMIT
ARG GIT_TAG

FROM golang:${GO_VERSION} as builder

RUN mkdir -p /go/src/github.com/4data-ch/openldap_exporter
WORKDIR /go/src/github.com/4data-ch/openldap_exporter

COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
  -ldflags="-w -s -X github.com/4data-ch/openldap_exporter.commit=${GIT_COMMIT} -X github.com/4data-ch/openldap_exporter.tag=${GIT_TAG}" \
  -o /go/bin/openldap_exporter


FROM scratch

LABEL org.opencontainers.image.title="OpenLDAP Prometheus Exporter"
LABEL org.opencontainers.image.description="This is a simple service that scrapes metrics from OpenLDAP and exports them via HTTP for Prometheus consumption."
LABEL org.opencontainers.image.authors="Lucien Stuker <lucien.stuker@4data.ch>"
LABEL org.opencontainers.image.url="https://github.com/4data-ch/openldap_exporter"
LABEL org.opencontainers.image.documentation="https://github.com/4data-ch/openldap_exporter"
LABEL org.opencontainers.image.source="https://github.com/4data-ch/openldap_exporter.git"
LABEL org.label-schema.vcs-ref=$VCS_REF
LABEL org.opencontainers.image.created=$BUILD_DATE
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.base.name="scratch"

USER exporter

COPY --from=builder /go/src/github.com/4data-ch/openldap_exporter/target/openldap_exporter /usr/bin/openldap_exporter
ENTRYPOINT ["/usr/bin/openldap_exporter"]