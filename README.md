# Companies House Payments API

[![GoDoc](https://godoc.org/github.com/companieshouse/payments.api.ch.gov.uk?status.svg)](https://godoc.org/github.com/companieshouse/payments.api.ch.gov.uk)
[![Go Report Card](https://goreportcard.com/badge/github.com/companieshouse/payments.api.ch.gov.uk)](https://goreportcard.com/report/github.com/companieshouse/payments.api.ch.gov.uk)

The Companies House API for handling payments.

## Requirements
In order to run this API locally you will need to install the following:

- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/downloads)
- [MongoDB](https://www.mongodb.com/)

## Getting Started
1. Clone this repository: `go get github.com/companieshouse/payments.api.ch.gov.uk`
1. Build the executable: `make build`

## Configuration

Variable                | Default   | Description
:-----------------------|:----------|:------------
`BIND_ADDR`             |           | Payments API Port
`MONGODB_URL`           |           | MongoDB URL
`MONGODB_DATABASE`      | `payments`| MongoDB database name
`MONGODB_COLLECTION`    | `payments`| MongoDB collection name
`DOMAIN_WHITELIST`      |           | List of valid domains for the Resource URL
`PAYMENTS_WEB_URL`      |           | URL for the [Payments Web](https://github.com/companieshouse/payments.web.ch.gov.uk) service
`PAYMENTS_API_URL`      |           | URL for the Payments API
`GOV_PAY_URL`           |           | URL for [GOV.UK Pay](https://www.payments.service.gov.uk)
`GOV_PAY_BEARER_TOKEN`  |           | Bearer Token for [GOV.UK Pay](https://www.payments.service.gov.uk)
`EXPIRY_TIME_IN_MINUTES`|           | Number of minutes before a payment session expires
`KAFKA_BROKER_ADDR`     |           | Kafka Broker address
`SCHEMA_REGISTRY_URL`   |           | Schema Registry URL

## Endpoints

Method    | Path                                            | Description
:---------|:------------------------------------------------|:-----------
**GET**   | /healthcheck                                    | Checks the health of the service
**POST**  | /payments                                       | Create Payment Session
**GET**   | /payments/{payment_id}                          | Get Payment Session
**POST**  | /payments/{payment_id}/refunds                 | Create Refund
**PATCH** | /private/payments/{payment_id}                  | Patch Payment Session
**POST**  | /private/payments/{payment_id}/external-journey | Returns URL for external Payment Provider
**GET**   | /callback/payments/govpay/{payment_id}          | [GOV.UK Pay](https://www.payments.service.gov.uk) callback

The `Create Payment Session` **POST** endpoint receives a `body` in the following format:

```json
{
    "redirect_uri": "string",
    "reference": "string",
    "resource": "string",
    "state": "string"
}
```
and returns a Payment Resource in the response:

```json
{
    "amount": "string",
    "available_payment_methods": [
        "string"
    ],
    "completed_at": "date-time",
    "created_at": "date-time",
    "created_by": {
        "email": "string",
        "forename": "string",
        "id": "string",
        "surname": "string"
    },
    "description": "string",
    "links": {
        "journey": "string",
        "resource": "string",
        "self": "string"
    },
    "payment_method": "string",
    "reference": "string",
    "status": "string"
}
```
---
The `Create Refund` **POST** endpoint receives a `body` in the following format:

```json
{
    "amount": 800
}
```
and returns a Refund Resource in the response:

```json
{
    "refund_id": "string",
    "created_date_time": "string",
    "amount": 800,
    "status": "string"
}
```

## External Payment Providers

The only external payment provider currently supported is [GOV.UK Pay](https://www.payments.service.gov.uk).

## Docker support

Pull image from private CH registry by running `docker pull 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payments.api.ch.gov.uk:latest` command or run the following steps to build image locally:

1. `export SSH_PRIVATE_KEY_PASSPHRASE='[your SSH key passhprase goes here]'` (optional, set only if SSH key is passphrase protected)
2. `docker build --build-arg SSH_PRIVATE_KEY="$(cat ~/.ssh/id_rsa)" --build-arg SSH_PRIVATE_KEY_PASSPHRASE -t 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payments.api.ch.gov.uk:latest .`
