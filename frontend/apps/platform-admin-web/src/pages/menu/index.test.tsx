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
    name: '平台管理后台',
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

/**
 * 内置应用（source=builtin）的菜单同样支持新增/删除：新增行不带种子身份键（seed_key 为空），
 * 种子不会认领；删除内置菜单会留下软删"墓碑"，种子下次启动不再复活它。
 * 回归背景：此前内置应用整棵树禁止增删，「功能扩展/调整」必须改代码发版。
 */
describe('内置应用的菜单支持增删', () => {
  it('选中内置应用时可新建根菜单，行内提供新增子级/删除，编辑弹窗内字段（含编码）均可改', async () => {
    mockGetApplicationPageList.mockResolvedValue({ list: apps, total: apps.length })
    mockGetApplicationDetail.mockResolvedValue(apps[0])
    mockGetMenuTree.mockResolvedValue({
      list: [
        {
          menuID: 'm1',
          appID: 'app-1',
          parentID: '',
          name: '工作台',
          code: 'dashboard',
          path: '/dashboard',
          icon: 'dashboard',
          sort: 1,
          type: 'menu',
          visibility: 'member',
          component: '/dashboard/index',
          redirect: '',
          hidden: 0,
          externalLink: 0,
          keepAlive: 0,
          status: 'enable',
        },
      ],
      total: 1,
    })

    render(<MenuList />)

    // 新建根菜单不再被内置应用禁用
    const createButton = await screen.findByRole('button', { name: /新建根菜单/ })
    await waitFor(() => expect(createButton).toBeEnabled())

    // 行内操作齐备：新增子级 / 编辑 / 删除
    expect(await screen.findByText('新增子级')).toBeInTheDocument()
    expect(await screen.findByText('删除')).toBeInTheDocument()

    // 打开编辑弹窗：名称/编码/路径/图标全部可改（种子按 seed_key 认行，改名不重建）
    fireEvent.click(await screen.findByText('编辑'))
    const nameInput = await screen.findByPlaceholderText('菜单显示名称')
    await waitFor(() => expect(nameInput).not.toBeDisabled())
    expect(screen.getByPlaceholderText('唯一编码，如 user:list')).not.toBeDisabled()
    expect(screen.getByPlaceholderText('如 /user/list')).not.toBeDisabled()
    expect(screen.getByPlaceholderText('如 UserOutlined')).not.toBeDisabled()
    // 不再出现「内置应用不可新增/删除」的提示
    expect(screen.queryByText(/不可新增\/删除/)).not.toBeInTheDocument()
  })
})
