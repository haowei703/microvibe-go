#!/bin/bash
# wrk 压测工具 Docker 封装脚本
# 用法: ./scripts/wrk.sh [wrk参数] <url>
#
# 示例:
#   ./scripts/wrk.sh -t4 -c100 -d30s http://host.docker.internal:8080/api/v1/health
#   ./scripts/wrk.sh -t2 -c50 -d10s --latency http://host.docker.internal:8080/api/v1/feed

exec docker run --rm williamyeh/wrk "$@"
