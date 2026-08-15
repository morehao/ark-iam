import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, Modal, Popconfirm, Space, Switch, Table, Tag, message } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { IDCell, PageContainer } from '@ark-iam/ui'
import { createResource, deleteResource, getResourceDetail, getResourcePageList, updateResource } from '@ark-iam/api'
import type { ResourceItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

export default function ResourceList() {
  const [data, setData] = useState<ResourceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ResourceItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getResourcePageList({ page, pageSize, name: keyword })
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

  const handleEdit = (record: ResourceItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      indicator: record.indicator,
      isDefault: record.isDefault,
      accessTokenTtl: record.accessTokenTtl,
    })
    setModalOpen(true)
    getResourceDetail(record.resourceID)
      .then((detail) => {
        form.setFieldsValue({
          name: detail.name,
          indicator: detail.indicator,
          isDefault: detail.isDefault,
          accessTokenTtl: detail.accessTokenTtl,
        })
      })
      .catch(() => {
        /* 详情拉取失败，回退行数据 */
      })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateResource({ resourceID: editing.resourceID, ...values })
        message.success('修改成功')
      } else {
        await createResource(values)
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

  const columns: ColumnsType<ResourceItem> = [
    { title: 'ID', dataIndex: 'resourceID', key: 'resourceID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '资源名', dataIndex: 'name', key: 'name', width: 180 },
    {
      title: '标识符',
      dataIndex: 'indicator',
      key: 'indicator',
      width: 200,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v || '-'}</span>,
    },
    {
      title: '默认',
      dataIndex: 'isDefault',
      key: 'isDefault',
      width: 90,
      render: (v: number) => (v === 1 ? <Tag color="blue">默认</Tag> : <Tag>非默认</Tag>),
    },
    {
      title: 'AccessTokenTTL',
      dataIndex: 'accessTokenTtl',
      key: 'accessTokenTtl',
      width: 160,
      render: (v: number) => `${v}s`,
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
      width: 120,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该资源？"
            onConfirm={async () => {
              try {
                await deleteResource(r.resourceID)
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
      title="资源管理"
      description="管理 IAM 资源、标识符与访问令牌有效期"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建资源
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按资源名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<ResourceItem>
        rowKey="resourceID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 900 }}
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
        title={editing ? '编辑资源' : '新建资源'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="资源名" rules={[{ required: true, message: '请输入资源名' }]}>
            <Input placeholder="如：用户服务" />
          </Form.Item>
          <Form.Item name="indicator" label="标识符" rules={[{ required: true, message: '请输入标识符' }]}>
            <Input placeholder="如：iam:user" />
          </Form.Item>
          <Form.Item
            name="isDefault"
            label="默认资源"
            valuePropName="checked"
            getValueFromEvent={(c: boolean) => (c ? 1 : 0)}
            getValueProps={(v?: number) => ({ checked: v === 1 })}
            initialValue={0}
          >
            <Switch checkedChildren="默认" unCheckedChildren="非默认" />
          </Form.Item>
          <Form.Item
            name="accessTokenTtl"
            label="AccessTokenTTL（秒）"
            rules={[{ required: true, message: '请输入 AccessTokenTTL' }]}
            initialValue={3600}
          >
            <InputNumber style={{ width: '100%' }} placeholder="默认 3600" min={1} precision={0} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
