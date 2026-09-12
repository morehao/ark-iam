import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ApplicationItem } from '@ark-iam/types'

const mockGetApplicationPageList = vi.fn()
const mockGetApplicationDetail = vi.fn()
const mockGetMenuTree = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createMenu: vi.fn(),
  deleteMenu: vi.fn(),
  getApplicationDetail: (...args: unknown[]) => mockGetApplicationDetail(...args),
  getApplicationPageList: (...args: unknown[]) => mockGetApplicationPageList(...args),
  getMenuTree: (...args: unknown[]) => mockGetMenuTree(...args),
  updateMenu: vi.fn(),
}))

const MenuList = (await import('./index')).default

const apps: ApplicationItem[] = [
  {
    appID: 'app-1',
    code: 'console',
    name: '管理后台',
    description: '',
    logoUrl: '',
    homepageUrl: '',
    source: 'builtin',
    status: 'enable',
    sort: 1,
  },
  {
    appID: 'app-2',
    code: 'order',
    name: '订单中心',
    description: '',
    logoUrl: '',
    homepageUrl: '',
    source: 'third_party',
    status: 'enable',
    sort: 2,
  },
]

/**
 * 回归（2026-09）：应用可增长，菜单页的「所属应用」切换器不能只在已加载的前 100 条里本地过滤，
 * 必须按关键字走服务端搜索；切换应用后菜单树要按新的 appID 重新加载。
 */
describe('菜单页应用切换器', () => {
  beforeEach(() => {
    localStorage.clear()
    mockGetApplicationPageList.mockReset()
    mockGetMenuTree.mockReset()
    mockGetApplicationDetail.mockReset()
    mockGetApplicationDetail.mockResolvedValue(apps[1])
    mockGetMenuTree.mockResolvedValue({ list: [], total: 0 })
    mockGetApplicationPageList.mockImplementation(async (params?: { name?: string }) =>
      params?.name ? { list: [apps[1]], total: 1 } : { list: apps, total: apps.length },
    )
  })

  it('按名称服务端搜索，并按所选应用重新加载菜单树', async () => {
    render(<MenuList />)

    // 首屏默认选中应用列表第一个，并加载它的菜单树
    await waitFor(() => expect(mockGetMenuTree).toHaveBeenCalledWith('app-1'))

    // 输入关键字 → 必须带 name 请求服务端（而非在前 100 条里本地过滤）
    fireEvent.mouseDown(document.querySelector('.ant-select-selector') as HTMLElement)
    fireEvent.change(document.querySelector('[role="combobox"]') as HTMLElement, { target: { value: '订单' } })
    await waitFor(() =>
      expect(mockGetApplicationPageList).toHaveBeenCalledWith(expect.objectContaining({ name: '订单' })),
    )

    // 选中搜索结果 → 菜单树按新 appID 重新加载
    fireEvent.click(await screen.findByText('订单中心（order）'))
    await waitFor(() => expect(mockGetMenuTree).toHaveBeenCalledWith('app-2'))
  })

  it('上次选择的应用已不存在时回退到应用列表第一个', async () => {
    localStorage.setItem('ark-iam:menu:appID', 'app-deleted')
    mockGetApplicationDetail.mockRejectedValue(new Error('应用不存在'))

    render(<MenuList />)

    await waitFor(() => expect(mockGetApplicationDetail).toHaveBeenCalledWith('app-deleted'))
    // 回退到列表第一个；无效 ID 不应被用来拉菜单树
    await waitFor(() => expect(mockGetMenuTree).toHaveBeenCalledWith('app-1'))
    expect(mockGetMenuTree).not.toHaveBeenCalledWith('app-deleted')
  })
})
