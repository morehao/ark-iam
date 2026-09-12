import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const mockGetTenantApplicationPageList = vi.fn()
const mockGetTenantPageList = vi.fn()
const mockGetApplicationPageList = vi.fn()
const mockCreateTenantApplication = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createTenantApplication: (...args: unknown[]) => mockCreateTenantApplication(...args),
  deleteTenantApplication: vi.fn(),
  getApplicationPageList: (...args: unknown[]) => mockGetApplicationPageList(...args),
  getTenantApplicationPageList: (...args: unknown[]) => mockGetTenantApplicationPageList(...args),
  getTenantPageList: (...args: unknown[]) => mockGetTenantPageList(...args),
  updateTenantApplication: vi.fn(),
}))

const TenantApplicationList = (await import('./index')).default

describe('租户应用订阅', () => {
  beforeEach(() => {
    mockGetTenantApplicationPageList.mockReset().mockResolvedValue({ list: [], total: 0 })
    mockGetTenantPageList.mockReset().mockResolvedValue({ list: [{ tenantID: 't2', name: '租户二' }], total: 1 })
    mockGetApplicationPageList.mockReset().mockResolvedValue({
      list: [{ appID: 'app-1', name: '平台管理后台' }],
      total: 1,
    })
    mockCreateTenantApplication.mockReset().mockResolvedValue({ tenantAppID: 'ta-1' })
  })

  /**
   * 回归（2026-09 修复）：平台侧订阅必须显式指定归属租户，且租户/应用主键都是 UUID 字符串。
   * 原表单用 InputNumber 手输「应用ID」且无法指定租户——字符串 ID 录不进去，开通只能落到平台租户。
   * 改为双选择器后，提交必须带上所选 tenantID + appID。
   */
  it('新建订阅通过租户/应用选择器提交 tenantID + appID', async () => {
    render(<TenantApplicationList />)

    fireEvent.click(await screen.findByRole('button', { name: /新建订阅/ }))
    await waitFor(() => {
      expect(mockGetTenantPageList).toHaveBeenCalled()
      expect(mockGetApplicationPageList).toHaveBeenCalled()
    })

    const modal = document.querySelector('.ant-modal') as HTMLElement
    const selects = modal.querySelectorAll('.ant-select-selector')
    // 弹窗内下拉顺序：租户 → 应用 → 状态
    fireEvent.mouseDown(selects[0])
    fireEvent.click(await screen.findByText('租户二'))
    fireEvent.mouseDown(modal.querySelectorAll('.ant-select-selector')[1])
    fireEvent.click(await screen.findByText('平台管理后台'))
    fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    await waitFor(() =>
      expect(mockCreateTenantApplication).toHaveBeenCalledWith(
        expect.objectContaining({ tenantID: 't2', appID: 'app-1' }),
      ),
    )
  })

  /**
   * 回归（2026-09）：租户/应用都可能超过 100 条，下拉必须按关键字走服务端搜索。
   * 这里锁死「筛选区输入 → 带 name 参数请求后端」这条链路（后端 DTO 的查询参数名就是 name，
   * 拼错会静默退化成"永远只返回第一页"，用例可拦住）。
   */
  it('租户筛选按名称请求服务端搜索', async () => {
    render(<TenantApplicationList />)

    await waitFor(() => expect(mockGetTenantPageList).toHaveBeenCalled())

    const filterSelector = document.querySelectorAll('.ant-select-selector')[0]
    fireEvent.mouseDown(filterSelector as HTMLElement)
    fireEvent.change(document.querySelectorAll('[role="combobox"]')[0], { target: { value: '租' } })

    await waitFor(() =>
      expect(mockGetTenantPageList).toHaveBeenCalledWith(expect.objectContaining({ page: 1, name: '租' })),
    )
  })
})
