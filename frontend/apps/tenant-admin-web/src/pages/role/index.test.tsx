import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { TenantRoleItem } from '@ark-iam/types'
import { ROLE_CODE_PATTERN } from './index'

const mockGetTenantRolePageList = vi.fn()
const mockGetTenantApps = vi.fn()
const mockCreateTenantRole = vi.fn()
const mockUpdateTenantRole = vi.fn()

vi.mock('../../api/role', () => ({
  createTenantRole: (...args: unknown[]) => mockCreateTenantRole(...args),
  deleteTenantRole: vi.fn(),
  getTenantRoleMenus: vi.fn().mockResolvedValue({ list: [], menuIDs: [] }),
  getTenantRolePageList: (...args: unknown[]) => mockGetTenantRolePageList(...args),
  updateTenantRole: (...args: unknown[]) => mockUpdateTenantRole(...args),
  updateTenantRoleMenus: vi.fn(),
}))
vi.mock('../../api/menu', () => ({
  getTenantApps: (...args: unknown[]) => mockGetTenantApps(...args),
}))

const TenantRolePage = (await import('./index')).default

const appID = '01a09372-0000-7000-8000-00000000a1b1'
const appName = '租户控制台'
/** 自建角色：不参与跨系统契约，编码恒为空串（契约值只能来自应用角色模板）。 */
const customRole: TenantRoleItem = {
  roleID: '01a09372-0000-7000-8000-00000000r001',
  appID,
  appName,
  code: '',
  name: '部门管理员',
  description: '管理本部门成员',
  source: 'custom',
  adminType: 'normal',
  memberCount: 2,
  menuCount: 3,
  createdAt: 1789142400,
  updatedAt: 1789228800,
}
/** 模板角色：由应用角色模板物化（source=builtin、admin_type=normal），编码是下游认策略名的契约值。 */
const templateRole: TenantRoleItem = {
  roleID: '01a09372-0000-7000-8000-00000000r003',
  appID,
  appName,
  code: 'storage_readonly',
  name: '存储只读',
  description: '',
  source: 'builtin',
  adminType: 'normal',
  memberCount: 0,
  menuCount: 0,
  createdAt: 1789142400,
  updatedAt: 1789228800,
}
/** 产品锚点角色（种子 RoleCodeTenantAdmin）：删除禁用、编辑提示更强。 */
const builtinRole: TenantRoleItem = {
  roleID: '01a09372-0000-7000-8000-00000000r002',
  appID: '',
  appName: '',
  code: 'tenant_admin',
  name: '租户管理员',
  description: '',
  source: 'builtin',
  adminType: 'admin',
  memberCount: 1,
  menuCount: 12,
  createdAt: 1789142400,
  updatedAt: 1789228800,
}

/** 展开弹窗内的「所属应用」下拉并选中一项（antd Select 在 jsdom 下需 mouseDown 才渲染选项）。 */
async function selectApp(label: string) {
  const modal = document.querySelector('.ant-modal') as HTMLElement
  fireEvent.mouseDown(modal.querySelector('.ant-select-selector') as HTMLElement)
  fireEvent.click(await screen.findByTitle(label))
}

/** 点击弹窗底部主按钮（提交）。 */
function submitModal() {
  const modal = document.querySelector('.ant-modal') as HTMLElement
  fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)
}

const CODE_INPUT_PLACEHOLDER = '唯一编码，如 dept_manager'
const CODE_NAME_PLACEHOLDER = '如：部门管理员（应用内唯一）'

beforeEach(() => {
  mockGetTenantRolePageList.mockReset().mockResolvedValue({ list: [customRole], total: 1 })
  mockGetTenantApps.mockReset().mockResolvedValue({ list: [{ appID, code: 'tenant_admin_web', name: appName }] })
  mockCreateTenantRole.mockReset().mockResolvedValue({ roleID: 'r-new' })
  mockUpdateTenantRole.mockReset().mockResolvedValue(undefined)
})

/**
 * 形状规则必须与后端 `model.RoleCodePattern`（`^[a-z][a-z0-9_]*$`）逐字同口径：
 * 正则跨语言无法共享，改一处必须同步另一处，故这里直接对前端常量做正反用例断言。
 */
describe('ROLE_CODE_PATTERN 与后端 model.RoleCodePattern 同口径', () => {
  it.each(['a', 'dept_manager', 'tenant_admin', 'role1', 'a_b_c_9'])('接受 %s', (code) => {
    expect(ROLE_CODE_PATTERN.test(code)).toBe(true)
  })

  it.each(['Dept_Manager', 'dept-manager', '1dept', '_dept', 'dept manager', 'dept.manager', ''])(
    '拒绝 %s',
    (code) => {
      expect(ROLE_CODE_PATTERN.test(code)).toBe(false)
    },
  )
})

