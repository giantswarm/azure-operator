FROM alpine:3.7

RUN apk add --no-cache ca-certificates

ADD ./azure-operator /azure-operator

ENTRYPOINT ["/azure-operator"]
