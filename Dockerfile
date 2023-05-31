FROM golang:1.20 as build-env

RUN apt install -yq git make 

COPY . /go/src/github.com/tomcz/openldap_exporter
WORKDIR /go/src/github.com/tomcz/openldap_exporter
# Build
ENV GOPATH=/go
RUN make compile

FROM library/debian:bullseye
COPY --from=build-env /go/src/github.com/tomcz/openldap_exporter/target/openldap_exporter /usr/bin/openldap_exporter
ENTRYPOINT ["openldap_exporter"]
