import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import type { OAuthClientDetail as OAuthClientDetailType, OAuthClientItem } from '@ark-iam/types'
import { OAUTH_CLIENT_DETAIL_ROUTE, OAUTH_CLIENT_LIST_PATH, oauthClientDetailPath } from '../../routes'

const mockGetOAuthClientPageList = vi.fn()
const mockGetOAuthClientDetail = vi.fn()
const mockListOAuthSecrets = vi.fn()
vi.mock('@ark-iam/api', () => ({
  createOAuthClient: vi.fn(),
  createOAuthSecret: vi.fn(),
  deleteOAuthClient: vi.fn(),
  deleteOAuthSecret: vi.fn(),
  getApplicationPageList: vi.fn(),
  getOAuthClientDetail: (...args: unknown[]) => mockGetOAuthClientDetail(...args),
  getOAuthClientPageList: (...args: unknown[]) => mockGetOAuthClientPageList(...args),
  listOAuthSecrets: (...args: unknown[]) => mockListOAuthSecrets(...args),
  updateOAuthClient: vi.fn(),
}))

const OAuthClientList = (await import('./index')).default
const OAuthClientDetail = (await import('./Detail')).default

const createdAt = 1789142400
const updatedAt = 1789228800
/** 内置客户端的客户端编码（= OIDC client_id，种子写入 model.ApplicationClientEntity.Code）。 */
const clientCode = 'platform-admin-web'
/** 所属应用名称（后端列表/详情回填 appName）。 */
const appName = '平台管理后台'

const clients: OAuthClientItem[] = [
  {
    applicationClientID: '01a09372-0000-7000-8000-00000000f153',
    appID: '01a09372-0000-7000-8000-00000000a2b2',
    appName,
    code: clientCode,
    name: '平台管理后台客户端',
    source: 'builtin',
    status: 'enable',
    grantTypes: ['authorization_code', 'refresh_token'],
    tokenEndpointAuthMethod: 'none',
    createdAt,
    updatedAt,
  },
]

const detail: OAuthClientDetailType = {
  ...clients[0],
  tenantID: '01a09372-0000-7000-8000-00000000c001',
  redirectURIs: ['http://localhost:4001/auth/callback'],
  postLogoutRedirectURIs: ['http://localhost:4001/login'],
  backChannelLogoutURI: 'http://localhost:8100/oidc/bc-logout/platform',
  responseTypes: ['code'],
  allowedOrigins: [],
  requirePKCE: 1,
  requireAuthTime: 0,
  defaultScopes: ['openid', 'profile', 'email'],
  accessTokenTTL: 900,
  refreshTokenTTL: 2592000,
}

/** 按真实路由注册列表与详情（path 取自 routes.ts，与 App.tsx 同一真相源）。 */
function renderApp(initialEntry: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path={OAUTH_CLIENT_LIST_PATH} element={<OAuthClientList />} />
        <Route path={OAUTH_CLIENT_DETAIL_ROUTE} element={<OAuthClientDetail />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  mockGetOAuthClientPageList.mockReset().mockResolvedValue({ list: clients, total: clients.length })
  mockGetOAuthClientDetail.mockReset().mockResolvedValue(detail)
  mockListOAuthSecrets.mockReset().mockResolvedValue({ total: 0, secrets: [] })
})

describe('OAuth 客户端列表', () => {
  /**
   * 回归：客户端编码列展示的是 OIDC client_id（后端 DTO/DB 字段名都是 `code`，列名与字段名同构）。
   * 曾因前端读 `clientID` 而字段缺失，整列渲染成 '-'（内置客户端的可读 client_id 全部不可见）。
   */
  it('客户端编码 列展示后端 code（OIDC client_id）', async () => {
    renderApp(OAUTH_CLIENT_LIST_PATH)

    expect(await screen.findByText(clientCode)).toBeInTheDocument()
    // 表头在 antd 内部会渲染多份（度量行），只断言存在
    expect(screen.getAllByText('客户端编码').length).toBeGreaterThan(0)
  })

  /** 所属应用列展示应用名称（后端回填 appName），不再展示内部 appID。 */
  it('所属应用列展示应用名称', async () => {
    renderApp(OAUTH_CLIENT_LIST_PATH)

    expect(await screen.findByText(appName)).toBeInTheDocument()
    expect(screen.queryByText(clients[0].appID)).not.toBeInTheDocument()
  })

  /** 列顺序：名称在客户端编码之前（列表页主键在前、编码其次）。 */
  it('名称列排在客户端编码列之前', async () => {
    renderApp(OAUTH_CLIENT_LIST_PATH)

    await screen.findByText(clientCode)
    const headers = screen.getAllByRole('columnheader').map((th) => th.textContent)
    expect(headers.indexOf('名称')).toBeGreaterThanOrEqual(0)
    expect(headers.indexOf('名称')).toBeLessThan(headers.indexOf('客户端编码'))
  })

  /**
   * 回归：点名称必须能进详情。
   * 曾因跳转写成 `/oauthClient/{id}`（驼峰）而 App.tsx 注册的是 `/oauth-client/:id`，
   * 路由不匹配 → 详情渲染空白。本用例用 routes.ts 的真实 path 注册路由并按名称点击跳转。
   */
  it('点击名称进入详情页', async () => {
    renderApp(OAUTH_CLIENT_LIST_PATH)

    fireEvent.click(await screen.findByText(clients[0].name))

    // 详情页独有元素（返回列表 + 密钥管理卡片）
    expect(await screen.findByRole('button', { name: /返回列表/ })).toBeInTheDocument()
    expect(mockGetOAuthClientDetail).toHaveBeenCalledWith(clients[0].applicationClientID)
    expect(await screen.findByText('密钥管理')).toBeInTheDocument()
  })
})

describe('OAuth 客户端详情', () => {
  it('客户端编码 展示后端 code（OIDC client_id）', async () => {
    renderApp(oauthClientDetailPath(clients[0].applicationClientID))

    expect(await screen.findByText('客户端编码')).toBeInTheDocument()
    expect(await screen.findByText(clientCode)).toBeInTheDocument()
  })

  it('所属应用展示应用名称', async () => {
    renderApp(oauthClientDetailPath(clients[0].applicationClientID))

    expect(await screen.findByText('所属应用')).toBeInTheDocument()
    expect(await screen.findByText(appName)).toBeInTheDocument()
  })

  /** 返回列表按钮必须回到列表路由（曾误写为 /oauthClient）。 */
  it('返回列表回到列表页', async () => {
    renderApp(oauthClientDetailPath(clients[0].applicationClientID))

    fireEvent.click(await screen.findByRole('button', { name: /返回列表/ }))

    expect(await screen.findByRole('button', { name: /新建客户端/ })).toBeInTheDocument()
  })
})
