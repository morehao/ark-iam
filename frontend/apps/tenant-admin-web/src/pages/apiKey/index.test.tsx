import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, waitFor } from '@testing-library/react'
import type { TenantMachineUserItem } from '@ark-iam/types'

const mockGetApiKeyPageList = vi.fn()
const mockGetMachineUserPageList = vi.fn()

vi.mock('../../api/apiKey', () => ({
  createApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
  getApiKeyPageList: (...args: unknown[]) => mockGetApiKeyPageList(...args),
  revokeApiKey: vi.fn(),
}))
vi.mock('../../api/machineUser', () => ({
  getMachineUserPageList: (...args: unknown[]) => mockGetMachineUserPageList(...args),
}))

const ApiKeyPage = (await import('./index')).default

const machineUsers: TenantMachineUserItem[] = [
  {
    machineUserID: 'mu-1',
    tenantID: 't1',
    name: 'CI 构建账号',
    description: '',
    primaryDepartmentID: 'd1',
    primaryDepartmentName: '平台组',
    isSuspended: false,
  },
  {
    machineUserID: 'mu-2',
    tenantID: 't1',
    name: '监控采集账号',
    description: '',
    primaryDepartmentID: 'd1',
    primaryDepartmentName: '平台组',
    isSuspended: false,
  },
]

/**
 * 回归（2026-09）：服务账号可增长，密钥页「归属服务账号」下拉不能只取前 100 条在前端过滤，
 * 必须按名称走服务端搜索（后端查询参数名是 name）。
 */
describe('API 密钥页服务账号下拉', () => {
  beforeEach(() => {
    mockGetApiKeyPageList.mockReset().mockResolvedValue({ list: [], total: 0 })
    mockGetMachineUserPageList.mockReset().mockImplementation(async (params?: { name?: string }) =>
      params?.name ? { list: [machineUsers[1]], total: 1 } : { list: machineUsers, total: machineUsers.length },
    )
  })

  it('按名称服务端搜索服务账号', async () => {
    render(<ApiKeyPage />)

    await waitFor(() => expect(mockGetMachineUserPageList).toHaveBeenCalled())

    fireEvent.mouseDown(document.querySelector('.ant-select-selector') as HTMLElement)
    fireEvent.change(document.querySelector('[role="combobox"]') as HTMLElement, { target: { value: '监控' } })

    await waitFor(() =>
      expect(mockGetMachineUserPageList).toHaveBeenCalledWith(expect.objectContaining({ name: '监控' })),
    )
  })
})
