import { describe, expect, it } from 'vitest'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import { CompletionScreen } from './InstallPage'
import type { InstallInitializeResp } from '../install/types'

// AC-12：第③步提交后回显的控制台地址必须**等于服务端下发的配置值**。
//
// 为什么需要渲染级断言：`consoleEntries` 的单测（status.test.ts）只覆盖了
// "服务端 JSON -> 条目列表"这一跳，链路最后一跳 `<a href={entry.url}>` 是否原样输出
// 测不到——而"页面自己写死 http://localhost:4001"正是这条 AC 要防的回归：
// 那样非 localhost 部署的运维点进去必然打不开，且页面看起来完全正常。
//
// 用 renderToStaticMarkup 而不是 testing-library：react-dom 已是本 app 依赖，
// 不需要为一条 href 断言引入 jest-dom / @testing-library/react。
const result: InstallInitializeResp = {
  report: { tenantId: 't-1', changes: [] },
  adminUsername: 'opsadmin',
  loginURL: 'http://localhost:4001/login',
  consoles: { platformAdminWeb: 'https://admin.example.com', tenantAdminWeb: 'https://tenant.example.com' },
}

// 与 InstallPage.renderBody 的接线保持一致：控制台地址优先取 initialize 响应里的，
// 其次回落到进入页面时读到的 status（两个来源都是服务端值，页面不参与拼装）。
function render(resp: InstallInitializeResp, fallbackConsoles?: InstallInitializeResp['consoles']) {
  const consoles = resp.consoles ?? fallbackConsoles
  return renderToStaticMarkup(createElement(CompletionScreen, { result: resp, consoles }))
}

/** 取出「控制台入口」区块里的 href，避免把"前往登录"那个链接也算进来。 */
function consoleHrefs(html: string): string[] {
  return [...html.matchAll(/<a href="([^"]+)"[^>]*>[\s\S]*?<\/a>/g)]
    .map((m) => m[1])
    .filter((href) => href !== result.loginURL)
}

describe('AC-12：第③步回显的控制台地址来自服务端', () => {
  it('渲染出的 href 逐字等于服务端下发的地址', () => {
    const html = render(result)
    expect(html).toContain('控制台入口')
    expect(consoleHrefs(html)).toEqual([
      'https://admin.example.com',
      'https://tenant.example.com',
    ])
  })

  it('换一组服务端地址，渲染结果随之改变（证明不是页面内写死的常量）', () => {
    const other = render({
      ...result,
      consoles: { platformAdminWeb: 'https://a.internal:8443/console', tenantAdminWeb: 'http://b.internal' },
    })
    const hrefs = consoleHrefs(other)
    expect(hrefs).toEqual(['https://a.internal:8443/console', 'http://b.internal'])
    // 关键：既不是上一组值，也不含任何 localhost 默认值。
    // 注意只对 console 链接断言——登录入口（loginURL）本来就可能指向 localhost。
    for (const href of hrefs) {
      expect(href).not.toContain('admin.example.com')
      expect(href).not.toContain('localhost')
    }
  })

  it('地址里带路径与查询串时原样保留（不做客户端拼接/裁剪）', () => {
    const html = render({
      ...result,
      consoles: { platformAdminWeb: 'https://admin.example.com/oidc/console?x=1&y=2' },
    })
    expect(consoleHrefs(html)).toEqual(['https://admin.example.com/oidc/console?x=1&amp;y=2'])
  })

  it('响应未回传 consoles 时回落到进入页面读到的 status 值', () => {
    const html = render({ ...result, consoles: undefined }, {
      platformAdminWeb: 'https://from-status.example.com',
    })
    expect(consoleHrefs(html)).toEqual(['https://from-status.example.com'])
  })

  it('initialize 响应与 status 都有值时，以 initialize 响应为准（与页面接线一致）', () => {
    const html = render(result, { platformAdminWeb: 'https://stale-from-status.example.com' })
    expect(consoleHrefs(html)).toEqual([
      'https://admin.example.com',
      'https://tenant.example.com',
    ])
  })

  it('服务端未配置控制台地址时不渲染空区块（而不是给出死链）', () => {
    const html = render({ ...result, consoles: {} }, {})
    expect(html).not.toContain('控制台入口')
    expect(consoleHrefs(html)).toEqual([])
  })

  it('管理员账号与登录入口同样来自响应', () => {
    const html = render(result)
    expect(html).toContain('opsadmin')
    expect(html).toContain(`href="${result.loginURL}"`)
  })
})
