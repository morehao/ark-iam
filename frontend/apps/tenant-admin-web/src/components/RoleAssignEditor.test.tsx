import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { TenantRoleItem, TenantUserRoleItem } from '@ark-iam/types'

const mockGetMachineUserRoles = vi.fn()
const mockUpdateMachineUserRoles = vi.fn()
const mockGetTenantApps = vi.fn()
const mockGetTenantRolePageList = vi.fn()
const mockGetTenantUserRoles = vi.fn()
const mockUpdateTenantUserRoles = vi.fn()

vi.mock('../api/machineUser', () => ({
  getMachineUserRoles: (...args: unknown[]) => mockGetMachineUserRoles(...args),
  updateMachineUserRoles: (...args: unknown[]) => mockUpdateMachineUserRoles(...args),
}))
vi.mock('../api/menu', () => ({
  getTenantApps: (...args: unknown[]) => mockGetTenantApps(...args),
}))
vi.mock('../api/role', () => ({
  getTenantRolePageList: (...args: unknown[]) => mockGetTenantRolePageList(...args),
}))
vi.mock('../api/user', () => ({
  getTenantUserRoles: (...args: unknown[]) => mockGetTenantUserRoles(...args),
  updateTenantUserRoles: (...args: unknown[]) => mockUpdateTenantUserRoles(...args),
}))

const RoleAssignEditor = (await import('./RoleAssignEditor')).default

const assignedRoles: TenantUserRoleItem[] = [
  { roleID: 'r1', appID: 'app1', appName: '管理后台', name: '租户管理员', description: '' },
]

const pageRoles: TenantRoleItem[] = [
  { roleID: 'r1', appID: 'app1', appName: '管理后台', name: '租户管理员', description: '', adminType: 'normal', memberCount: 0, menuCount: 0 },
  { roleID: 'r2', appID: 'app1', appName: '管理后台', name: '只读成员', description: '', adminType: 'normal', memberCount: 0, menuCount: 0 },
]

const searchedRole: TenantRoleItem = {
  roleID: 'r9',
  appID: 'app1',
  appName: '管理后台',
  name: '运维管理员',
  description: '',
  adminType: 'normal',
  memberCount: 0,
  menuCount: 0,
}

/**
 * 回归（2026-09）：角色可增长，按应用授权的角色多选框不能只取前 100 条在前端过滤，
 * 必须按名称走服务端搜索（后端查询参数名是 keyword），且已分配角色要回显名称。
 */
describe('按应用授权编辑器', () => {
  beforeEach(() => {
    mockGetMachineUserRoles.mockReset().mockResolvedValue({ list: assignedRoles })
    mockGetTenantUserRoles.mockReset().mockResolvedValue({ list: assignedRoles })
    mockUpdateMachineUserRoles.mockReset()
    mockUpdateTenantUserRoles.mockReset()
    mockGetTenantApps.mockReset().mockResolvedValue({
      list: [
        { appID: 'app1', code: 'console', name: '管理后台' },
        { appID: 'app2', code: 'order', name: '订单中心' },
      ],
    })
    mockGetTenantRolePageList.mockReset().mockImplementation(async (params?: { keyword?: string }) =>
      params?.keyword ? { list: [searchedRole], total: 1 } : { list: pageRoles, total: pageRoles.length },
    )
  })

  it('角色按名称服务端搜索，并回显已分配角色名称', async () => {
    render(<RoleAssignEditor kind="machine" subjectID="mu-1" />)

    // 已分配角色按名称回显（而不是裸 roleID）
    expect(await screen.findByText('租户管理员')).toBeInTheDocument()

    // 角色多选框（第 2 个下拉）输入关键字 → 带 keyword + 当前 appID 请求服务端
    fireEvent.mouseDown(document.querySelectorAll('.ant-select-selector')[1] as HTMLElement)
    fireEvent.change(document.querySelectorAll('[role="combobox"]')[1] as HTMLElement, {
      target: { value: '运维' },
    })

    await waitFor(() =>
      expect(mockGetTenantRolePageList).toHaveBeenCalledWith(
        expect.objectContaining({ appID: 'app1', keyword: '运维' }),
      ),
    )
    expect(await screen.findByText('运维管理员')).toBeInTheDocument()
  })

  it('切换到系统角色组时用 unassigned 开关请求服务端', async () => {
    mockGetMachineUserRoles.mockResolvedValue({
      list: [{ roleID: 'r0', appID: '', appName: '', name: '系统角色', description: '' }],
    })

    render(<RoleAssignEditor kind="machine" subjectID="mu-1" />)

    // 默认选中「当前持有角色的应用」＝系统角色组（appID 空串）
    await waitFor(() =>
      expect(mockGetTenantRolePageList).toHaveBeenCalledWith(expect.objectContaining({ unassigned: true })),
    )
  })
})
