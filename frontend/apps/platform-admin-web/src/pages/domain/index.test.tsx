import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { VerifiedTag } from '@ark-iam/ui'
import type { DomainItem } from '@ark-iam/types'

const mockGetDomainPageList = vi.fn()
const mockGetDomainDetail = vi.fn()
const mockUpdateDomain = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createDomain: vi.fn(),
  deleteDomain: vi.fn(),
  getDomainDetail: (...args: unknown[]) => mockGetDomainDetail(...args),
  getDomainPageList: (...args: unknown[]) => mockGetDomainPageList(...args),
  updateDomain: (...args: unknown[]) => mockUpdateDomain(...args),
}))

const DomainList = (await import('./index')).default

const domain: DomainItem = {
  id: 'd1',
  domain: 'example.com',
  verificationStatus: 'unverified',
  createdAt: 1700000000,
  updatedAt: 1700000000,
}

/**
 * D2 回归（P3 枚举契约）：domain.verification_status 已从 boolean/0-1 改为字符串枚举
 * unverified/verified，且 domain.verified_at 已从后端删除（前端「验证时间」列同批删除）。
 */
describe('域名页验证状态枚举', () => {
  beforeEach(() => {
    mockGetDomainPageList.mockReset().mockResolvedValue({ list: [domain], total: 1 })
    mockGetDomainDetail.mockReset().mockResolvedValue({ ...domain, verificationStatus: 'verified' })
    mockUpdateDomain.mockReset().mockResolvedValue(undefined)
  })

  it('编辑态按 verificationStatus 回显并提交 verified 字符串枚举', async () => {
    render(<DomainList />)

    fireEvent.click(await screen.findByText('编辑'))
    // 详情接口回传 verified：提交时不得再提交 isVerified 数字
    await waitFor(() => expect(mockGetDomainDetail).toHaveBeenCalledWith('d1'))

    const modal = document.querySelector('.ant-modal') as HTMLElement
    fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    await waitFor(() => expect(mockUpdateDomain).toHaveBeenCalledTimes(1))
    const payload = mockUpdateDomain.mock.calls[0][0] as Record<string, unknown>
    expect(payload.id).toBe('d1')
    expect(payload.verificationStatus).toBe('verified')
    expect('isVerified' in payload).toBe(false)
  })

  it('验证状态下拉选项取值为 unverified/verified，选择未验证后提交字符串枚举', async () => {
    render(<DomainList />)

    fireEvent.click(await screen.findByText('编辑'))
    await waitFor(() => expect(mockGetDomainDetail).toHaveBeenCalledWith('d1'))

    // 打开弹窗内的验证状态 Select（页面上还有分页 sizeChanger 的 Select，必须限定在弹窗内），
    // 选「未验证」（当前回显为 verified，故该选项只存在于下拉里）
    const modal = document.querySelector('.ant-modal') as HTMLElement
    fireEvent.mouseDown(modal.querySelector('.ant-select-selector') as HTMLElement)
    fireEvent.click(await screen.findByTitle('未验证'))

    fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    await waitFor(() => expect(mockUpdateDomain).toHaveBeenCalledTimes(1))
    const payload = mockUpdateDomain.mock.calls[0][0] as Record<string, unknown>
    expect(payload.verificationStatus).toBe('unverified')
  })

  it('不再展示「验证时间」列（verifiedAt 已下线）', async () => {
    render(<DomainList />)

    await screen.findByText('example.com')
    expect(screen.queryByText('验证时间')).not.toBeInTheDocument()
  })
})

describe('VerifiedTag 验证状态映射', () => {
  it("只认枚举：'verified' → 已验证，'unverified' → 未验证", () => {
    const { rerender } = render(<VerifiedTag value="verified" />)
    expect(screen.getByText('已验证')).toBeInTheDocument()

    rerender(<VerifiedTag value="unverified" />)
    expect(screen.getByText('未验证')).toBeInTheDocument()
  })

  it('不再兼容 1/0 数字取值：1 不再显示「已验证」', () => {
    render(<VerifiedTag value={1} />)
    expect(screen.queryByText('已验证')).not.toBeInTheDocument()
    expect(screen.getByText('未验证')).toBeInTheDocument()
  })
})
