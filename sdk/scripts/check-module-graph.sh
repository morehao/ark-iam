#!/usr/bin/env bash
# Module 依赖方向守卫（CI 可强制）：确保多模块 DAG 无环、无反向依赖。
#
# 目标依赖方向（单向）：
#   contract → (无 ark-iam 依赖，仅 stdlib + jwt)
#   pkg      → contract            （不得 → sdk / apps）
#   sdk      → contract, pkg, apps （对外入口；rp 不依赖框架）
#   apps/*   → contract, pkg       （不得 → sdk，组合根 cmd 例外另行处理）
#
# 一旦出现 pkg → sdk 或 apps(库) → sdk 边，即为解环退化，立即失败。
#
# 用法：bash sdk/scripts/check-module-graph.sh   （仓库根执行）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

fail=0

# 1) pkg 不得依赖 sdk（GOWORK=off 单建视角下的 module 图）。
bad_pkg=$( (cd pkg && GOWORK=off go mod graph 2>/dev/null | grep -E '^github\.com/morehao/ark-iam/pkg .*github\.com/morehao/ark-iam/sdk' || true) )
if [ -n "$bad_pkg" ]; then
  echo "❌ pkg → sdk 依赖边存在（解环退化）："
  echo "$bad_pkg"
  fail=1
fi

# 2) contract 不得依赖任何 ark-iam 其它 module。
bad_contract=$( (cd contract && GOWORK=off go mod graph 2>/dev/null | grep -E '^github\.com/morehao/ark-iam/contract .*github\.com/morehao/ark-iam/(pkg|sdk|backend)' || true) )
if [ -n "$bad_contract" ]; then
  echo "❌ contract 反向依赖 ark-iam module（应仅 stdlib + jwt）："
  echo "$bad_contract"
  fail=1
fi

# 3) pkg 独立单建（GOWORK=off）必须通过（证明 pkg 不依赖 sdk）。
if ! (cd pkg && GOWORK=off go build ./... >/dev/null 2>&1); then
  echo "❌ pkg GOWORK=off go build 失败（pkg 仍依赖 sdk 或其它不可脱环项）"
  fail=1
fi

# 4) apps 库代码（非 cmd 组合根、非 _test）不得 import sdk。
#    组合根 cmd/main.go 用 sdk/server 装配自身属预期；库层（ctroidc/router/service/...）
#    必须只依赖 contract/pkg——验签器/密钥来源经装配层注入，不直接 import sdk。
bad_apps_lib=$(grep -rln "github.com/morehao/ark-iam/sdk" backend/apps --include="*.go" 2>/dev/null | grep -v "_test.go" | grep -v "/cmd/" || true)
if [ -n "$bad_apps_lib" ]; then
  echo "❌ apps 库代码仍 import sdk（应经注入，不直接依赖 sdk）："
  echo "$bad_apps_lib"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "✅ module 依赖方向守卫通过：无 pkg → sdk / apps(库) → sdk / contract 反向依赖，pkg 可独立单建"
