#!/usr/bin/env bash
# Module 依赖方向守卫（CI 可强制）：确保多模块 DAG 无环、无反向依赖。
#
# 目标依赖方向（单向）：
#   contract    → (无 ark-iam 依赖，仅 stdlib + jwt)
#   pkg         → contract + apps/*   （appassembly 装配内核合法依赖 apps 域库；仍不得 → sdk）
#   sdk         → contract            （仅 sdk/rp：框架无关验签引擎；不再含 server/apps）
#   apps/*(库)  → contract, pkg       （不得 → sdk）
#   apps/*/app  → pkg/appassembly + sdk/rp   （组合根：注入验签器实现，允许 → sdk）
#   apps/*/cmd  → apps/*/app          （进程入口，允许 → sdk 经 app 转手）
#
# 一旦出现 pkg → sdk 边，或 apps 库层（非 app/cmd 组合根）→ sdk 边，即为解环退化，立即失败。
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
bad_contract=$( (cd contract && GOWORK=off go mod graph 2>/dev/null | grep -E '^github\.com/morehao/ark-iam/contract .*github\.com/morehao/ark-iam/(pkg|sdk|apps)' || true) )
if [ -n "$bad_contract" ]; then
  echo "❌ contract 反向依赖 ark-iam module（应仅 stdlib + jwt）："
  echo "$bad_contract"
  fail=1
fi

# 3) pkg 不得依赖 sdk（module 图视角；pkg 合法依赖 apps/*，故不再要求 pkg 能 GOWORK=off 独立单建）。
bad_pkg_sdk=$( (cd pkg && GOWORK=off go mod graph 2>/dev/null | grep -E '^github\.com/morehao/ark-iam/pkg .*github\.com/morehao/ark-iam/sdk' || true) )
if [ -n "$bad_pkg_sdk" ]; then
  echo "❌ pkg → sdk 依赖边存在（解环退化）："
  echo "$bad_pkg_sdk"
  fail=1
fi

# 4) apps 库代码（非组合根 app/、非 cmd 组合根、非 _test）不得 import sdk。
#    组合根 apps/*/app 与 apps/*/cmd 用 sdk/rp 实现验签器注入属预期；
#    库层（ctroidc/router/service/...）必须只依赖 contract/pkg——验签器/密钥来源经装配层注入。
bad_apps_lib=$(grep -rln "github.com/morehao/ark-iam/sdk" apps --include="*.go" 2>/dev/null | grep -v "_test.go" | grep -v "/cmd/" | grep -v "/app/" || true)
if [ -n "$bad_apps_lib" ]; then
  echo "❌ apps 库代码仍 import sdk（应经注入，不直接依赖 sdk）："
  echo "$bad_apps_lib"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "✅ module 依赖方向守卫通过：无 pkg → sdk、无 apps 库层 → sdk、contract 无反向依赖"
