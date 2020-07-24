# syntax=docker/dockerfile:1.0.0-experimental

FROM 169942020521.dkr.ecr.eu-west-1.amazonaws.com/ci-golang-build:latest as builder

WORKDIR /build

COPY . ./

RUN --mount=type=ssh go fmt ./... && go build

FROM golang:1.14-alpine

WORKDIR /app

COPY --from=builder /build/payments.api.ch.gov.uk ./

CMD ["/app/payments.api.ch.gov.uk"]
