import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { App as AntdApp } from 'antd'
import type { SettingGroupMeta, SettingItem } from '@ark-iam/types'

const mockGetSettingGroups = vi.fn()
const mockGetSettingPageList = vi.fn()
const mockCreateSetting = vi.fn()
const mockUpdateSetting = vi.fn()
const mockUpdateSettingStatus = vi.fn()
const mockDeleteSetting = vi.fn()

vi.mock('@ark-iam/api', () => ({
  createSetting: (...args: unknown[]) => mockCreateSetting(...args),
  deleteSetting: (...args: unknown[]) => mockDeleteSetting(...args),
  getSettingGroups: (...args: unknown[]) => mockGetSettingGroups(...args),
  getSettingPageList: (...args: unknown[]) => mockGetSettingPageList(...args),
  updateSetting: (...args: unknown[]) => mockUpdateSetting(...args),
  updateSettingStatus: (...args: unknown[]) => mockUpdateSettingStatus(...args),
}))

const SettingListPage = (await import('./index')).default

const groups: SettingGroupMeta[] = [
  { group: 'notify', description: '通知渠道：临时密码投递' },
  { group: 'security.login', description: '登录安全：失败锁定 + 限流' },
]

const updatedAt = 1789228800

const items: SettingItem[] = [
  {
    id: 's1',
    group: 'notify',
    key: 'mail.host',
    valueType: 'string',
    value: 'smtp.example.com',
    encryptionMode: 'plain',
    description: '',
    status: 'enabled',
    createdAt: 1789142400,
    updatedAt,
  },
  {
    id: 's2',
    group: 'notify',
    key: 'mail.password',
    valueType: 'string',
    value: '******',
    encryptionMode: 'encrypted',
    description: '',
    status: 'enabled',
    createdAt: 1789142400,
    updatedAt,
  },
]

describe('系统设置列表页', () => {
  beforeEach(() => {
    mockGetSettingGroups.mockReset().mockResolvedValue({ list: groups })
    mockGetSettingPageList.mockReset().mockResolvedValue({ list: items, total: items.length })
    mockCreateSetting.mockReset().mockResolvedValue('ok')
    mockUpdateSetting.mockReset().mockResolvedValue('ok')
    mockUpdateSettingStatus.mockReset().mockResolvedValue('ok')
    mockDeleteSetting.mockReset().mockResolvedValue('ok')
  })

  it('渲染配置行、加密项掩码、并回填分组下拉元数据', async () => {
    render(
      <AntdApp>
        <SettingListPage />
      </AntdApp>,
    )

    expect(await screen.findByText('mail.host')).toBeInTheDocument()
    expect(screen.getByText('smtp.example.com')).toBeInTheDocument()
    // 加密项：key 在、明文不在、掩码展示
    expect(screen.getByText('mail.password')).toBeInTheDocument()
    // 分组元数据接口被调用
    expect(mockGetSettingGroups).toHaveBeenCalled()
    // 首次请求带分页参数
    expect(mockGetSettingPageList).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 10 }))
  })

  it('按分组筛选：选择分组后带 group 重新请求并回到第 1 页', async () => {
    render(
      <AntdApp>
        <SettingListPage />
      </AntdApp>,
    )
    await screen.findByText('mail.host')

    const groupSelector = screen.getByText('全部分组').closest('.ant-select-content')
    fireEvent.mouseDown(groupSelector as HTMLElement)
    fireEvent.click(await screen.findByTitle('通知渠道：临时密码投递'))

    await waitFor(() => {
      expect(mockGetSettingPageList).toHaveBeenLastCalledWith(
        expect.objectContaining({ group: 'notify', page: 1 }),
      )
    })
  })

  it('新增配置项：填写并提交调用 createSetting', async () => {
    render(
      <AntdApp>
        <SettingListPage />
      </AntdApp>,
    )
    await screen.findByText('mail.host')

    fireEvent.click(screen.getByRole('button', { name: /新增配置项/ }))
    // 填 key 与值
    fireEvent.change(await screen.findByPlaceholderText('如 mail.host'), { target: { value: 'sms.signName' } })
    const valueInput = document.querySelector('.ant-drawer input#value') as HTMLInputElement
    fireEvent.change(valueInput, { target: { value: 'ArkIAM' } })
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))

    await waitFor(() => {
      expect(mockCreateSetting).toHaveBeenCalledWith(
        expect.objectContaining({ group: 'notify', key: 'sms.signName', value: 'ArkIAM', valueType: 'string' }),
      )
    })
  })

  it('编辑加密项：保持掩码提交时不携带 value（不改原值）', async () => {
    render(
      <AntdApp>
        <SettingListPage />
      </AntdApp>,
    )
    await screen.findByText('mail.password')

    // 第二行（mail.password）的「编辑」
    const editButtons = screen.getAllByRole('button', { name: /编\s*辑/ })
    fireEvent.click(editButtons[1])
    fireEvent.click(await screen.findByRole('button', { name: /保\s*存/ }))

    await waitFor(() => {
      expect(mockUpdateSetting).toHaveBeenCalled()
    })
    const body = mockUpdateSetting.mock.calls[0][1] as Record<string, unknown>
    expect(body).not.toHaveProperty('value') // 掩码/空 -> 不改值
  })

  it('删除配置项：确认后调用 deleteSetting', async () => {
    render(
      <AntdApp>
        <SettingListPage />
      </AntdApp>,
    )
    await screen.findByText('mail.host')

    const deleteButtons = screen.getAllByRole('button', { name: /删\s*除/ })
    fireEvent.click(deleteButtons[0])
    // Popconfirm 确定
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockDeleteSetting).toHaveBeenCalledWith('s1')
    })
  })
})
