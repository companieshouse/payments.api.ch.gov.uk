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

Variable               | Default   | Description
:----------------------|:----------|:------------
`BIND_ADDR`            |           | Payment API Port
`MONGODB_DATABASE`     | `payments`| MongoDB database name
`MONGODB_COLLECTION`   | `payments`| MongoDB collection name
`DOMAIN_WHITELIST`     |           | List of valid domains for the Resource URL
`PAYMENTS_WEB_URL`     |           | URL for the [Payments Web](https://github.com/companieshouse/payments.web.ch.gov.uk) service
`GOV_PAY_URL`          |           | URL for [GOV.UK Pay](https://www.payments.service.gov.uk)
`GOV_PAY_BEARER_TOKEN` |           | Bearer Token for [GOV.UK Pay](https://www.payments.service.gov.uk)

## Endpoints

Method    | Path                                            | Description
:---------|:------------------------------------------------|:-----------
**GET**   | /healthcheck                                    | Checks the health of the service
**POST**  | /payments                                       | Create Payment Session
**GET**   | /payments/{payment_id}                          | Get Payment Session
**PATCH** | /private/payments/{payment_id}                  | Patch Payment Session
**POST**  | /private/payments/{payment_id}/external-journey | Returns URL for external Payment Provider

The `Create Payment Session` endpoint receives a `body` in the following format:

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

## External Payment Providers

The only external payment provider currently supported is [GOV.UK Pay](https://www.payments.service.gov.uk).