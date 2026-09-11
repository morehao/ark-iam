import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { SuspendedTag, fmtTime } from '@ark-iam/ui'
import type { TenantItem } from '@ark-iam/types'

const mockGetTenantPageList = vi.fn()
vi.mock('@ark-iam/api', () => ({
  createTenant: vi.fn(),
  deleteTenant: vi.fn(),
  getTenantPageList: (...args: unknown[]) => mockGetTenantPageList(...args),
  updateTenant: vi.fn(),
}))

const TenantList = (await import('./index')).default

const createdAt = 1789142400
const updatedAt = 1789228800

const tenants: TenantItem[] = [
  {
    tenantID: 't1',
    code: 't_3f7a9c1d2e4b',
    name: 'Acme Corp',
    status: 'active',
    type: 'customer',
    tag: 'default',
    dbUser: 'acme',
    createdAt,
    updatedAt,
  },
  {
    tenantID: 't2',
    code: 't_000000000002',
    name: 'Globex',
    status: 'suspended',
    type: 'platform',
    tag: 'default',
    dbUser: 'globex',
    createdAt,
    updatedAt,
  },
]

describe('租户列表', () => {
  beforeEach(() => {
    mockGetTenantPageList.mockReset()
  })

  it('按 status 枚举渲染状态列，并同时展示创建时间与更新时间', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: tenants, total: tenants.length })

    render(<TenantList />)

    expect(await screen.findByText('Acme Corp')).toBeInTheDocument()
    expect(screen.getByText('Globex')).toBeInTheDocument()

    // 状态列读 status：挂起租户必须显示「挂起」（回归：读废弃的 isSuspended 会恒显示「正常」）
    expect(screen.getByText('挂起')).toBeInTheDocument()
    expect(screen.getByText('正常')).toBeInTheDocument()

    // 创建时间/更新时间两列都必须渲染秒级时间戳（回归：字段缺失会渲染 '-'）
    expect(screen.getAllByText(fmtTime(createdAt))).toHaveLength(2)
    expect(screen.getAllByText(fmtTime(updatedAt))).toHaveLength(2)

    // 编码列展示服务端生成值
    expect(screen.getByText('t_3f7a9c1d2e4b')).toBeInTheDocument()
  })

  it('状态筛选：选择「挂起」后带 status=suspended 重新请求，并回到第 1 页', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: tenants, total: tenants.length })

    render(<TenantList />)
    await screen.findByText('Acme Corp')

    // 默认不筛选：请求不携带 status
    expect(mockGetTenantPageList).toHaveBeenCalledWith(expect.objectContaining({ page: 1, status: undefined }))

    // 页面上还有分页器的下拉，这里按占位文案定位状态筛选下拉
    const statusSelector = screen.getByText('全部状态').closest('.ant-select-selector')
    fireEvent.mouseDown(statusSelector as HTMLElement)
    fireEvent.click(await screen.findByTitle('挂起'))

    await waitFor(() => {
      expect(mockGetTenantPageList).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 1, status: 'suspended' }),
      )
    })
  })
})

describe('SuspendedTag 租户状态映射', () => {
  it("status 枚举：'suspended' → 挂起，'active' → 正常", () => {
    const { rerender } = render(<SuspendedTag value="suspended" />)
    expect(screen.getByText('挂起')).toBeInTheDocument()

    rerender(<SuspendedTag value="active" />)
    expect(screen.getByText('正常')).toBeInTheDocument()
  })

  it('兼容历史的 isSuspended 布尔/0-1 字段（用户列表等仍在用）', () => {
    const { rerender } = render(<SuspendedTag value={1} />)
    expect(screen.getByText('挂起')).toBeInTheDocument()

    rerender(<SuspendedTag value={false} />)
    expect(screen.getByText('正常')).toBeInTheDocument()
  })
})
