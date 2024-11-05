#FROM 169942020521.dkr.ecr.eu-west-2.amazonaws.com/base/golang:1.19-bullseye-builder AS builder
FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.23:latest AS builder

# WORKDIR /build

# COPY . ./

# FROM 169942020521.dkr.ecr.eu-west-2.amazonaws.com/base/golang:debian11-runtime

# WORKDIR /app

# COPY --from=builder /build/payments.api.ch.gov.uk ./

CMD ["-bind-addr=:3055"]

EXPOSE 3055
