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
/** 自建角色：编码是跨系统授权契约值（OIDC groups 取值），可改。 */
const customRole: TenantRoleItem = {
  roleID: '01a09372-0000-7000-8000-00000000r001',
  appID,
  appName,
  code: 'dept_manager',
  name: '部门管理员',
  description: '管理本部门成员',
  source: 'custom',
  adminType: 'normal',
  memberCount: 2,
  menuCount: 3,
  createdAt: 1789142400,
  updatedAt: 1789228800,
}
/** 内置角色（种子 RoleCodeTenantAdmin）：编码是下游策略供给的锚点，删除禁用、编辑提示更强。 */
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
const CODE_INVALID_MESSAGE = '以小写字母开头，仅含小写字母、数字与下划线'

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
  /** 回归：编码是下游认策略名的契约值，列表必须能逐字看到（后端 DTO/DB 字段名都是 code）。 */
  it('展示后端 code 且有「编码」表头', async () => {
    render(<TenantRolePage />)

    expect(await screen.findByText('dept_manager')).toBeInTheDocument()
    // 表头在 antd 内部会渲染多份（度量行），只断言存在
    expect(screen.getAllByText('编码').length).toBeGreaterThan(0)
  })

  /** 列顺序：编码紧随角色名称之后（名称是主键、编码其次）。 */
  it('编码列排在角色名称列之后', async () => {
    render(<TenantRolePage />)

    await screen.findByText('dept_manager')
    const headers = screen.getAllByRole('columnheader').map((th) => th.textContent)
    expect(headers.indexOf('角色名称')).toBeGreaterThanOrEqual(0)
    expect(headers.indexOf('角色名称')).toBeLessThan(headers.indexOf('编码'))
  })
})

describe('新建角色提交编码', () => {
  it('提交 payload 含 code / name / appID', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByRole('button', { name: /新建角色/ }))
    fireEvent.change(await screen.findByPlaceholderText(CODE_INPUT_PLACEHOLDER), {
      target: { value: 'dept_manager' },
    })
    fireEvent.change(screen.getByPlaceholderText(CODE_NAME_PLACEHOLDER), { target: { value: '部门管理员' } })
    await selectApp(appName)
    submitModal()

    await waitFor(() => expect(mockCreateTenantRole).toHaveBeenCalledTimes(1))
    expect(mockCreateTenantRole.mock.calls[0][0]).toMatchObject({
      appID,
      code: 'dept_manager',
      name: '部门管理员',
    })
  })

  /**
   * 回归：编码是创建时的必填入参（服务端不再生成），形状必须在表单层拦下——
   * 大写/连字符/数字开头/空格/点 全部非法，且非法时不得发出创建请求。
   */
  it('编码非法时拦截提交，不发创建请求', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByRole('button', { name: /新建角色/ }))
    const codeInput = await screen.findByPlaceholderText(CODE_INPUT_PLACEHOLDER)
    fireEvent.change(screen.getByPlaceholderText(CODE_NAME_PLACEHOLDER), { target: { value: '部门管理员' } })
    await selectApp(appName)

    for (const badCode of ['Dept_Manager', 'dept-manager', '1dept', 'dept manager']) {
      fireEvent.change(codeInput, { target: { value: badCode } })
      submitModal()
      expect(await screen.findByText(CODE_INVALID_MESSAGE)).toBeInTheDocument()
    }
    expect(mockCreateTenantRole).not.toHaveBeenCalled()

    // 换成合法编码后放行
    fireEvent.change(codeInput, { target: { value: 'dept_manager' } })
    submitModal()
    await waitFor(() => expect(mockCreateTenantRole).toHaveBeenCalledTimes(1))
  })

  /** 编码为空同样被必填规则拦下（创建必填，服务端 binding:"required"）。 */
  it('编码为空时拦截提交', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByRole('button', { name: /新建角色/ }))
    fireEvent.change(await screen.findByPlaceholderText(CODE_NAME_PLACEHOLDER), { target: { value: '部门管理员' } })
    await selectApp(appName)
    submitModal()

    expect(await screen.findByText('请输入角色编码')).toBeInTheDocument()
    expect(mockCreateTenantRole).not.toHaveBeenCalled()
  })
})

describe('编辑角色回显并提交编码', () => {
  /** 更新是全量覆盖：编码必须回显并回传，漏传会把下游认策略名的契约值清空。 */
  it('编辑态回显 code 且提交 payload 含 code', async () => {
    render(<TenantRolePage />)

    fireEvent.click(await screen.findByText('编辑'))
    const codeInput = await screen.findByPlaceholderText(CODE_INPUT_PLACEHOLDER)
    expect(codeInput).toHaveValue('dept_manager')

    submitModal()

    await waitFor(() => expect(mockUpdateTenantRole).toHaveBeenCalledTimes(1))
    expect(mockUpdateTenantRole.mock.calls[0][0]).toMatchObject({
      roleID: customRole.roleID,
      code: 'dept_manager',
      name: '部门管理员',
    })
  })
})

/**
 * 内置角色整体只读：编码是下游策略供给的锚点（下游按「前缀 + 编码」认策略名），名称/描述同样由种子与运维负责，
 * 故「编辑」与「删除」在列表页都置灰，权威判定在后端（RoleUpdateBuiltinForbiddenError / RoleDeleteBuiltinForbiddenError）；
 * 菜单授权走独立接口，仍可点。
 */
describe('内置角色只读', () => {
  it('内置角色的「编辑」「删除」置灰，「菜单权限」仍可点', async () => {
    mockGetTenantRolePageList.mockResolvedValue({ list: [builtinRole], total: 1 })
    render(<TenantRolePage />)

    expect(await screen.findByText('tenant_admin')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '编辑' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '菜单权限' })).not.toBeDisabled()
  })
})
