#!/usr/bin/env bash
# 前端依赖方向守卫：packages/{ui,api,auth,types} 不得依赖 @ark-iam/console。
#
# 依赖方向必须保持单向 DAG：console → {ui,api,auth,types}；react → console（仅 re-export）。
# 底层共享包若反向依赖页面包（console），会让「组件级」包被迫拉入 antd 页面实现、并形成环。
# 参照 sdk/scripts/check-module-graph.sh 的思路，做一次简单 grep 兜底。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

lower_pkgs=(ui api auth types)
violations=0

for p in "${lower_pkgs[@]}"; do
  pkgjson="packages/$p/package.json"
  if [[ ! -f "$pkgjson" ]]; then
    echo "✗ missing $pkgjson"
    violations=1
    continue
  fi
  if grep -q '"@ark-iam/console"' "$pkgjson"; then
    echo "✗ 依赖方向违规：packages/$p/package.json 依赖了 @ark-iam/console（底层包不得依赖页面包）"
    violations=1
  fi
done

if [[ "$violations" -ne 0 ]]; then
  echo "frontend-check-deps: FAILED"
  exit 1
fi

echo "frontend-check-deps: OK（packages/{ui,api,auth,types} 均未依赖 @ark-iam/console）"
