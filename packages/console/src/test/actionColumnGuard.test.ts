import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * 操作列规范守卫（对应 AGENTS.md「列表页操作列规范」R1 与 DESIGN.md §7.3）：
 * 全仓 console 源码里每一处 `actionColumn<...>({...})` 调用都必须
 *   - 显式传 `variant: 'icon'`（图标 + 文字，项目既定风格）
 *   - 每个 RowAction 都带语义 `icon`
 * 否则会渲染成纯文字 link（不合规的历史形态，曾出现在设置页/用户身份 Tab）。
 *
 * 本用例纯静态扫描源码文本，不渲染组件，作为防回归护栏。
 */

const SRC_ROOT = join(__dirname, '..') // packages/console/src

function walk(dir: string): string[] {
  const out: string[] = []
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) {
      // 跳过测试目录自身（含本守卫用例），避免把用例里的示例字符串算进去
      if (name === 'test' || name === '__tests__') continue
      out.push(...walk(p))
    } else if (p.endsWith('.tsx') || p.endsWith('.ts')) out.push(p)
  }
  return out
}

// 从 `(` 起做括号配对，返回 actionColumn 调用块的结束下标（含）。
function matchParen(s: string, openIdx: number): number {
  let depth = 0
  for (let j = openIdx; j < s.length; j++) {
    if (s[j] === '(') depth++
    else if (s[j] === ')') {
      depth--
      if (depth === 0) return j
    }
  }
  return s.length - 1
}

interface Block {
  file: string
  line: number
  text: string
}

function collectActionColumnBlocks(): Block[] {
  const blocks: Block[] = []
  for (const file of walk(SRC_ROOT)) {
    // 守卫文件自身不参与
    if (file.endsWith('__tests__/actionColumnGuard.test.ts')) continue
    if (file.includes('actionColumn.guard')) continue
    const s = readFileSync(file, 'utf8')
    const re = /actionColumn<[^>]*>\(/g
    let m: RegExpExecArray | null
    while ((m = re.exec(s)) !== null) {
      const openIdx = s.indexOf('(', m.index)
      const end = matchParen(s, openIdx)
      const line = s.slice(0, m.index).split('\n').length
      blocks.push({ file, line, text: s.slice(m.index, end + 1) })
    }
  }
  return blocks
}

describe('列表操作列规范守卫', () => {
  const blocks = collectActionColumnBlocks()

  it('至少扫描到多处 actionColumn 调用（哨兵，避免扫描失效假绿）', () => {
    expect(blocks.length).toBeGreaterThan(5)
  })

  it('每处 actionColumn 都显式传 variant:\'icon\' 且每个 action 都带 icon', () => {
    const violations: string[] = []
    for (const b of blocks) {
      const hasVariant = /variant:\s*['"]icon['"]/.test(b.text)
      const nLabel = (b.text.match(/label:/g) || []).length
      const nIcon = (b.text.match(/icon:/g) || []).length
      const issues: string[] = []
      if (!hasVariant) issues.push("缺 variant:'icon'")
      if (nIcon < nLabel) issues.push(`icon(${nIcon}) < label(${nLabel})，有动作缺语义 icon`)
      if (issues.length) {
        violations.push(`${b.file.replace(SRC_ROOT + '/', '')}:${b.line} — ${issues.join('；')}`)
      }
    }
    expect(violations, violations.join('\n')).toEqual([])
  })
})
