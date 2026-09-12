import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RemoteMultiSelect, RemoteSelect } from '@ark-iam/ui'
import type { RemoteSelectOption } from '@ark-iam/ui'

/**
 * RemoteSelect 回归要点（解决「下拉只取前 N 条 → 数据量上百后选不到目标」）：
 * 1. 选项由服务端按关键字返回，本地过滤关闭；
 * 2. 过期响应被丢弃：更早请求的结果晚到时不得覆盖最新请求的结果；
 * 3. 搜索替换选项后，当前已选值仍显示名称而不是裸 ID。
 */
describe('RemoteSelect', () => {
  it('选项被新搜索结果替换后，已选值仍显示名称', async () => {
    const fetchOptions = vi.fn().mockResolvedValue([])
    render(<RemoteSelect value="t2" initialLabel="租户二" fetchOptions={fetchOptions} />)

    await waitFor(() => expect(fetchOptions).toHaveBeenCalledWith(''))
    expect(await screen.findByText('租户二')).toBeInTheDocument()
    expect(screen.queryByText('t2')).toBeNull()
  })

  it('输入关键字走服务端搜索并展示返回项', async () => {
    const fetchOptions = vi.fn(async (keyword: string): Promise<RemoteSelectOption[]> =>
      keyword ? [{ value: 't9', label: '目标租户' }] : [{ value: 't2', label: '租户二' }],
    )
    const onChange = vi.fn()
    render(<RemoteSelect fetchOptions={fetchOptions} debounceMs={0} onChange={onChange} placeholder="选择租户" />)

    await waitFor(() => expect(fetchOptions).toHaveBeenCalledWith(''))

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '目标' } })
    await waitFor(() => expect(fetchOptions).toHaveBeenCalledWith('目标'))

    fireEvent.click(await screen.findByText('目标租户'))
    expect(onChange).toHaveBeenCalledWith('t9', { value: 't9', label: '目标租户' })
  })

  it('过期请求的结果晚到时不会覆盖最新结果', async () => {
    const calls: Array<{ keyword: string; resolve: (options: RemoteSelectOption[]) => void }> = []
    const fetchOptions = vi.fn(
      (keyword: string) =>
        new Promise<RemoteSelectOption[]>((resolve) => {
          calls.push({ keyword, resolve })
        }),
    )
    render(<RemoteSelect fetchOptions={fetchOptions} debounceMs={0} />)

    // 输入关键字本身会展开下拉（antd 行为），无需再点开
    const input = screen.getByRole('combobox')
    fireEvent.change(input, { target: { value: 'old' } })
    await waitFor(() => expect(calls.some((c) => c.keyword === 'old')).toBe(true))
    fireEvent.change(input, { target: { value: 'new' } })
    await waitFor(() => expect(calls.some((c) => c.keyword === 'new')).toBe(true))

    // 最新一次请求（关键字 'new'）先返回，界面显示「新结果」
    const newest = calls[calls.length - 1]
    expect(newest.keyword).toBe('new')
    newest.resolve([{ value: 'n1', label: '新结果' }])
    expect(await screen.findByText('新结果')).toBeInTheDocument()

    // 更早的请求（空关键字与 'old'）随后才返回「旧结果」：必须被丢弃
    calls
      .filter((c) => c !== newest)
      .forEach((c) => c.resolve([{ value: 'o1', label: '旧结果' }]))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(screen.queryByText('旧结果')).toBeNull()
    expect(screen.getByText('新结果')).toBeInTheDocument()
  })

  it('多选模式：已选项回显名称，搜索后以数组回调', async () => {
    const fetchOptions = vi.fn(async (keyword: string): Promise<RemoteSelectOption[]> =>
      keyword ? [{ value: 'r2', label: '订单管理员' }] : [],
    )
    const onChange = vi.fn()
    render(
      <RemoteMultiSelect
        value={['r1']}
        initialLabels={{ r1: '租户管理员' }}
        fetchOptions={fetchOptions}
        debounceMs={0}
        onChange={onChange}
      />,
    )

    // 已选项不在搜索结果里，仍必须显示名称而不是 ID
    expect(await screen.findByText('租户管理员')).toBeInTheDocument()
    expect(screen.queryByText('r1')).toBeNull()

    // 服务端搜索后勾选第二项 → 回调数组带上两项（含名称）
    fireEvent.change(screen.getByRole('combobox'), { target: { value: '订单' } })
    await waitFor(() => expect(fetchOptions).toHaveBeenCalledWith('订单'))
    fireEvent.click(await screen.findByText('订单管理员'))
    expect(onChange).toHaveBeenCalledWith(
      ['r1', 'r2'],
      expect.arrayContaining([
        { value: 'r1', label: '租户管理员' },
        { value: 'r2', label: '订单管理员' },
      ]),
    )
  })
})
