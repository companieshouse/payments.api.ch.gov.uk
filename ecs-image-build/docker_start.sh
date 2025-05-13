#!/bin/bash
#
# Start script for payments-api
PORT="20188"

exec ./payments.api.ch.gov.uk "-bind-addr=:${PORT}"