FROM alpine:3.8

RUN apk add --no-cache ca-certificates

RUN mkdir -p /opt/ignition
ADD vendor/github.com/giantswarm/k8scloudconfig/ /opt/ignition

ADD ./azure-operator /azure-operator

ENTRYPOINT ["/azure-operator"]
