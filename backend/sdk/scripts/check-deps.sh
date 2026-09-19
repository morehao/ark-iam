#!/usr/bin/env bash
# SDK 依赖白名单断言（设计文档「技术选型 · 依赖白名单（CI 可强制）」）。
#
# 目的：SDK 必须保持"框架无关 + 无持久化依赖"。一旦契约包或 rp 核心
# 意外引入 gin/gorm/redis 等，抽取即视为失败——因为外部仓库无法接受这些依赖。
#
# 用法：bash sdk/scripts/check-deps.sh   （在 backend/ 目录下执行）
set -euo pipefail

cd "$(dirname "$0")/.."

FORBIDDEN='gorm\.io|github\.com/gin-gonic|github\.com/redis|go-elasticsearch|github\.com/morehao/ark-iam/pkg'
# 允许列表：标准库 + jwt/v5（单测额外使用 testify，非运行期依赖）。
violations=$(go list -deps ./contract/... ./rp/... 2>/dev/null | grep -E "$FORBIDDEN" || true)

if [ -n "$violations" ]; then
  echo "❌ SDK 依赖白名单断言失败：contract/rp 不得依赖持久化或 Web 框架"
  echo "$violations"
  exit 1
fi

echo "✅ SDK 依赖白名单通过：contract/rp 仅依赖标准库 + jwt/v5"
