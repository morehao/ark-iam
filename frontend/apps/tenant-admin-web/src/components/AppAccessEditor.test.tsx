import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { TenantUserApplicationItem } from '@ark-iam/types'

const mockGetTenantUserApplications = vi.fn()
const mockUpdateTenantUserApplications = vi.fn()

vi.mock('../api/user', () => ({
  getTenantUserApplications: (...args: unknown[]) => mockGetTenantUserApplications(...args),
  updateTenantUserApplications: (...args: unknown[]) => mockUpdateTenantUserApplications(...args),
}))

const AppAccessEditor = (await import('./AppAccessEditor')).default

/** 一行准入清单：订阅中 + 已准入 + 带 2 个角色（撤销需二次确认） */
const grantedWithRoles: TenantUserApplicationItem = {
  appID: 'app1',
  code: 'tenant_admin',
  name: '租户管理后台',
  appStatus: 'enable',
  subscribed: true,
  granted: true,
  status: 'enable',
  source: 'provision',
  roleCount: 2,
  updatedAt: 1,
}

/** 订阅中但未准入：可勾选、默认不选中 */
const subscribedNotGranted: TenantUserApplicationItem = {
  appID: 'app2',
  code: 'order',
  name: '订单中心',
  appStatus: 'enable',
  subscribed: true,
  granted: false,
  status: '',
  source: '',
  roleCount: 0,
  updatedAt: 1,
}

/** 订阅已停用但准入行仍在：不参与勾选，只作为「准入被订阅挡住」的提示 */
const unsubscribed: TenantUserApplicationItem = {
  appID: 'app3',
  code: 'legacy',
  name: '历史应用',
  appStatus: 'enable',
  subscribed: false,
  granted: true,
  status: 'enable',
  source: 'manual',
  roleCount: 0,
  updatedAt: 1,
}

/**
 * 回归（2026-09）：应用准入是「用户 × 应用」的准入层，独立于角色。
 * - 勾选集只含「订阅中且应用启用」的应用（未订阅者不出现，撤销行会保留）；
 * - 保存是**全量替换**（PUT appIDs），传空数组即清空全部准入；
 * - 取消勾选一个 roleCount>0 的应用必须二次确认（撤销会级联撤销该应用下的角色绑定）。
 */
describe('应用准入编辑器', () => {
  beforeEach(() => {
    mockGetTenantUserApplications
      .mockReset()
      .mockResolvedValue({ list: [grantedWithRoles, subscribedNotGranted, unsubscribed] })
    mockUpdateTenantUserApplications.mockReset().mockResolvedValue('')
  })

  it('默认勾选已准入应用，未准入的订阅应用可勾选，未订阅应用不出现', async () => {
    render(<AppAccessEditor userID="u-1" userName="张三" />)

    const grantedBox = (await screen.findByLabelText(/租户管理后台/)) as HTMLInputElement
    const subscribedBox = screen.getByLabelText(/订单中心/) as HTMLInputElement
    expect(grantedBox.checked).toBe(true)
    expect(subscribedBox.checked).toBe(false)
    // 未订阅的应用不参与勾选，仅以提示语说明其准入行被保留
    expect(screen.queryByLabelText(/历史应用/)).toBeNull()
    expect(screen.getByText(/1 个应用当前不可用/)).toBeInTheDocument()
  })

  it('勾选新应用后保存为全量替换（appIDs = 已选集合）', async () => {
    render(<AppAccessEditor userID="u-1" userName="张三" />)

    fireEvent.click(await screen.findByLabelText(/订单中心/))
    fireEvent.click(screen.getByRole('button', { name: '保存准入' }))

    await waitFor(() => expect(mockUpdateTenantUserApplications).toHaveBeenCalledTimes(1))
    const [userID, appIDs] = mockUpdateTenantUserApplications.mock.calls[0] as [string, string[]]
    expect(userID).toBe('u-1')
    // app3（订阅已停用但准入行仍在）不在 UI 上出现，但全量替换必须原样带回：
    // 否则"保存一次"就等于顺手撤销了它，重新订阅后访问不会恢复（不变式 I3）
    expect([...appIDs].sort()).toEqual(['app1', 'app2', 'app3'])
  })

  it('取消勾选带角色的应用先弹二次确认，确认后才提交', async () => {
    render(<AppAccessEditor userID="u-1" userName="张三" />)

    fireEvent.click(await screen.findByLabelText(/租户管理后台/))
    fireEvent.click(screen.getByRole('button', { name: '保存准入' }))

    // 二次确认：文案必须点明"将同时撤销 N 个角色"（antd 会在标题与 aria 节点各渲染一次）
    expect((await screen.findAllByText('撤销应用准入将同时撤销角色')).length).toBeGreaterThan(0)
    expect(screen.getByText(/撤销其在该应用下的角色绑定/)).toBeInTheDocument()
    expect(screen.getByText(/租户管理后台（2 个角色）/)).toBeInTheDocument()
    expect(mockUpdateTenantUserApplications).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: '确认撤销' }))
    // 取消 app1 后勾选集为空；app2 未被勾选、app3 是保留的订阅失效准入行
    await waitFor(() => expect(mockUpdateTenantUserApplications).toHaveBeenCalledWith('u-1', ['app3']))
  })

  it('取消勾选不带角色的应用无需二次确认，直接提交', async () => {
    mockGetTenantUserApplications.mockResolvedValue({ list: [{ ...grantedWithRoles, roleCount: 0 }] })
    render(<AppAccessEditor userID="u-1" userName="张三" />)

    fireEvent.click(await screen.findByLabelText(/租户管理后台/))
    fireEvent.click(screen.getByRole('button', { name: '保存准入' }))

    await waitFor(() => expect(mockUpdateTenantUserApplications).toHaveBeenCalledWith('u-1', []))
    expect(screen.queryAllByText('撤销应用准入将同时撤销角色')).toHaveLength(0)
  })
})
