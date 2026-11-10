#!/usr/bin/env bash
# SDK 依赖白名单断言（设计文档「技术选型 · 依赖白名单（CI 可强制）」）。
#
# 目的：SDK 必须保持"框架无关 + 无持久化依赖"；contract 作为最底层契约 module
# 必须仅依赖标准库（+ jwt/v5，因 claims 复用 jwt.RegisteredClaims）。一旦 contract
# 意外引入 gin/gorm/redis/pkg/apps/sdk，或 rp 核心引入框架/持久化依赖，即视为失败。
#
# 用法：bash sdk/scripts/check-deps.sh   （在仓库根执行，脚本自行定位）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"   # sdk/scripts -> repo root
cd "$REPO_ROOT"

FORBIDDEN='gorm\.io|github\.com/gin-gonic|github\.com/redis|go-elasticsearch|github\.com/morehao/ark-iam/pkg|github\.com/morehao/ark-iam/backend'

# 1) contract module：仅允许标准库 + jwt/v5（不得依赖任何 ark-iam 其它 module 或框架）。
contract_extra=$(cd contract && GOWORK=off go list -deps ./... 2>/dev/null | grep -E 'github\.com/morehao/ark-iam/(pkg|sdk|backend|contract/[a-z])|github\.com/gin-gonic|gorm\.io|github\.com/redis|go-elasticsearch' | grep -v '^github\.com/morehao/ark-iam/contract' || true)
if [ -n "$contract_extra" ]; then
  echo "❌ contract 依赖断言失败：contract 仅允许标准库 + jwt/v5"
  echo "$contract_extra"
  exit 1
fi
echo "✅ contract 依赖白名单通过：仅标准库 + jwt/v5"

# 2) sdk/rp（验签核心 + 客户端）：不得引入 Web 框架 / 持久化 / ark-iam pkg、apps。
violations=$( (cd sdk && go list -deps ./rp/... 2>/dev/null | grep -E "$FORBIDDEN" || true) )
if [ -n "$violations" ]; then
  echo "❌ sdk/rp 依赖白名单断言失败：rp 不得依赖持久化、Web 框架或 ark-iam pkg/apps"
  echo "$violations"
  exit 1
fi
echo "✅ sdk/rp 依赖白名单通过：仅标准库 + jwt/v5 + contract"

echo "✅ SDK 依赖白名单全部通过"
