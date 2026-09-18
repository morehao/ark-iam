import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { ApplicationItem } from '@ark-iam/types'

const mockGetApplicationPageList = vi.fn()
const mockGetApplicationDetail = vi.fn()
const mockGetMenuTree = vi.fn()
const mockCreateMenu = vi.fn()
const mockUpdateMenu = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createMenu: (...args: unknown[]) => mockCreateMenu(...args),
  deleteMenu: vi.fn(),
  getApplicationDetail: (...args: unknown[]) => mockGetApplicationDetail(...args),
  getApplicationPageList: (...args: unknown[]) => mockGetApplicationPageList(...args),
  getMenuTree: (...args: unknown[]) => mockGetMenuTree(...args),
  updateMenu: (...args: unknown[]) => mockUpdateMenu(...args),
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

  it('上次选择的应用已不存在时回退到应用列表第一个，且不弹全局错误提示（silent）', async () => {
    localStorage.setItem('ark-iam:menu:appID', 'app-deleted')
    mockGetApplicationDetail.mockRejectedValue(new Error('应用不存在'))

    render(<MenuList />)

    // 首屏恢复属 best-effort 探测：必须带 silent，失败由页面自行回退（见 packages/api request.ts）
    await waitFor(() => expect(mockGetApplicationDetail).toHaveBeenCalledWith('app-deleted', { silent: true }))
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
          hidden: 'disable',
          externalLink: 'disable',
          keepAlive: 'disable',
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

/**
 * D1 回归（P3 枚举契约）：menu.hidden / external_link / keep_alive 已从 boolean/0-1
 * 改为字符串枚举 enable/disable，表单必须提交 'enable'/'disable'，不得再出现 1/0。
 * 回归背景：此前用 getValueFromEvent={(c) => (c ? 1 : 0)} 把 Switch 的 boolean 转成数字提交。
 */
describe('菜单开关提交字符串枚举', () => {
  const menuRow = {
    menuID: 'm1',
    appID: 'app-1',
    parentID: '',
    name: '工作台',
    code: 'dashboard',
    path: '/dashboard',
    icon: 'dashboard',
    sort: 1,
    type: 'menu' as const,
    visibility: 'member' as const,
    component: '/dashboard/index',
    redirect: '',
    hidden: 'enable' as const,
    externalLink: 'disable' as const,
    keepAlive: 'enable' as const,
    status: 'enable' as const,
  }

  beforeEach(() => {
    mockGetApplicationPageList.mockResolvedValue({ list: apps, total: apps.length })
    mockGetApplicationDetail.mockResolvedValue(apps[0])
    mockGetMenuTree.mockResolvedValue({ list: [menuRow], total: 1 })
    mockCreateMenu.mockReset()
    mockUpdateMenu.mockReset()
  })

  it('新建菜单：切换开关后提交（含新建时的三个开关）均须为 enable/disable 字符串', async () => {
    render(<MenuList />)

    const createButton = await screen.findByRole('button', { name: /新建根菜单/ })
    await waitFor(() => expect(createButton).toBeEnabled())
    fireEvent.click(createButton)

    fireEvent.change(await screen.findByPlaceholderText('菜单显示名称'), { target: { value: '报表中心' } })
    fireEvent.change(screen.getByPlaceholderText('唯一编码，如 user:list'), { target: { value: 'report:list' } })

    const modal = document.querySelector('.ant-modal') as HTMLElement
    const switches = within(modal).getAllByRole('switch')
    expect(switches).toHaveLength(3)
    // 三个开关默认关闭 → 逐个打开
    switches.forEach((s) => fireEvent.click(s))

    fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    await waitFor(() => expect(mockCreateMenu).toHaveBeenCalledTimes(1))
    const payload = mockCreateMenu.mock.calls[0][0] as Record<string, unknown>
    expect(payload.hidden).toBe('enable')
    expect(payload.externalLink).toBe('enable')
    expect(payload.keepAlive).toBe('enable')
  })

  it('编辑菜单：按枚举回显开关（enable→开、disable→关），切换后提交仍为字符串枚举', async () => {
    render(<MenuList />)

    fireEvent.click(await screen.findByText('编辑'))
    await screen.findByPlaceholderText('菜单显示名称')

    const modal = document.querySelector('.ant-modal') as HTMLElement
    const switches = within(modal).getAllByRole('switch')
    // 回显：hidden=enable → 开；externalLink=disable → 关；keepAlive=enable → 开
    expect(switches[0]).toHaveAttribute('aria-checked', 'true')
    expect(switches[1]).toHaveAttribute('aria-checked', 'false')
    expect(switches[2]).toHaveAttribute('aria-checked', 'true')

    // 关闭「隐藏」后提交：三个开关都必须是字符串枚举
    fireEvent.click(switches[0])
    fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    await waitFor(() => expect(mockUpdateMenu).toHaveBeenCalledTimes(1))
    const payload = mockUpdateMenu.mock.calls[0][0] as Record<string, unknown>
    expect(payload.hidden).toBe('disable')
    expect(payload.externalLink).toBe('disable')
    expect(payload.keepAlive).toBe('enable')
  })
})
