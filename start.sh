#!/bin/bash
# ZebraCICD 启动脚本（集成 Nacos）

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

# Nacos 连接配置
export NACOS_SERVER_ADDR="localhost:8848"
export NACOS_NAMESPACE=""
export NACOS_USERNAME="nacos"
export NACOS_PASSWORD="nacos"
export NACOS_GROUP="DEFAULT_GROUP"
export ZEBRA_NACOS_SERVER_ADDR="$NACOS_SERVER_ADDR"
export ZEBRA_NACOS_NAMESPACE="$NACOS_NAMESPACE"
export ZEBRA_NACOS_USERNAME="$NACOS_USERNAME"
export ZEBRA_NACOS_PASSWORD="$NACOS_PASSWORD"
export ZEBRA_NACOS_GROUP="$NACOS_GROUP"

# Go 模块下载代理（当前网络环境下默认 proxy.golang.org 不稳定）
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOSUMDB="${GOSUMDB:-sum.golang.google.cn}"

# 服务配置（可选，用于服务注册）
export SERVICE_IP="127.0.0.1"
export SERVICE_PORT="4123"

echo "=========================================="
echo "启动 ZebraCICD 服务"
echo "=========================================="
echo "Nacos 服务器: $NACOS_SERVER_ADDR"
echo "命名空间: ${NACOS_NAMESPACE:-public(default)}"
echo "配置分组: $NACOS_GROUP"
echo "服务地址: $SERVICE_IP:$SERVICE_PORT"
echo "Go 代理: $GOPROXY"
echo "=========================================="
echo ""

# 启动服务
go run main.go
