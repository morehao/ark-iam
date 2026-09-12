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
 * 内置应用（source=builtin）的名称/描述由平台版本定义（后端字段权威矩阵 reconcile，控制台拒写），
 * 前端必须置灰并给出说明；启停与排序仍可改。回归背景：种子会收敛 name/description，
 * 若前端仍可编辑，运维改完重启就被收回（双写者）。
 */
describe('内置应用的身份字段只读', () => {
  it('编辑内置应用时名称与描述置灰', async () => {
    mockGetApplicationPageList.mockResolvedValue({
      list: [{ ...applications[0], source: 'builtin' as const }],
      total: 1,
    })
    render(<ApplicationList />)

    fireEvent.click(await screen.findByText('编辑'))
    const nameInput = await screen.findByPlaceholderText('应用名称')
    await waitFor(() => expect(nameInput).toBeDisabled())
    expect(screen.getByPlaceholderText('选填')).toBeDisabled()
    expect(screen.getByText('内置应用的名称由平台版本定义，不可修改')).toBeInTheDocument()
    // 排序归运维，仍可编辑
    expect(screen.getByPlaceholderText('数字越小越靠前')).not.toBeDisabled()
  })
})
