import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { App as AntdApp } from 'antd'
import { EnableTag, SuspendedTag, fmtTime } from '@ark-iam/ui'
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

  it('不再兼容历史的 isSuspended 布尔/0-1 字段：1/true/0/false 一律显示「正常」', () => {
    // P3 起 person/user 状态统一为 status 枚举；继续兼容 1/true 会掩盖读错字段的缺陷
    for (const v of [1, true, 0, false]) {
      const { unmount } = render(<SuspendedTag value={v} />)
      expect(screen.queryByText('挂起')).not.toBeInTheDocument()
      expect(screen.getByText('正常')).toBeInTheDocument()
      unmount()
    }
  })
})

describe('EnableTag 启用/停用映射', () => {
  it("只认 enable/disable：'enable' → 启用，'disable' → 停用", () => {
    const { rerender } = render(<EnableTag value="enable" />)
    expect(screen.getByText('启用')).toBeInTheDocument()

    rerender(<EnableTag value="disable" />)
    expect(screen.getByText('停用')).toBeInTheDocument()
  })

  it('不再兼容 active/inactive/1/0 等历史与数字取值，一律原样回显', () => {
    // 这正是本次收敛要消除的能力：把 active 错标成「启用」（应归 SuspendedTag 的「正常」）
    for (const v of ['active', 'inactive', 'enabled', '1', '0', '']) {
      const { unmount } = render(<EnableTag value={v} />)
      expect(screen.queryByText('启用')).not.toBeInTheDocument()
      expect(screen.queryByText('停用')).not.toBeInTheDocument()
      expect(screen.getByText(v || '-')).toBeInTheDocument()
      unmount()
    }
  })
})

/**
 * 平台自运营租户（种子租户 t_platform）不可挂起：它是平台控制台自身所在租户，
 * 挂起后整栈失联；后端 svctenant.Update 会拒写，前端必须同步置灰。
 */
describe('平台租户不可挂起', () => {
  it('编辑平台租户时挂起开关置灰', async () => {
    const platform: TenantItem = {
      tenantID: 't0',
      code: 't_platform',
      name: '平台运营中心',
      status: 'active',
      type: 'platform',
      tag: 'default',
      dbUser: 'default_user',
      createdAt,
      updatedAt,
    }
    mockGetTenantPageList.mockResolvedValue({ list: [platform], total: 1 })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )

    fireEvent.click(await screen.findByText('编辑'))
    const suspendSwitch = await screen.findByRole('switch')
    await waitFor(() => expect(suspendSwitch).toBeDisabled())
  })
})

/**
 * 平台自运营租户（种子租户 t_platform）不可删除：删除会让平台控制台整体失联且无恢复路径。
 * 前端按种子编码（而非 type）把「删除」置灰保留展示（不隐藏），
 * 后端 svctenant.Delete 以 TenantBuiltInDeleteForbiddenError 兜底。
 */
describe('平台租户不可删除', () => {
  const platformTenant: TenantItem = {
    tenantID: 't0',
    code: 't_platform',
    name: '平台运营中心',
    status: 'active',
    type: 'platform',
    tag: 'default',
    dbUser: 'default_user',
    createdAt,
    updatedAt,
  }

  it('平台租户行的「删除」置灰不可点', async () => {
    mockGetTenantPageList.mockResolvedValue({ list: [platformTenant], total: 1 })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )

    expect(await screen.findByText('平台运营中心')).toBeInTheDocument()
    // 3 个操作（编辑/重置管理员密码/删除）→ 删除收在「更多」下拉中，菜单项置灰
    fireEvent.click(screen.getByRole('button', { name: /更多/ }))
    const deleteItem = await screen.findByRole('menuitem', { name: '删除' })
    expect(deleteItem).toHaveClass('ant-dropdown-menu-item-disabled')
  })

  it('同 type=platform 但非种子编码的租户「删除」可点', async () => {
    const ordinary: TenantItem = { ...platformTenant, tenantID: 't2', code: 't_000000000002', name: 'Globex' }
    mockGetTenantPageList.mockResolvedValue({ list: [ordinary], total: 1 })

    render(
      <AntdApp>
        <TenantList />
      </AntdApp>,
    )
    expect(await screen.findByText('Globex')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /更多/ }))
    const deleteItem = await screen.findByRole('menuitem', { name: '删除' })
    expect(deleteItem).not.toHaveClass('ant-dropdown-menu-item-disabled')
  })
})
