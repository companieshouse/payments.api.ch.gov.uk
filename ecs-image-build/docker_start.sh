#!/bin/bash
#
# Start script for payments-api
PORT="20188"

exec ./payments-api "-bind-addr=:${PORT}"