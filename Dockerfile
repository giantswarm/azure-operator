FROM alpine:3.5

RUN apk add --no-cache ca-certificates

ADD ./azure-operator /azure-operator

ENTRYPOINT ["/azure-operator"]
