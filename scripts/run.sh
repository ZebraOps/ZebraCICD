#!/bin/sh
# Example run script
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOSUMDB="${GOSUMDB:-sum.golang.google.cn}"
export ZEBRA_DATABASE_URL="${ZEBRA_DATABASE_URL:-postgres://postgres:postgres@127.0.0.1:5432/zebra?sslmode=disable}"
export ZEBRA_PORT="${ZEBRA_PORT:-4123}"
go run ./cmd