import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { fmtTime } from '@ark-iam/ui'
import type { ApplicationItem } from '@ark-iam/types'

const mockGetApplicationPageList = vi.fn()
vi.mock('@ark-iam/api', () => ({
  createApplication: vi.fn(),
  deleteApplication: vi.fn(),
  getApplicationDetail: vi.fn(),
  getApplicationPageList: (...args: unknown[]) => mockGetApplicationPageList(...args),
  updateApplication: vi.fn(),
}))

const ApplicationList = (await import('./index')).default

const createdAt = 1789142400
const updatedAt = 1789228800

const applications: ApplicationItem[] = [
  {
    appID: 'app-1',
    code: 'platform-admin',
    name: '管理后台',
    description: '',
    logoUrl: '',
    homepageUrl: '',
    type: 'first_party',
    status: 'enable',
    sort: 0,
    createdAt,
    updatedAt,
  },
]

describe('应用列表', () => {
  beforeEach(() => {
    mockGetApplicationPageList.mockReset()
  })

  it('同时展示创建时间与更新时间两列', async () => {
    mockGetApplicationPageList.mockResolvedValue({ list: applications, total: applications.length })

    render(<ApplicationList />)

    expect(await screen.findByText('管理后台')).toBeInTheDocument()
    // 时间列读 createdAt/updatedAt（回归：字段缺失会渲染 '-'）
    expect(screen.getByText(fmtTime(createdAt))).toBeInTheDocument()
    expect(screen.getByText(fmtTime(updatedAt))).toBeInTheDocument()
  })
})
