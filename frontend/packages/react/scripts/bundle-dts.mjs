#!/usr/bin/env node
// 类型打包：把 @ark-iam/{types,api,auth,ui} 的 .d.ts 全部内联进 @morehao/ark-iam-react 的 dist，
// 使「对外只有一个 npm 包」时消费者无需再装/解析内部包的声明文件。
//
// 做法（离线、无第三方 dts 打包器）：
// 1. 把每个内部包的 dist/*.d.ts 递归复制到 react/dist/_types/<pkg>/ 下（保留目录结构）。
// 2. 在复制出的文件里把 `@ark-iam/<x>` 引用改写为 _types 内的相对路径。
// 3. 重写门面包入口 dist/{index,types,api,auth,ui}.d.ts 的 `@ark-iam/<x>` 为 `./_types/<x>/index`。

import { readFileSync, writeFileSync, mkdirSync, cpSync, readdirSync, statSync, existsSync } from 'node:fs'
import { resolve, dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const PKG_ROOT = resolve(__dirname, '..')
const DEST_BASE = resolve(PKG_ROOT, 'dist/_types')
const INTERNAL = ['types', 'api', 'auth', 'ui']

function walk(dir) {
  const out = []
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) out.push(...walk(p))
    else if (p.endsWith('.d.ts')) out.push(p)
  }
  return out
}

// 计算从某个 _types 下文件到 _types/<x>/index 的相对 specifier。
function rewriteInternalRefs(fileAbs, content) {
  return content.replace(/('@ark-iam\/([a-z]+))(['"])/g, (m, _full, pkg, quote) => {
    const target = join(DEST_BASE, pkg, 'index')
    let rel = relative(dirname(fileAbs), target).replace(/\\/g, '/')
    if (!rel.startsWith('.')) rel = './' + rel
    return `${quote}${rel}${quote}`
  })
}

// 1+2. 复制并改写内部包 d.ts
mkdirSync(DEST_BASE, { recursive: true })
for (const pkg of INTERNAL) {
  const srcDist = resolve(PKG_ROOT, '..', pkg, 'dist')
  if (!existsSync(srcDist)) {
    console.warn(`[bundle-dts] missing ${srcDist}, skip`)
    continue
  }
  const destDir = join(DEST_BASE, pkg)
  cpSync(srcDist, destDir, { recursive: true })
  for (const f of walk(destDir)) {
    const content = readFileSync(f, 'utf8')
    if (/@ark-iam\//.test(content)) writeFileSync(f, rewriteInternalRefs(f, content))
  }
}

// 3. 重写门面包入口 d.ts
const ENTRY_MAP = {
  'index.d.ts': ['types', 'api', 'auth', 'ui'],
  'types.d.ts': ['types'],
  'api.d.ts': ['api'],
  'auth.d.ts': ['auth'],
  'ui.d.ts': ['ui'],
}
for (const [entryFile, pkgs] of Object.entries(ENTRY_MAP)) {
  const abs = resolve(PKG_ROOT, 'dist', entryFile)
  const body = pkgs.map((p) => `export * from './_types/${p}/index';`).join('\n') + '\n'
  writeFileSync(abs, body)
}

console.log('[bundle-dts] done')
