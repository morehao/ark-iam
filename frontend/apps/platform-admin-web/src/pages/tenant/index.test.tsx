import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { App as AntdApp } from 'antd'
import { SuspendedTag, fmtTime } from '@ark-iam/ui'
import type { TenantItem } from '@ark-iam/types'

const mockGetTenantPageList = vi.fn()
const mockCreateTenant = vi.fn()
const mockResetTenantAdminPassword = vi.fn()
vi.mock('@ark-iam/api', () => ({
  createTenant: (...args: unknown[]) => mockCreateTenant(...args),
  deleteTenant: vi.fn(),
  getTenantPageList: (...args: unknown[]) => mockGetTenantPageList(...args),
  resetTenantAdminPassword: (...args: unknown[]) => mockResetTenantAdminPassword(...args),
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
    mockCreateTenant.mockReset()
    mockResetTenantAdminPassword.mockReset()
  })

  it('按 status 枚举渲染状态列，并同时展示创建时间与更新时间', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: tenants, total: tenants.length })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )

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

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )
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
  it('新建租户必填管理员，且初始临时密码只在弹窗中展示一次', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: tenants, total: tenants.length })
    mockCreateTenant.mockResolvedValue({ tenantID: 't9', adminUserID: 'u9', adminInitialPassword: 'Temp1234' })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )
    await screen.findByText('Acme Corp')

    fireEvent.click(screen.getByRole('button', { name: /新建租户/ }))
    fireEvent.change(await screen.findByPlaceholderText('租户名称'), { target: { value: 'New Co' } })
    fireEvent.change(screen.getByPlaceholderText('管理员姓名'), { target: { value: '张三' } })
    fireEvent.change(screen.getByPlaceholderText('用于登录与密码交接'), { target: { value: 'admin@new.co' } })
    fireEvent.click(screen.getByRole('button', { name: 'OK' }))

    // 管理员信息随建租户一并提交（不含密码：密码由服务端生成）
    await waitFor(() => {
      expect(mockCreateTenant).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'New Co',
          admin: expect.objectContaining({ name: '张三', primaryEmail: 'admin@new.co' }),
        }),
      )
    })
    expect(mockCreateTenant.mock.calls[0][0]).not.toHaveProperty('password')
    expect(mockCreateTenant.mock.calls[0][0].admin).not.toHaveProperty('password')

    // 初始临时密码仅在弹窗中一次性展示
    expect(await screen.findByText('Temp1234')).toBeInTheDocument()
    expect(screen.getByText('请立即保存，关闭后不再显示')).toBeInTheDocument()
  })

  it('重置内置管理员密码：仅展示一次新临时密码，不弹出密码输入框', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: tenants, total: tenants.length })
    mockResetTenantAdminPassword.mockResolvedValue({ userID: 'u1', initialPassword: 'Reset5678' })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )
    await screen.findByText('Acme Corp')

    // 「重置管理员密码」是次要危险操作，收在「更多」下拉里（操作列不因长文案被撑宽）
    fireEvent.click(screen.getAllByRole('button', { name: /更多/ })[0])
    fireEvent.click(await screen.findByText('重置管理员密码'))
    // Modal.confirm 的确认按钮（测试环境未注入中文 locale，按钮文案为 OK）
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockResetTenantAdminPassword).toHaveBeenCalledWith('t1')
    })
    expect(await screen.findByText('Reset5678')).toBeInTheDocument()
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
