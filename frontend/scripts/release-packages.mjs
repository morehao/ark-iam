#!/usr/bin/env node
// 前端门面包 @morehao/ark-iam-react 的一键发布脚本（P0 T7 / A 方案：对外只一个包）。
//
// @morehao/ark-iam-react 是聚合壳：构建时把 @ark-iam/{types,api,auth,ui}(内部包) 的代码与类型**打进自身 dist**，
// 故发布时只有 @morehao/ark-iam-react 一个包需要 publish，消费者无需再装内部四包。
//
// 用法：
//   node scripts/release-packages.mjs version 1.0.1               # 改 @morehao/ark-iam-react 版本
//   node scripts/release-packages.mjs pack                        # build + pnpm pack 产 tarball
//   node scripts/release-packages.mjs publish [--registry <url>]  # 发布到 registry
//
// 说明：内部四包仅作为 devDependencies(workspace:*) 参与构建，不会随发布带出。
// 发布 scope 为 @morehao（公共 npm 免费 org，已验证可建）。
// 私有 registry / GitHub Packages 的 @morehao:registry 配置见 frontend/.npmrc 注释模板。

import { readFileSync, writeFileSync } from 'node:fs'
import { execSync } from 'node:child_process'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const pkgRoot = resolve(__dirname, '..', 'packages')
// 对外发布的门面包（内部四包不单独发布）。
const PUBLISH_PKGS = ['react']

function run(cmd, cwd) {
  console.log(`$ (${cwd.replace(pkgRoot, 'packages')}) ${cmd}`)
  execSync(cmd, { cwd, stdio: 'inherit' })
}

const [action, ...rest] = process.argv.slice(2)

if (action === 'version') {
  const v = rest[0]
  if (!v || !/^\d+\.\d+\.\d+/.test(v)) {
    console.error('用法: release-packages.mjs version <x.y.z>')
    process.exit(1)
  }
  for (const p of PUBLISH_PKGS) {
    const file = resolve(pkgRoot, p, 'package.json')
    const json = JSON.parse(readFileSync(file, 'utf8'))
    json.version = v
    writeFileSync(file, JSON.stringify(json, null, 2) + '\n')
    console.log(`@morehao/ark-iam-react -> ${v}`)
  }
} else if (action === 'pack' || action === 'publish') {
  for (const p of PUBLISH_PKGS) {
    const cwd = resolve(pkgRoot, p)
    run('pnpm run build', cwd)
    if (action === 'pack') {
      run('pnpm pack', cwd)
    } else {
      const regIdx = rest.indexOf('--registry')
      const registry = regIdx >= 0 ? rest[regIdx + 1] : ''
      run(`pnpm publish${registry ? ` --registry ${registry}` : ''} --no-git-checks`, cwd)
    }
  }
} else {
  console.error('用法: release-packages.mjs <version|pack|publish> [args]')
  process.exit(1)
}
