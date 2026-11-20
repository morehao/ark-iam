import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, waitFor } from '@testing-library/react'
import type { ConnectorFactoryItem, ConnectorItem, TenantItem } from '@ark-iam/types'

const mockGetConnectorPageList = vi.fn()
const mockDeleteConnector = vi.fn()
const mockGetConnectorFactories = vi.fn()
const mockCreateConnector = vi.fn()
const mockUpdateConnector = vi.fn()
const mockGetConnectorDetail = vi.fn()
const mockTestConnector = vi.fn()
const mockGetTenantPageList = vi.fn()

vi.mock('@ark-iam/api', () => ({
  getConnectorPageList: (...args: unknown[]) => mockGetConnectorPageList(...args),
  deleteConnector: (...args: unknown[]) => mockDeleteConnector(...args),
  getConnectorFactories: (...args: unknown[]) => mockGetConnectorFactories(...args),
  createConnector: (...args: unknown[]) => mockCreateConnector(...args),
  updateConnector: (...args: unknown[]) => mockUpdateConnector(...args),
  getConnectorDetail: (...args: unknown[]) => mockGetConnectorDetail(...args),
  testConnector: (...args: unknown[]) => mockTestConnector(...args),
  getTenantPageList: (...args: unknown[]) => mockGetTenantPageList(...args),
}))

const ConnectorPage = (await import('./index')).default

const connectors: ConnectorItem[] = [
  {
    connectorID: 'cn-1',
    defaultTenantID: 't1',
    defaultTenantName: '平台运营中心',
    name: 'google-workspace',
    displayName: 'Google Workspace',
    protocol: 'oidc',
    provider: 'google',
    status: 'enable',
    allowAutoCreateUser: 'enable',
    allowAccountLink: 'enable',
    syncProfile: 'enable',
    enableTokenStorage: 'disable',
    config: { protocol: 'oidc', provider: 'google', issuer: 'https://accounts.google.com' },
    claimMapping: {},
    domainPolicy: {},
    createdAt: 1789142400,
    updatedAt: 1789228800,
  },
]

const factories: ConnectorFactoryItem[] = [
  {
    factoryID: 'oidc-google',
    protocol: 'oidc',
    provider: 'google',
    displayName: 'Google',
    isStandard: true,
    defaultScopes: ['openid', 'profile', 'email'],
    capabilities: ['authorize', 'callback'],
    configSchema: { required: ['issuer', 'clientId', 'clientSecret', 'redirectUri'] },
  },
]

const tenants: TenantItem[] = [
  {
    tenantID: 't1',
    code: 't_platform',
    name: '平台运营中心',
    status: 'active',
    type: 'platform',
    tag: 'default',
    dbUser: 'platform',
    createdAt: 1,
    updatedAt: 2,
  },
]

describe('身份连接器列表页（平台）', () => {
  beforeEach(() => {
    mockGetConnectorPageList.mockReset().mockResolvedValue({ list: connectors, total: 1 })
    mockDeleteConnector.mockReset().mockResolvedValue(undefined)
    mockGetConnectorFactories.mockReset().mockResolvedValue({ list: factories })
    mockCreateConnector.mockReset().mockResolvedValue({ connectorID: 'cn-new' })
    mockUpdateConnector.mockReset().mockResolvedValue(undefined)
    mockGetConnectorDetail.mockReset().mockResolvedValue(connectors[0])
    mockTestConnector.mockReset().mockResolvedValue({ success: true, message: '' })
    mockGetTenantPageList.mockReset().mockResolvedValue({ list: tenants, total: 1 })
  })

  it('首屏拉取平台连接器列表并渲染默认落租户列', async () => {
    render(<ConnectorPage />)
    await waitFor(() => expect(mockGetConnectorPageList).toHaveBeenCalled())
    await waitFor(() => expect(document.body.textContent).toContain('Google Workspace'))
    // 默认落租户列标题与 defaultTenantName 值均渲染
    expect(document.body.textContent).toContain('默认落租户')
    expect(document.body.textContent).toContain('平台运营中心')
  })

  it('点击「新建连接器」进入向导并加载工厂列表', async () => {
    render(<ConnectorPage />)
    await waitFor(() => expect(mockGetConnectorPageList).toHaveBeenCalled())

    fireEvent.click(document.querySelector('button.ant-btn-primary') as HTMLElement)

    await waitFor(() => expect(mockGetConnectorFactories).toHaveBeenCalled())
    await waitFor(() => expect(document.body.textContent).toContain('选择提供商'))
  })

  it('切换状态筛选会带 status 重新拉取', async () => {
    render(<ConnectorPage />)
    await waitFor(() => expect(mockGetConnectorPageList).toHaveBeenCalled())

    const enableOption = Array.from(document.querySelectorAll('.ant-segmented-item-label')).find(
      (el) => (el.textContent || '').trim() === '启用',
    ) as HTMLElement | undefined
    expect(enableOption).toBeTruthy()
    fireEvent.click(enableOption as HTMLElement)

    await waitFor(() =>
      expect(mockGetConnectorPageList).toHaveBeenCalledWith(expect.objectContaining({ status: 'enable' })),
    )
  })
})
