FROM quay.io/giantswarm/golang:1.21.4 AS builder
ENV GO111MODULE=on
COPY go.mod /etc/go.mod
RUN git clone --depth 1 --branch $(cat /etc/go.mod | grep k8scloudconfig | awk '{print $2}') https://github.com/giantswarm/k8scloudconfig.git && cp -r k8scloudconfig /opt/k8scloudconfig

FROM alpine:3.17.3

RUN apk add --update ca-certificates \
    && rm -rf /var/cache/apk/*

RUN mkdir -p /opt/ignition
COPY --from=builder /opt/k8scloudconfig /opt/ignition

ADD ./azure-operator /azure-operator

ENTRYPOINT ["/azure-operator"]
