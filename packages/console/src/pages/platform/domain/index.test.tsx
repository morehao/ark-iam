import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { DomainStatusTag } from '@ark-iam/ui'
import type { DomainItem } from '@ark-iam/types'

const mockGetDomainPageList = vi.fn()
const mockCreateDomain = vi.fn()
const mockVerifyDomain = vi.fn()
const mockActivateDomain = vi.fn()
const mockDisableDomain = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createDomain: (...args: unknown[]) => mockCreateDomain(...args),
  deleteDomain: vi.fn(),
  disableDomain: (...args: unknown[]) => mockDisableDomain(...args),
  enableDomain: vi.fn(),
  getDomainDetail: vi.fn(),
  getDomainPageList: (...args: unknown[]) => mockGetDomainPageList(...args),
  activateDomain: (...args: unknown[]) => mockActivateDomain(...args),
  releaseDomain: vi.fn(),
  retireDomain: vi.fn(),
  tlsReadyDomain: vi.fn(),
  verifyDomain: (...args: unknown[]) => mockVerifyDomain(...args),
}))

const DomainList = (await import('./index')).default

const draftDomain: DomainItem = {
  id: 'd1',
  tenantID: 't1',
  domain: 'example.com',
  domainRole: 'primary',
  domainUsage: 'login',
  status: 'draft',
  createdAt: 1700000000,
  updatedAt: 1700000000,
}

const activeAlias: DomainItem = {
  id: 'd2',
  tenantID: 't1',
  domain: 'alias.example.com',
  domainRole: 'alias',
  domainUsage: 'login',
  status: 'active',
  createdAt: 1700000000,
  updatedAt: 1700000000,
}

/** P0 回归：域名状态机 7 态、动作端点驱动、创建入参不含旧验证态字段。 */
describe('域名页 P0 状态机', () => {
  beforeEach(() => {
    mockGetDomainPageList.mockReset()
    mockCreateDomain.mockReset()
    mockVerifyDomain.mockReset()
    mockActivateDomain.mockReset()
    mockDisableDomain.mockReset()
  })

  it('draft 状态展示「草稿」tag，且只显示「验证」动作按钮', async () => {
    mockGetDomainPageList.mockResolvedValue({ list: [draftDomain], total: 1 })
    render(<DomainList />)

    expect(await screen.findByText('example.com')).toBeInTheDocument()
    expect(screen.getByText('草稿')).toBeInTheDocument()
    // draft → 可「验证」；不应出现「激活」「停用」
    expect(screen.getByRole('button', { name: /验证/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /激活/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /停用/ })).not.toBeInTheDocument()
  })

  it('点击「验证」调用 verify 端点', async () => {
    mockGetDomainPageList.mockResolvedValue({ list: [draftDomain], total: 1 })
    mockVerifyDomain.mockResolvedValue({ domainId: 'd1', status: 'verified' })
    render(<DomainList />)

    fireEvent.click(await screen.findByRole('button', { name: /验证/ }))
    await waitFor(() => expect(mockVerifyDomain).toHaveBeenCalledWith('d1', { method: 'manual' }))
  })

  it('alias active 展示「启用」tag，且显示「停用」动作（primary 不可直接停用）', async () => {
    mockGetDomainPageList.mockResolvedValue({ list: [activeAlias], total: 1 })
    render(<DomainList />)

    expect(await screen.findByText('alias.example.com')).toBeInTheDocument()
    expect(screen.getByText('启用')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /停用/ })).toBeInTheDocument()
  })

  it('新建域名提交 tenantId/domainRole/domainUsage/verificationMethod（后端 DomainCreateReq）', async () => {
    mockGetDomainPageList.mockResolvedValue({ list: [], total: 0 })
    mockCreateDomain.mockResolvedValue({ id: 'dn' })
    render(<DomainList />)

    fireEvent.click(await screen.findByRole('button', { name: /新建域名/ }))
    fireEvent.change(await screen.findByPlaceholderText('目标租户 ID（UUID）'), { target: { value: 't-1' } })
    fireEvent.change(screen.getByPlaceholderText('如：example.com'), { target: { value: 'new.example.com' } })
    fireEvent.click(screen.getByRole('button', { name: 'OK' }))

    await waitFor(() => expect(mockCreateDomain).toHaveBeenCalledTimes(1))
    const payload = mockCreateDomain.mock.calls[0][0] as Record<string, unknown>
    expect(payload).toMatchObject({
      tenantID: 't-1',
      domain: 'new.example.com',
      domainRole: 'primary',
      domainUsage: 'login',
      verificationMethod: 'manual',
    })
  })
})

describe('DomainStatusTag 七态映射', () => {
  it('7 态各显示对应中文', () => {
    const cases: [string, string][] = [
      ['draft', '草稿'],
      ['verified', '已验证'],
      ['tls_ready', '证书就绪'],
      ['active', '启用'],
      ['disabled', '停用'],
      ['retired', '已退役'],
      ['released', '已释放'],
    ]
    for (const [value, label] of cases) {
      const { unmount } = render(<DomainStatusTag value={value} />)
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })
})
