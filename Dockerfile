# STAGE 1: Build binaries
FROM golang:1.13 as build
WORKDIR /go/src/
COPY . .
RUN make build

# STAGE 2: Build final image with minimal content
FROM alpine
RUN apk --no-cache add libc6-compat
COPY --from=build /go/src/target/openldap_exporter-linux /openldap_exporter
EXPOSE 9330
ENTRYPOINT [ "/openldap_exporter" ]