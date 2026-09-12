import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { fmtTime } from '@ark-iam/ui'
import type { ApplicationItem } from '@ark-iam/types'

const mockGetApplicationPageList = vi.fn()
const mockCreateApplication = vi.fn()
vi.mock('@ark-iam/api', () => ({
  createApplication: (...args: unknown[]) => mockCreateApplication(...args),
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
    code: 'platform_admin',
    name: '平台管理后台',
    description: '',
    logoUrl: '',
    homepageUrl: '',
    source: 'third_party',
    status: 'enable',
    sort: 0,
    createdAt,
    updatedAt,
  },
]

describe('应用列表', () => {
  beforeEach(() => {
    mockGetApplicationPageList.mockReset().mockResolvedValue({ list: applications, total: applications.length })
    mockCreateApplication.mockReset().mockResolvedValue({ appID: 'app-2', code: 'my_app' })
  })

  it('同时展示创建时间与更新时间两列', async () => {
    render(<ApplicationList />)

    expect(await screen.findByText('平台管理后台')).toBeInTheDocument()
    // 时间列读 createdAt/updatedAt（回归：字段缺失会渲染 '-'）
    expect(screen.getByText(fmtTime(createdAt))).toBeInTheDocument()
    expect(screen.getByText(fmtTime(updatedAt))).toBeInTheDocument()
  })

  /**
   * 应用编码规则为下划线连接（与后端 model.AppCodePattern 同口径）：
   * 连字符/大写等非法编码必须在表单层被拦下，不得发起创建请求；改为下划线形态后放行。
   */
  it('应用编码只接受小写字母、数字与下划线', async () => {
    render(<ApplicationList />)

    fireEvent.click(await screen.findByRole('button', { name: /新建应用/ }))

    const codeInput = await screen.findByPlaceholderText('唯一编码，如 iam_web')
    const nameInput = screen.getByPlaceholderText('应用名称')
    const modal = document.querySelector('.ant-modal') as HTMLElement
    const submit = () => fireEvent.click(modal.querySelector('.ant-modal-footer .ant-btn-primary') as HTMLElement)

    fireEvent.change(codeInput, { target: { value: 'my-app' } })
    fireEvent.change(nameInput, { target: { value: '测试应用' } })
    submit()

    expect(await screen.findByText('以小写字母开头，仅含小写字母、数字与下划线')).toBeInTheDocument()
    expect(mockCreateApplication).not.toHaveBeenCalled()

    fireEvent.change(codeInput, { target: { value: 'my_app' } })
    submit()

    await waitFor(() =>
      expect(mockCreateApplication).toHaveBeenCalledWith(expect.objectContaining({ code: 'my_app' })),
    )
  })
})

/**
 * 内置应用（source=builtin）的展示类字段归运维：名称/描述/排序都可改，前端不置灰。
 * 回归背景：名称与描述一度是 reconcile（前端置灰 + 后端拒写），编码一度只读（种子按 code 定位）；
 * 引入 seed_key 后种子不再按 code 认行，名称/描述/排序交还运维。
 * **唯编码仍只读**（见下一组用例）：控制台菜单入口按它定位。
 */
describe('内置应用的字段可编辑', () => {
  it('编辑内置应用时名称、描述与排序均可编辑', async () => {
    mockGetApplicationPageList.mockResolvedValue({
      list: [{ ...applications[0], source: 'builtin' as const }],
      total: 1,
    })
    render(<ApplicationList />)

    fireEvent.click(await screen.findByText('编辑'))
    const nameInput = await screen.findByPlaceholderText('应用名称')
    await waitFor(() => expect(nameInput).not.toBeDisabled())
    expect(screen.getByPlaceholderText('选填')).not.toBeDisabled()
    // 排序归运维，仍可编辑
    expect(screen.getByPlaceholderText('数字越小越靠前')).not.toBeDisabled()
  })
})

/**
 * 应用编码可改，但**内置应用保持只读**：`platform_admin` / `tenant_admin` 是控制台菜单入口的
 * 定位值（svcpermission.MyTree 按编码查应用、tenantadmin loadConsoleApps 只保留 tenant_admin），
 * 从控制台改名会当场让对应控制台侧边栏失联且无法从界面恢复。
 * 后端 svcapplication.Update 以 ApplicationBuiltInCodeImmutableError 兜底。
 */
describe('编辑态应用编码的可写性', () => {
  it('内置应用编码置灰且回显当前值', async () => {
    mockGetApplicationPageList.mockResolvedValue({
      list: [{ ...applications[0], appID: 'app-builtin', code: 'platform_admin', source: 'builtin' as const }],
      total: 1,
    })
    render(<ApplicationList />)

    fireEvent.click(await screen.findByText('编辑'))
    const codeInput = await screen.findByPlaceholderText('唯一编码，如 iam_web')
    await waitFor(() => expect(codeInput).toBeDisabled())
    expect(codeInput).toHaveValue('platform_admin')
  })

  it('第三方应用编码可改且回显当前值', async () => {
    mockGetApplicationPageList.mockResolvedValue({
      list: [{ ...applications[0], appID: 'app-custom', code: 'customer_app', name: '客户应用', source: 'third_party' as const }],
      total: 1,
    })
    render(<ApplicationList />)

    fireEvent.click(await screen.findByText('编辑'))
    const codeInput = await screen.findByPlaceholderText('唯一编码，如 iam_web')
    expect(codeInput).not.toBeDisabled()
    expect(codeInput).toHaveValue('customer_app')
  })
})

/**
 * 内置应用（source=builtin，平台管理后台/租户管理后台）禁删：它们由平台版本交付，
 * 删除会让对应控制台失去应用主体。前端把「删除」置灰保留展示（不隐藏），
 * 后端 svcapplication.Delete 以 ApplicationBuiltInErr 兜底。
 */
describe('内置应用不可删除', () => {
  it('内置应用行的「删除」置灰不可点', async () => {
    mockGetApplicationPageList.mockResolvedValue({
      list: [{ ...applications[0], source: 'builtin' as const }],
      total: 1,
    })
    render(<ApplicationList />)

    expect(await screen.findByText('平台管理后台')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '删除' })).toBeDisabled()
  })

  it('第三方应用行的「删除」可点', async () => {
    mockGetApplicationPageList.mockResolvedValue({ list: applications, total: 1 })
    render(<ApplicationList />)

    expect(await screen.findByText('平台管理后台')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '删除' })).not.toBeDisabled()
  })
})
