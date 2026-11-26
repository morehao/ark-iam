import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { TenantSubscribedApplicationItem } from '@ark-iam/types'

const mockGetTenantApps = vi.fn()
const mockGetAppMenuTree = vi.fn()
const mockCreateAppMenu = vi.fn()
const mockUpdateAppMenu = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createAppMenu: (...args: unknown[]) => mockCreateAppMenu(...args),
  deleteAppMenu: vi.fn(),
  getAppMenuTree: (...args: unknown[]) => mockGetAppMenuTree(...args),
  getOwnedApps: (...args: unknown[]) => mockGetTenantApps(...args),
  updateAppMenu: (...args: unknown[]) => mockUpdateAppMenu(...args),
}))

const MenuList = (await import('./index')).default

const apps: TenantSubscribedApplicationItem[] = [
  { appID: 'app-1', code: 'console', name: '平台管理后台' },
  { appID: 'app-2', code: 'order', name: '订单中心' },
]

/**
 * 菜单页数据来源迁移到租户侧（v2）：应用选择器来自归属应用接口（getOwnedApps），
 * 菜单树来自 /tenant/applications/{appID}/menus/tree（getAppMenuTree）；切换应用后按新 appID 重新加载。
 */
describe('菜单页应用切换器', () => {
  beforeEach(() => {
    localStorage.clear()
    mockGetTenantApps.mockReset()
    mockGetAppMenuTree.mockReset()
    mockGetAppMenuTree.mockResolvedValue({ list: [], total: 0 })
    mockGetTenantApps.mockResolvedValue({ list: apps })
  })

  it('首屏默认选中应用清单第一个并加载其菜单树，切换应用后按新 appID 重新加载', async () => {
    render(<MenuList />)

    // 首屏默认选中应用清单第一个，并加载它的菜单树
    await waitFor(() => expect(mockGetAppMenuTree).toHaveBeenCalledWith('app-1'))

    // 打开下拉并选中第二个应用 → 菜单树按新 appID 重新加载
    fireEvent.mouseDown(document.querySelector('.ant-select-content') as HTMLElement)
    fireEvent.click(await screen.findByText('订单中心（order）'))
    await waitFor(() => expect(mockGetAppMenuTree).toHaveBeenCalledWith('app-2'))
  })

  it('上次选择的应用已不在租户应用清单里时回退到第一个，且不用无效 ID 拉菜单树', async () => {
    localStorage.setItem('ark-iam:menu:appID', 'app-deleted')

    render(<MenuList />)

    // 无效 ID 不属于清单 → 回退到第一个
    await waitFor(() => expect(mockGetAppMenuTree).toHaveBeenCalledWith('app-1'))
    expect(mockGetAppMenuTree).not.toHaveBeenCalledWith('app-deleted')
  })
})

/**
 * 内置应用（source=builtin）的菜单同样支持新增/删除：新增行不带种子身份键（seed_key 为空），
 * 种子不会认领；删除内置菜单会留下软删"墓碑"，种子下次启动不再复活它。
 * 回归背景：此前内置应用整棵树禁止增删，「功能扩展/调整」必须改代码发版。
 */
describe('内置应用的菜单支持增删', () => {
  it('选中内置应用时可新建根菜单，行内提供新增子级/删除，编辑弹窗内字段（含编码）均可改', async () => {
    mockGetTenantApps.mockResolvedValue({ list: apps, total: apps.length })
    mockGetAppMenuTree.mockResolvedValue({
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
    // 图标字段已由 Input 改为 Select（图标下拉）：编辑态可见可交互的 combobox
    const iconSelect = screen.getAllByRole('combobox').find((el) => !(el as HTMLInputElement).disabled)
    expect(iconSelect).toBeTruthy()
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
    mockGetTenantApps.mockResolvedValue({ list: apps, total: apps.length })
    mockGetAppMenuTree.mockResolvedValue({ list: [menuRow], total: 1 })
    mockCreateAppMenu.mockReset()
    mockUpdateAppMenu.mockReset()
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

    await waitFor(() => expect(mockCreateAppMenu).toHaveBeenCalledTimes(1))
    const payload = mockCreateAppMenu.mock.calls[0][0] as Record<string, unknown>
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

    await waitFor(() => expect(mockUpdateAppMenu).toHaveBeenCalledTimes(1))
    const payload = mockUpdateAppMenu.mock.calls[0][0] as Record<string, unknown>
    expect(payload.hidden).toBe('disable')
    expect(payload.externalLink).toBe('disable')
    expect(payload.keepAlive).toBe('enable')
  })
})
