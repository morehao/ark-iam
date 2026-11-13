import { useCallback, useEffect, useRef, useState, type CSSProperties } from 'react'
import { Select, Spin } from 'antd'

export interface RemoteSelectOption {
  value: string
  label: string
}

interface RemoteOptionsProps {
  /** 远程取数：关键字为空表示取默认首页；返回值直接作为下拉选项 */
  fetchOptions: (keyword: string) => Promise<RemoteSelectOption[]>
  /** 搜索去抖毫秒数，默认 300 */
  debounceMs?: number
  /** 可选的外部名称缓存（`Map<值, 名称>`）：多个下拉共享已知名称，如筛选区选过的服务账号在弹窗表单里直接显示名称 */
  labelCache?: Map<string, string>
}

/**
 * 远程选项 + 「值 → 名称」缓存的共用逻辑（单选 / 多选只差渲染与回调形态）。
 *
 * 三条不变量：
 * 1. `filterOption={false}`：本地过滤关闭，过滤交给服务端（否则只能搜到已加载的那一页）；
 * 2. 过期响应丢弃（序号比对）：快速输入时旧关键字的结果不会覆盖新结果；
 * 3. 已选值始终有名称：`labelCache`（含外部缓存）与历史搜索结果累积，新搜索结果替换选项后
 *    当前选中项仍然显示名称而不是裸 ID。
 */
function useRemoteOptions({ fetchOptions, debounceMs = 300, labelCache }: RemoteOptionsProps) {
  const [options, setOptions] = useState<RemoteSelectOption[]>([])
  const [loading, setLoading] = useState(false)
  const internalLabelsRef = useRef(new Map<string, string>())
  const labels = labelCache ?? internalLabelsRef.current
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const seqRef = useRef(0)

  const search = useCallback(
    async (keyword: string) => {
      const seq = ++seqRef.current
      setLoading(true)
      try {
        const list = await fetchOptions(keyword)
        if (seq !== seqRef.current) return // 过期响应：丢弃，避免旧结果覆盖新关键字
        for (const item of list) labels.set(item.value, item.label)
        setOptions(list)
      } catch {
        /* 请求错误由 axios 拦截器统一提示 */
      } finally {
        if (seq === seqRef.current) setLoading(false)
      }
    },
    [fetchOptions, labels],
  )

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    },
    [],
  )

  const handleSearch = useCallback(
    (keyword: string) => {
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => void search(keyword), debounceMs)
    },
    [search, debounceMs],
  )

  // 只读取名：名称缓存 → 当前选项 → 兜底显示值本身
  const labelOf = useCallback(
    (v: string) => labels.get(v) ?? options.find((o) => o.value === v)?.label ?? v,
    [labels, options],
  )

  // 已选值不在当前选项里时补上前缀项，保证选中项显示名称而不是 ID
  const withSelected = useCallback(
    (values: string[]) => {
      const missing = values.filter((v) => !options.some((o) => o.value === v))
      return missing.length ? [...missing.map((v) => ({ value: v, label: labelOf(v) })), ...options] : options
    },
    [labelOf, options],
  )

  // 写入名称缓存，并把选中项返回给调用方（onChange 第二参）
  const remember = useCallback(
    (values: string[]) =>
      values.map((v) => {
        const label = labelOf(v)
        labels.set(v, label)
        return { value: v, label }
      }),
    [labelOf, labels],
  )

  return { loading, search, handleSearch, withSelected, remember, labels }
}

export interface RemoteSelectProps extends RemoteOptionsProps {
  /** 受控值（`Form.Item` 自动注入） */
  value?: string
  /** 选中回调；第二参为选中项，便于调用方保存名称（平台端列表经常要回显名称） */
  onChange?: (value?: string, option?: RemoteSelectOption) => void
  placeholder?: string
  disabled?: boolean
  allowClear?: boolean
  /** 固定宽度（筛选区用）；表单内不传则由容器撑满 */
  width?: number
  /** 额外样式（如 `{ width: '100%' }` 撑满表单） */
  style?: CSSProperties
  /** 已知当前值对应的名称：编辑态回显，避免选项未加载时显示裸 ID */
  initialLabel?: string
  id?: string
}

