import { useCallback, useEffect, useState } from 'react'
import { Button, Drawer, Form, Input, InputNumber, Select, Space, Switch, Table, Tag, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import {
  actionColumn,
  CODE_COL_WIDTH,
  LONG_TEXT_COL_WIDTH,
  NAME_COL_WIDTH,
  PageContainer,
  STATUS_COL_WIDTH,
  StatusSwitch,
  TAG_COL_WIDTH,
  tableScrollX,
  textColumn,
  timeColumn,
} from '@ark-iam/ui'
import {
  createSetting,
  deleteSetting,
  getSettingGroups,
  getSettingPageList,
  updateSetting,
  updateSettingStatus,
} from '@ark-iam/api'
import type { SettingGroupMeta, SettingItem, SettingStatus, SettingValueType } from '@ark-iam/types'

/** 掩码值：后端对加密项回显此值；提交该值表示「不修改」。 */
const MASKED_VALUE = '******'

/** 值类型下拉选项（后端 gsetting 支持的类型）。 */
const VALUE_TYPE_OPTIONS: { value: SettingValueType; label: string }[] = [
  { value: 'string', label: 'string' },
  { value: 'int', label: 'int' },
  { value: 'float', label: 'float' },
  { value: 'bool', label: 'bool' },
  { value: 'json', label: 'json' },
  { value: 'toml', label: 'toml' },
  { value: 'yaml', label: 'yaml' },
]

/** 状态下拉选项。 */
const STATUS_OPTIONS: { value: SettingStatus; label: string }[] = [
  { value: 'enabled', label: '启用' },
  { value: 'disabled', label: '停用' },
]

/** 新增/编辑抽屉的表单值。 */
interface SettingFormValues {
  group: string
  key: string
  valueType: SettingValueType
  value: string | number | boolean
  encrypted: boolean
  status: SettingStatus
  description?: string
}

/** 把一个配置项的值字符串转成表单控件值（bool/int/float 转原生类型，其余保留字符串）。 */
function toFormValue(it: SettingItem): string | number | boolean {
  if (it.encryptionMode === 'encrypted') return MASKED_VALUE // 加密项保持掩码，提交即「不改」
  switch (it.valueType) {
    case 'bool':
      return it.value === 'true'
    case 'int':
    case 'float':
      return it.value === '' ? '' : Number(it.value)
    default:
      return it.value
  }
}

/** 表单控件值转提交用的字符串。 */
function toSubmitStr(v: string | number | boolean, type: SettingValueType): string {
  if (type === 'bool') return v ? 'true' : 'false'
  if (v === undefined || v === null) return ''
  return String(v)
}

export default function SettingListPage() {
  const [data, setData] = useState<SettingItem[]>([])
  const [groups, setGroups] = useState<SettingGroupMeta[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [filterGroup, setFilterGroup] = useState<string | undefined>(undefined)
  const [filterKey, setFilterKey] = useState('')
  const [filterStatus, setFilterStatus] = useState<SettingStatus | undefined>(undefined)

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<SettingItem | null>(null)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [form] = Form.useForm<SettingFormValues>()
  // 监听值类型，决定「值」字段的控件形态。
  const watchedValueType = Form.useWatch('valueType', form)

  const fetchGroups = useCallback(async () => {
    try {
      const resp = await getSettingGroups()
      setGroups(resp?.list || [])
    } catch {
      /* 拦截器已提示 */
    }
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getSettingPageList({
        group: filterGroup,
        key: filterKey || undefined,
        status: filterStatus,
        page,
        pageSize,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [filterGroup, filterKey, filterStatus, page, pageSize])

  useEffect(() => {
    void fetchGroups()
  }, [fetchGroups])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      group: filterGroup ?? groups[0]?.group,
      valueType: 'string',
      encrypted: false,
      status: 'enabled',
    })
    setDrawerOpen(true)
  }

  const openEdit = (record: SettingItem) => {
    setEditing(record)
    form.setFieldsValue({
      group: record.group,
      key: record.key,
      valueType: record.valueType,
      value: toFormValue(record),
      encrypted: record.encryptionMode === 'encrypted',
      status: record.status,
      description: record.description,
    })
    setDrawerOpen(true)
  }

  const handleSubmit = async () => {
    let values: SettingFormValues
    try {
      values = await form.validateFields()
    } catch {
      return
    }
    setSubmitLoading(true)
    try {
      if (!editing) {
        // 新增：值必填，按类型转字符串。
        await createSetting({
          group: values.group,
          key: values.key,
          valueType: values.valueType,
          value: toSubmitStr(values.value, values.valueType),
          encrypted: values.encrypted,
          status: values.status,
          description: values.description,
        })
      } else {
        // 编辑：group/key/valueType 只读（后端不支持改），只提交 value/encrypted/status/description。
        const body: { value?: string; encrypted?: boolean; status?: SettingStatus; description?: string } = {
          status: values.status,
          description: values.description,
        }
        // 加密项：留空或仍为掩码 = 不修改值。
        const isSecret = editing.encryptionMode === 'encrypted'
        const rawStr = toSubmitStr(values.value, editing.valueType)
        if (!isSecret) {
          body.value = rawStr
        } else if (rawStr !== '' && rawStr !== MASKED_VALUE) {
          body.value = rawStr
          body.encrypted = true
        }
        await updateSetting(editing.id, body)
      }
      message.success('保存成功')
      setDrawerOpen(false)
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (record: SettingItem) => {
    try {
      await deleteSetting(record.id)
      message.success('删除成功')
      // 删除当前页最后一条时回退一页，避免停在空页。
      if (data.length === 1 && page > 1) setPage(page - 1)
      else void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleToggleStatus = async (record: SettingItem, nextEnabled: boolean) => {
    const next: SettingStatus = nextEnabled ? 'enabled' : 'disabled'
    try {
      await updateSettingStatus(record.id, next)
      message.success('修改成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<SettingItem> = [
    {
      title: '分组',
      dataIndex: 'group',
      key: 'group',
      width: CODE_COL_WIDTH,
      render: (group: string) => <Tag color="blue" style={{ fontFamily: 'monospace' }}>{group}</Tag>,
    },
    textColumn<SettingItem>({ title: 'Key', dataIndex: 'key', width: NAME_COL_WIDTH, monospace: true }),
    {
      title: '类型',
      dataIndex: 'valueType',
      key: 'valueType',
      width: TAG_COL_WIDTH,
      render: (type: string, r) => (
        <Space size={4}>
          <Tag>{type}</Tag>
          {r.encryptionMode === 'encrypted' ? <Tag color="orange">加密</Tag> : null}
        </Space>
      ),
    },
    {
      title: '值',
      dataIndex: 'value',
      key: 'value',
      width: 200,
      render: (value: string, r) =>
        r.encryptionMode === 'encrypted' ? (
          <span style={{ letterSpacing: 2, color: 'rgba(0,0,0,0.45)' }}>{MASKED_VALUE}</span>
        ) : (
          <span style={{ fontFamily: 'ui-monospace, monospace', wordBreak: 'break-all' }}>{value || '-'}</span>
        ),
    },
    textColumn<SettingItem>({ title: '描述', dataIndex: 'description', width: LONG_TEXT_COL_WIDTH }),
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: STATUS_COL_WIDTH,
      render: (_: SettingStatus, r) => (
        <StatusSwitch
          checked={r.status === 'enabled'}
          checkedChildren="启用"
          unCheckedChildren="停用"
          onConfirmChange={(next) => handleToggleStatus(r, next)}
        />
      ),
    },
    timeColumn<SettingItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<SettingItem>({
      max: 3,
      variant: 'icon',
      actions: (r) => [
        { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openEdit(r) },
        { key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true, confirm: '确认删除该配置项？', onClick: () => void handleDelete(r) },
      ],
    }),
  ]

  const isSecretEditing = editing?.encryptionMode === 'encrypted'

  // 值字段控件：新增/编辑共用一个渲染逻辑。
  const renderValueControl = () => {
    // 布尔用 Switch
    if (watchedValueType === 'bool') {
      return <Switch checkedChildren="true" unCheckedChildren="false" />
    }
    // 数字用 InputNumber
    if (watchedValueType === 'int' || watchedValueType === 'float') {
      return <InputNumber style={{ width: '100%' }} step={watchedValueType === 'float' ? 0.1 : 1} />
    }
    // 加密项用密码框，占位提示留空/掩码=不修改
    if (editing?.encryptionMode === 'encrypted') {
      return <Input.Password placeholder="留空或保持掩码表示不修改" autoComplete="new-password" />
    }
    return <Input />
  }

  return (
    <PageContainer
      title="配置中心"
      description="运行时配置中心 · 保存后即时生效，无需重启服务"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增配置项
          </Button>
        </Space>
      }
      toolbar={
        <Space wrap>
          <Select
            placeholder="全部分组"
            allowClear
            style={{ width: 200 }}
            value={filterGroup}
            onChange={(v) => {
              setFilterGroup(v)
              setPage(1)
            }}
            options={groups.map((g) => ({ value: g.group, label: g.description ?? g.group }))}
          />
          <Input.Search
            placeholder="搜索 key…"
            allowClear
            style={{ width: 220 }}
            onSearch={(v) => {
              setFilterKey(v)
              setPage(1)
            }}
          />
          <Select
            placeholder="全部状态"
            allowClear
            style={{ width: 140 }}
            value={filterStatus}
            onChange={(v) => {
              setFilterStatus(v)
              setPage(1)
            }}
            options={STATUS_OPTIONS}
          />
        </Space>
      }
    >
      <Table<SettingItem>
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        tableLayout="fixed"
        scroll={tableScrollX(columns)}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Drawer
        title={editing ? '编辑配置项' : '新增配置项'}
        open={drawerOpen}
        width={520}
        onClose={() => setDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={submitLoading} onClick={() => void handleSubmit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item label="分组" name="group" rules={[{ required: true, message: '请选择分组' }]}>
            <Select
              disabled={Boolean(editing)}
              placeholder="选择分组"
              options={groups.map((g) => ({ value: g.group, label: g.description ?? g.group }))}
            />
          </Form.Item>
          <Form.Item
            label="Key"
            name="key"
            rules={[{ required: true, message: '请输入 key' }]}
            extra={editing ? 'Key 创建后不可修改' : undefined}
          >
            <Input disabled={Boolean(editing)} placeholder="如 mail.host" />
          </Form.Item>
          <Form.Item label="值类型" name="valueType" rules={[{ required: true, message: '请选择值类型' }]}>
            <Select disabled={Boolean(editing)} options={VALUE_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item
            label="值"
            name="value"
            valuePropName={watchedValueType === 'bool' ? 'checked' : 'value'}
            rules={editing ? [] : [{ required: true, message: '请输入值' }]}
            extra={isSecretEditing ? '留空或保持掩码表示不修改' : undefined}
          >
            {renderValueControl()}
          </Form.Item>
          <Form.Item label="加密存储" name="encrypted" valuePropName="checked">
            <Switch disabled={Boolean(editing)} />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true }]}>
            <Select options={STATUS_OPTIONS} />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} placeholder="可选" />
          </Form.Item>
        </Form>
      </Drawer>
    </PageContainer>
  )
}
