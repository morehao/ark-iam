import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, Modal, Popconfirm, Space, Table, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, fmtTime, IDCell, PageContainer } from '@ark-iam/ui'
import {
  createSystemConfig,
  deleteSystemConfig,
  getSystemDetail,
  getSystemPageList,
  updateSystemConfig,
} from '@ark-iam/api'
import type { SystemConfigItem } from '@ark-iam/types'

/** 序列化配置值：对象 JSON 化，其余转字符串 */
function formatValue(value: unknown): string {
  if (value == null) return '-'
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value) ?? String(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

interface SystemFormValues {
  key: string
  value?: string
}

export default function SystemConfigList() {
  const [data, setData] = useState<SystemConfigItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<SystemConfigItem | null>(null)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [form] = Form.useForm<SystemFormValues>()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getSystemPageList({ page, pageSize, key: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = async (record: SystemConfigItem) => {
    try {
      const detail = await getSystemDetail(record.systemID)
      setEditing(detail)
      form.setFieldsValue({
        key: detail.key,
        value:
          typeof detail.value === 'object' && detail.value !== null
            ? (JSON.stringify(detail.value, null, 2) ?? '')
            : String(detail.value ?? ''),
      })
      setModalOpen(true)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      const raw = values.value?.trim() ?? ''
      let parsed: unknown = raw
      if (raw) {
        try {
          parsed = JSON.parse(raw)
        } catch {
          parsed = raw
        }
      }
      setSubmitLoading(true)
      if (editing) {
        await updateSystemConfig({ systemID: editing.systemID, value: parsed })
        message.success('修改成功')
      } else {
        await createSystemConfig({ key: values.key, value: parsed })
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const columns: ColumnsType<SystemConfigItem> = [
    { title: 'ID', dataIndex: 'systemID', key: 'systemID', width: 150, render: (v: string) => <IDCell value={v} /> },
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
      width: 220,
      render: (v: string) => <EllipsisCell value={v} monospace />,
    },
    {
      title: '配置值',
      dataIndex: 'value',
      key: 'value',
      render: (_, r) => <EllipsisCell value={formatValue(r.value)} monospace limit={40} />,
    },
    {
      title: '创建时间',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime(r.createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => void handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该配置？"
            onConfirm={async () => {
              try {
                await deleteSystemConfig(r.systemID)
                message.success('删除成功')
                void fetchData()
              } catch {
                /* 拦截器已提示 */
              }
            }}
          >
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="系统配置"
      description="租户级键值配置管理"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建配置
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按配置键搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>
      <Table<SystemConfigItem>
        rowKey="systemID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 800 }}
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

      <Modal
        title={editing ? '编辑配置' : '新建配置'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="key" label="配置键" rules={[{ required: true, message: '请输入配置键' }]}>
            <Input placeholder="如 system.xxx" />
          </Form.Item>
          <Form.Item name="value" label="配置值">
            <Input.TextArea rows={3} placeholder="JSON 或普通文本" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