/**
 * 远程搜索下拉（单选）：选项由 `fetchOptions` 按关键字从服务端取，不在前端一次性全量加载。
 *
 * 解决「下拉只取前 N 条 → 数据量上百后选不到目标」的问题（租户/应用/服务账号等可增长资源）。
 * 不变量见 `useRemoteOptions`。
 */
export function RemoteSelect({
  value,
  onChange,
  initialLabel,
  placeholder,
  disabled,
  allowClear,
  width,
  style,
  id,
  ...rest
}: RemoteSelectProps) {
  const remote = useRemoteOptions(rest)

  useEffect(() => {
    if (value && initialLabel) remote.labels.set(value, initialLabel)
  }, [value, initialLabel, remote.labels])

  // 首屏取默认首页：不展开下拉也能看到候选项
  useEffect(() => {
    if (disabled) return
    void remote.search('')
  }, [remote.search, disabled])

  const handleChange = (next?: string) => {
    onChange?.(next, next ? remote.remember([next])[0] : undefined)
  }

  return (
    <Select
      id={id}
      showSearch
      filterOption={false}
      allowClear={allowClear}
      placeholder={placeholder}
      disabled={disabled}
      loading={remote.loading}
      style={width ? { width, ...style } : style}
      value={value}
      onChange={handleChange}
      onSearch={remote.handleSearch}
      options={remote.withSelected(value ? [value] : [])}
      notFoundContent={remote.loading ? <Spin size="small" /> : null}
    />
  )
}

export interface RemoteMultiSelectProps extends RemoteOptionsProps {
  /** 受控值（多选，`Form.Item` 自动注入） */
  value?: string[]
  onChange?: (value?: string[], option?: RemoteSelectOption[]) => void
  /** 已知已选值的名称（如主体当前已分配的角色）：避免搜索结果未命中时显示裸 ID */
  initialLabels?: Record<string, string>
  placeholder?: string
  disabled?: boolean
  allowClear?: boolean
  /** 固定宽度（筛选区用）；表单内不传则由容器撑满 */
  width?: number
  /** 额外样式（如 `{ width: '100%' }` 撑满表单） */
  style?: CSSProperties
  id?: string
}

/**
 * 远程搜索下拉（多选）：与 `RemoteSelect` 共用同一套「服务端搜索 + 过期响应丢弃 + 已选值名称兜底」，
 * 用于角色授权这类「候选集可增长 + 已选项必须回显名称」的多值场景。
 */
export function RemoteMultiSelect({
  value,
  onChange,
  initialLabels,
  placeholder,
  disabled,
  allowClear,
  width,
  style,
  id,
  ...rest
}: RemoteMultiSelectProps) {
  const remote = useRemoteOptions(rest)
  const values = value ?? []

  useEffect(() => {
    if (!initialLabels) return
    Object.entries(initialLabels).forEach(([v, label]) => {
      if (v && label) remote.labels.set(v, label)
    })
  }, [initialLabels, remote.labels])

  useEffect(() => {
    if (disabled) return
    void remote.search('')
  }, [remote.search, disabled])

  const handleChange = (next?: string[]) => {
    const nextValues = next ?? []
    onChange?.(nextValues, nextValues.length ? remote.remember(nextValues) : [])
  }

  return (
    <Select
      id={id}
      mode="multiple"
      showSearch
      filterOption={false}
      allowClear={allowClear}
      placeholder={placeholder}
      disabled={disabled}
      loading={remote.loading}
      style={width ? { width, ...style } : style}
      value={values}
      onChange={handleChange}
      onSearch={remote.handleSearch}
      options={remote.withSelected(values)}
      notFoundContent={remote.loading ? <Spin size="small" /> : null}
    />
  )
}
