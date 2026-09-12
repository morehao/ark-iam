import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { TenantApiKeyItem, TenantMachineUserDetail, TenantMachineUserItem } from '@ark-iam/types'

const mockGetTenantUserPageList = vi.fn()
const mockGetMachineUserPageList = vi.fn()
const mockGetMachineUserDetail = vi.fn()
const mockGetApiKeyPageList = vi.fn()

vi.mock('../../api/user', () => ({
  createTenantUser: vi.fn(),
  createTenantUserIdentity: vi.fn(),
  deleteTenantUserIdentity: vi.fn(),
  getTenantUserDetail: vi.fn(),
  getTenantUserIdentities: vi.fn(),
  getTenantUserLoginLogs: vi.fn(),
  getTenantUserPageList: (...args: unknown[]) => mockGetTenantUserPageList(...args),
  getTenantUserRoles: vi.fn().mockResolvedValue({ list: [] }),
  resetTenantUserPassword: vi.fn(),
  updateTenantUser: vi.fn(),
  updateTenantUserRoles: vi.fn(),
}))
vi.mock('../../api/machineUser', () => ({
  createMachineUser: vi.fn(),
  deleteMachineUser: vi.fn(),
  getMachineUserDetail: (...args: unknown[]) => mockGetMachineUserDetail(...args),
  getMachineUserPageList: (...args: unknown[]) => mockGetMachineUserPageList(...args),
  getMachineUserRoles: vi.fn().mockResolvedValue({ list: [] }),
  updateMachineUser: vi.fn(),
  updateMachineUserRoles: vi.fn(),
  updateMachineUserStatus: vi.fn(),
}))
vi.mock('../../api/apiKey', () => ({
  createApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
  getApiKeyPageList: (...args: unknown[]) => mockGetApiKeyPageList(...args),
  revokeApiKey: vi.fn(),
}))
vi.mock('../../api/department', () => ({ getDepartmentTree: vi.fn().mockResolvedValue({ list: [] }) }))
vi.mock('../../api/menu', () => ({ getTenantApps: vi.fn().mockResolvedValue({ list: [] }) }))
vi.mock('../../api/role', () => ({ getTenantRolePageList: vi.fn().mockResolvedValue({ list: [], total: 0 }) }))

const TenantUserPage = (await import('./index')).default

const machineUser: TenantMachineUserItem = {
  machineUserID: 'mu-1',
  tenantID: 't1',
  name: 'CI 构建账号',
  description: '',
  primaryDepartmentID: 'd1',
  primaryDepartmentName: '平台组',
  isSuspended: false,
}

const machineDetail: TenantMachineUserDetail = {
  ...machineUser,
  departments: [{ departmentID: 'd1', departmentName: '平台组', relationType: 'primary' }],
  roles: [],
}

// 服务端共 57 条，抽屉只取最近一页 50 条
const keys: TenantApiKeyItem[] = Array.from({ length: 50 }, (_, i) => ({
  keyID: `k${i}`,
  name: `构建凭证 ${i}`,
  keyPrefix: 'ak_',
  ownerUserID: 'mu-1',
  ownerType: 'machine',
  ownerName: 'CI 构建账号',
  createdBy: 'op',
  creatorName: '运维',
  expiredAt: null,
  lastUsedAt: null,
  revokedAt: null,
  createdAt: 1700000000,
  updatedAt: 1700000000,
}))

/**
 * 回归（2026-09）：详情抽屉的密钥表只取最近一页（pageSize 50、无分页），
 * 截断必须显式告知用户（共 N 条 / 仅展示最近 50 条），不能静默少显示。
 */
describe('服务账号详情 · API 密钥速览', () => {
  beforeEach(() => {
    mockGetTenantUserPageList.mockReset().mockResolvedValue({ list: [], total: 0 })
    mockGetMachineUserPageList.mockReset().mockResolvedValue({ list: [machineUser], total: 1 })
    mockGetMachineUserDetail.mockReset().mockResolvedValue(machineDetail)
    mockGetApiKeyPageList.mockReset().mockResolvedValue({ list: keys, total: 57 })
  })

  it('密钥数超过一页时明示总数与截断', async () => {
    render(<TenantUserPage />)

    fireEvent.click(screen.getByRole('tab', { name: '服务账号' }))
    fireEvent.click(await screen.findByText('CI 构建账号'))

    await waitFor(() => expect(mockGetMachineUserDetail).toHaveBeenCalledWith('mu-1'))
    fireEvent.click(screen.getByRole('tab', { name: 'API 密钥' }))

    expect(await screen.findByText(/共 57 条/)).toBeInTheDocument()
    expect(screen.getByText(/仅展示最近 50 条/)).toBeInTheDocument()
  })

  it('密钥数未超过一页时不提示截断', async () => {
    mockGetApiKeyPageList.mockResolvedValue({ list: keys.slice(0, 3), total: 3 })

    render(<TenantUserPage />)

    fireEvent.click(screen.getByRole('tab', { name: '服务账号' }))
    fireEvent.click(await screen.findByText('CI 构建账号'))
    await waitFor(() => expect(mockGetMachineUserDetail).toHaveBeenCalledWith('mu-1'))
    fireEvent.click(screen.getByRole('tab', { name: 'API 密钥' }))

    expect(await screen.findByText(/共 3 条/)).toBeInTheDocument()
    expect(screen.queryByText(/仅展示最近/)).toBeNull()
  })
})
