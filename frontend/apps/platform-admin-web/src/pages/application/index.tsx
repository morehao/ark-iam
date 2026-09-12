import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Form, Input, InputNumber, message, Modal, Select, Space, Table } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, CODE_COL_WIDTH, fmtTime, IDCell, idColumn, nameColumn, PageContainer, STATUS_COL_WIDTH, StatusTag, tableScrollX, TAG_COL_WIDTH, textColumn, timeColumn, TypeTag } from '@ark-iam/ui'
import { createApplication, deleteApplication, getApplicationDetail, getApplicationPageList, updateApplication } from '@ark-iam/api'
import type { ApplicationItem } from '@ark-iam/types'

export default function ApplicationList() {
  const [data, setData] = useState<ApplicationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<ApplicationItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<ApplicationItem | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApplicationPageList({ page, pageSize, name: keyword })
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
    form.setFieldsValue({ type: 'first_party', sort: 0 })
    setModalOpen(true)
  }

  const handleEdit = (record: ApplicationItem) => {
    setEditing(record)
    form.setFieldsValue({
      code: record.code,
      name: record.name,
      type: record.type,
      status: record.status,
      description: record.description,
      logoUrl: record.logoUrl,
      homepageUrl: record.homepageUrl,
      sort: record.sort,
    })
    setModalOpen(true)
  }

  const handleOpenDetail = async (record: ApplicationItem) => {
    setDetail(record)
    setDetailOpen(true)
    try {
      const resp = await getApplicationDetail(record.appID)
      setDetail(resp)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateApplication({ appID: editing.appID, ...values })
        message.success('修改成功')
      } else {
        await createApplication(values)
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

  const handleDelete = async (record: ApplicationItem) => {
    try {
      await deleteApplication(record.appID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<ApplicationItem> = [
    idColumn<ApplicationItem>({ dataIndex: 'appID' }),
    nameColumn<ApplicationItem>({
      title: '应用名',
      dataIndex: 'name',
      onClick: (r) => void handleOpenDetail(r),
    }),
    textColumn<ApplicationItem>({ title: '编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
    { title: '类型', dataIndex: 'type', key: 'type', width: TAG_COL_WIDTH, render: (v: string) => <TypeTag value={v} /> },
    { title: '状态', dataIndex: 'status', key: 'status', width: STATUS_COL_WIDTH, render: (v: string) => <StatusTag value={v} /> },
    timeColumn<ApplicationItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<ApplicationItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<ApplicationItem>({
      max: 2,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
        { key: 'delete', label: '删除', danger: true, confirm: '确认删除该应用？', onClick: () => void handleDelete(r) },
      ],
    }),
  ]

  return (
    <PageContainer
      title="应用管理"
      description="接入平台的应用"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建应用
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按应用名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>
      <Table<ApplicationItem>
        rowKey="appID"
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

      <Modal
        title={editing ? '编辑应用' : '新建应用'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="code" label="应用编码" rules={[{ required: true, message: '请输入应用编码' }]}>
            <Input placeholder="唯一编码，如 iam-web" disabled={!!editing} />
          </Form.Item>
          <Form.Item name="name" label="应用名称" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="应用名称" />
          </Form.Item>
          <Form.Item name="type" label="应用类型">
            <Select
              options={[
                { value: 'first_party', label: '第一方' },
                { value: 'third_party', label: '第三方' },
              ]}
            />
          </Form.Item>
          {editing && (
            <Form.Item name="status" label="状态">
              <Select
                options={[
                  { value: 'enable', label: '启用' },
                  { value: 'disable', label: '停用' },
                ]}
              />
            </Form.Item>
          )}
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="选填" />
          </Form.Item>
          <Form.Item name="logoUrl" label="Logo 地址">
            <Input placeholder="https://... 选填" />
          </Form.Item>
          <Form.Item name="homepageUrl" label="首页地址">
            <Input placeholder="https://... 选填" />
          </Form.Item>
          <Form.Item name="sort" label="排序">
            <InputNumber style={{ width: '100%' }} placeholder="数字越小越靠前" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={detail ? `应用详情 - ${detail.name || ''}` : '应用详情'}
        width={560}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
      >
        {detail && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="应用ID"><IDCell value={detail.appID} /></Descriptions.Item>
            <Descriptions.Item label="编码">{detail.code || '-'}</Descriptions.Item>
            <Descriptions.Item label="名称">{detail.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="类型">
              <TypeTag value={detail.type} />
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <StatusTag value={detail.status} />
            </Descriptions.Item>
            <Descriptions.Item label="描述">{detail.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="Logo 地址">{detail.logoUrl || '-'}</Descriptions.Item>
            <Descriptions.Item label="首页地址">{detail.homepageUrl || '-'}</Descriptions.Item>
            <Descriptions.Item label="排序">{detail.sort ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{fmtTime(detail.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="个人自助创建租户">{detail.allowPersonCreateTenant ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="允许邀请加入租户">{detail.allowJoinByInvite ? '是' : '否'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </PageContainer>
  )
}
