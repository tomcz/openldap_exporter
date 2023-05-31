FROM golang:alpine as build-env

RUN apk add git

COPY . /go/src/github.com/tomcz/openldap_exporter
WORKDIR /go/src/github.com/tomcz/openldap_exporter
# Build
ENV GOPATH=/go
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -v -a -ldflags "-s -w" -o /go/bin/openldap_exporter .

FROM library/alpine:3.15.0
COPY --from=build-env /go/bin/openldap_exporter /usr/bin/openldap_exporter
ENTRYPOINT ["openldap_exporter"]
