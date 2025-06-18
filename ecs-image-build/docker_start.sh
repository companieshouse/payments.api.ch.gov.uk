#!/bin/bash
#
# Start script for payments-api
PORT="20188"

# Read brokers and topics from environment and split on comma
IFS=',' read -ra BROKERS <<< "${KAFKA_BROKER_ADDR}"

# Ensure we only populate the broker address and topic via application arguments
unset KAFKA_BROKER_ADDR

exec ./payments.api.ch.gov.uk "-bind-addr=:${PORT}" $(for broker in "${BROKERS[@]}"; do echo -n "-broker-addr=${broker} "; done)