describe('租户角色列表的编码列', () => {
  /** 编码是下游认策略名的契约值，模板角色必须能逐字看到（后端 DTO/DB 字段名都是 code）。 */
  it('模板角色展示编码，自建角色展示「不参与契约」文案', async () => {
    mockGetTenantRolePageList.mockResolvedValue({ list: [customRole, templateRole], total: 2 })
    render(<TenantRolePage />)

    expect(await screen.findByText('storage_readonly')).toBeInTheDocument()
    expect(screen.getByText('自建（不参与契约）')).toBeInTheDocument()
    // 表头在 antd 内部会渲染多份（度量行），只断言存在
    expect(screen.getAllByText('编码').length).toBeGreaterThan(0)
  })

  /** 列顺序：编码紧随角色名称之后（名称是主键、编码其次）。 */
  it('编码列排在角色名称列之后', async () => {
    mockGetTenantRolePageList.mockResolvedValue({ list: [templateRole], total: 1 })
    render(<TenantRolePage />)

    await screen.findByText('storage_readonly')
    const headers = screen.getAllByRole('columnheader').map((th) => th.textContent)
    expect(headers.indexOf('角色名称')).toBeGreaterThanOrEqual(0)
    expect(headers.indexOf('角色名称')).toBeLessThan(headers.indexOf('编码'))
  })
})

describe('租户侧不能写入角色编码', () => {
  /**
   * 契约值归应用方：租户控制台**结构上没有**编码入口（表单无该字段，提交也不带 code）。
   * 这是「自建角色不可能撞码/自造下游策略名」的前端一半保证，后端一半见
   * dtotenant.RoleCreateReq（无 Code 字段）+ svcrole.Create（Code 恒为空串）。
   */
  it('新建弹窗没有编码输入项，提交 payload 不含 code', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByRole('button', { name: /新建角色/ }))
    fireEvent.change(await screen.findByPlaceholderText(CODE_NAME_PLACEHOLDER), { target: { value: '部门管理员' } })
    expect(screen.queryByPlaceholderText(CODE_INPUT_PLACEHOLDER)).not.toBeInTheDocument()
    await selectApp(appName)
    submitModal()

    await waitFor(() => expect(mockCreateTenantRole).toHaveBeenCalledTimes(1))
    expect(mockCreateTenantRole.mock.calls[0][0]).toMatchObject({ appID, name: '部门管理员' })
    expect(mockCreateTenantRole.mock.calls[0][0]).not.toHaveProperty('code')
  })

  it('名称必填（编码已不再是必填项）', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByRole('button', { name: /新建角色/ }))
    await selectApp(appName)
    submitModal()

    expect(await screen.findByText('请输入角色名称')).toBeInTheDocument()
    expect(mockCreateTenantRole).not.toHaveBeenCalled()
  })
})

describe('编辑自建角色只提交名称/描述', () => {
  /** 更新只改名称/描述：编码不在入参里（自建角色恒为空串），也不得回传清空契约值。 */
  it('编辑态无编码项，提交 payload 不含 code', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByText('编辑'))
    const nameInput = await screen.findByPlaceholderText(CODE_NAME_PLACEHOLDER)
    expect(nameInput).toHaveValue('部门管理员')
    expect(screen.queryByPlaceholderText(CODE_INPUT_PLACEHOLDER)).not.toBeInTheDocument()

    submitModal()

    await waitFor(() => expect(mockUpdateTenantRole).toHaveBeenCalledTimes(1))
    expect(mockUpdateTenantRole.mock.calls[0][0]).toMatchObject({
      roleID: customRole.roleID,
      name: '部门管理员',
      description: '管理本部门成员',
    })
    expect(mockUpdateTenantRole.mock.calls[0][0]).not.toHaveProperty('code')
  })
})

/**
 * 内置角色整体只读：编码是下游策略供给的锚点（下游按「前缀 + 编码」认策略名），名称/描述同样由种子与运维负责，
 * 故「编辑」与「删除」在列表页都置灰，权威判定在后端（RoleUpdateBuiltinForbiddenError / RoleDeleteBuiltinForbiddenError）；
 * 菜单授权走独立接口，仍可点。
 */
describe('内置角色只读', () => {
  it('产品锚点与模板角色的「编辑」「删除」置灰，「菜单权限」仍可点', async () => {
    mockGetTenantRolePageList.mockResolvedValue({ list: [builtinRole, templateRole], total: 2 })
    render(<TenantRolePage />)

    expect(await screen.findByText('tenant_admin')).toBeInTheDocument()
    expect(screen.getByText('storage_readonly')).toBeInTheDocument()
    // 两个内置角色的编辑/删除都置灰（模板角色的编码与名称由应用方定义，租户不得改写）
    expect(screen.getAllByRole('button', { name: '编辑' })).toHaveLength(2)
    for (const btn of screen.getAllByRole('button', { name: '编辑' })) {
      expect(btn).toBeDisabled()
    }
    for (const btn of screen.getAllByRole('button', { name: '删除' })) {
      expect(btn).toBeDisabled()
    }
    expect(screen.getAllByRole('button', { name: '菜单权限' })).toHaveLength(2)
  })
})
