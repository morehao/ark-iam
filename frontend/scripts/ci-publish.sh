#!/usr/bin/env bash
# tag 触发的自动发布（vX.Y.Z → @morehao/ark-iam-react@X.Y.Z，经 npm Trusted Publishing / OIDC）。
#
# 由 .github/workflows/release.yml 调用；本地排障可：
#   RELEASE_TAG=v0.1.1 bash frontend/scripts/ci-publish.sh
# 三条守卫：tag 版本必须等于 packages/react/package.json 版本；同版本已发布则跳过；发布失败先回读 registry 再重试。
#
# 注意：
#   - 本仓是 pnpm monorepo，发布门面包必须用 `pnpm publish`（devDeps 里的 workspace:* 协议需由 pnpm rewrite）。
#   - 只发布 @morehao/ark-iam-react 一个包；内部四包 @ark-iam/{types,api,auth,ui} 不对外发布。
#   - 构建（内部四包 + 门面包）由 workflow 的 `pnpm run build:pkgs` 步骤完成，本脚本不重复 build。
set -euo pipefail

TAG="${RELEASE_TAG:-${GIT_TAG:-}}"
if [ -z "$TAG" ]; then
  echo "错误：未设置 RELEASE_TAG（应为 push 的 tag，如 v0.1.1）" >&2
  exit 1
fi
PKG_VERSION="${TAG#v}"

# 脚本位于 frontend/scripts/，工作区根为上一级；门面包在 packages/react。
cd "$(dirname "$0")/.."

name="@morehao/ark-iam-react"
pkg_dir="packages/react"
PKG_JSON_VERSION="$(node -p "require('./${pkg_dir}/package.json').version")"

echo "Publish check: ${name}@${PKG_JSON_VERSION} (tag=${TAG})"

# 1) tag 版本必须与门面包 package.json 一致，避免误发。
if [ "$PKG_JSON_VERSION" != "$PKG_VERSION" ]; then
  echo "错误：tag 版本 ${PKG_VERSION} 与 ${pkg_dir}/package.json 版本 ${PKG_JSON_VERSION} 不一致，拒绝发布" >&2
  exit 1
fi

# 2) 已发布则跳过（幂等，重跑安全）。
published="$(npm view "${name}@${PKG_VERSION}" version --registry https://registry.npmjs.org 2>/dev/null || true)"
if [ -n "$published" ]; then
  echo "${name}@${PKG_VERSION} 已发布，跳过"
  exit 0
fi

# 3) OIDC：job 提 id-token: write，npm CLI(>=11.5.1) 自动换取短时令牌，无需 NODE_AUTH_TOKEN。
#    重试理由：registry 偶发降级时 PUT 会回 E404（与权限失败同形），等几十秒通常就过；
#    每次失败先回读 registry —— 请求可能已被受理、只是响应丢了。
echo "npm version: $(npm --version)"
echo "Publishing ${name}@${PKG_VERSION}"

RETRY_BASE_SECONDS="${RETRY_BASE_SECONDS:-30}"
MAX_ATTEMPTS=3
attempt=1
while :; do
  # pnpm publish：rewrite workspace:* devDeps；--no-git-checks：CI 干净 checkout 无本地 git 状态依赖。
  if ( cd "${pkg_dir}" && pnpm publish --provenance --access public --no-git-checks --registry https://registry.npmjs.org ); then
    exit 0
  fi

  if npm view "${name}@${PKG_VERSION}" version --registry https://registry.npmjs.org >/dev/null 2>&1; then
    echo "${name}@${PKG_VERSION} 已出现在 registry，视为发布成功（首次响应丢失）"
    exit 0
  fi

  if [ "$attempt" -ge "$MAX_ATTEMPTS" ]; then
    echo "错误：pnpm publish 连续 ${attempt} 次失败" >&2
    echo "排查顺序：" >&2
    echo "  1) https://status.npmjs.org 的 Package publishing 是否 degraded" >&2
    echo "  2) 核对 Trusted Publisher（workflow=release.yml、Allowed actions 允许 npm publish）" >&2
    echo "  3) 恢复后重跑该 run：gh run rerun <run-id>" >&2
    exit 1
  fi

  wait_seconds=$((attempt * RETRY_BASE_SECONDS))
  echo "第 ${attempt} 次 publish 失败，${wait_seconds}s 后重试" >&2
  sleep "$wait_seconds"
  attempt=$((attempt + 1))
done
