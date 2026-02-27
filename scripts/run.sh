#!/bin/sh
# Example run script
export ZEBRA_DATABASE_URL="${ZEBRA_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:5432/zebra?sslmode=disable}"
export ZEBRA_PORT="${ZEBRA_PORT:-9527}"
go run ./cmd