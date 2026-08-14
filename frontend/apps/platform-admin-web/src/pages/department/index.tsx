import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, InputNumber, message, Modal, Popconfirm, Space, Table, Typography } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { createDepartment, deleteDepartment, getDepartmentTree, updateDepartment } from '@ark-iam/api'
import type { DepartmentItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

export default function DepartmentList() {
  const [data, setData] = useState<DepartmentItem[]>([])
  const [loading, setLoading] = useState(false)
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<DepartmentItem | null>(null)
  const [parentID, setParentID] = useState(0)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const collectKeys = (list: DepartmentItem[]): React.Key[] => {
    const keys: React.Key[] = []
    const walk = (nodes: DepartmentItem[]) => {
      nodes.forEach((n) => {
        keys.push(n.departmentID)
        if (n.children?.length) walk(n.children)
      })
    }
    walk(list)
    return keys
  }

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getDepartmentTree()
      const list = resp?.list || []
      setData(list)
      // 树形表格默认展开所有节点
      setExpandedRowKeys(collectKeys(list))
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const openCreate = (parentID: number) => {
    setEditing(null)
    setParentID(parentID)
    form.resetFields()
    form.setFieldsValue({ parentID, sort: 0 })
    setModalOpen(true)
  }

  const handleEdit = (record: DepartmentItem) => {
    setEditing(record)
    setParentID(record.parentID)
    form.setFieldsValue({
      parentID: record.parentID,
      name: record.name,
      code: record.code,
      sort: record.sort,
      leaderUserID: record.leaderUserID,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateDepartment({ departmentID: editing.departmentID, ...values })
        message.success('修改成功')
      } else {
        await createDepartment(values)
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

  const handleDelete = async (record: DepartmentItem) => {
    try {
      await deleteDepartment(record.departmentID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<DepartmentItem> = [
    { title: '部门ID', dataIndex: 'departmentID', key: 'departmentID', width: 90 },
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
      width: 280,
      render: (v: string, r) => (
        <Space size={8}>
          <Typography.Text strong>{v || '-'}</Typography.Text>
          {r.code ? (
            <Typography.Text type="secondary" style={{ fontSize: 12, fontFamily: 'monospace' }}>
              ({r.code})
            </Typography.Text>
          ) : null}
        </Space>
      ),
    },
    { title: '部门编码', dataIndex: 'code', key: 'code', width: 160, render: (v: string) => v || '-' },
    { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
    { title: '负责人用户ID', dataIndex: 'leaderUserID', key: 'leaderUserID', width: 130, render: (v: number) => v || '-' },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v?: number) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 240,
      render: (_, r) => (
        <Space size={0} onClick={(e) => e.stopPropagation()}>
          <Button type="link" size="small" onClick={() => openCreate(r.departmentID)}>
            新增子部门
          </Button>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除该部门？"
            description="删除后其下所有子部门将级联删除。"
            onConfirm={() => void handleDelete(r)}
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
      title="部门管理"
      description="组织架构树形管理"
      loading={loading}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate(0)}>
            新建根部门
          </Button>
        </Space>
      }
    >
      <Table<DepartmentItem>
        rowKey="departmentID"
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={false}
        scroll={{ x: 1150 }}
        expandable={{
          expandedRowKeys,
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys as React.Key[]),
        }}
      />

      <Modal
        title={editing ? '编辑部门' : modalParentTitle(parentID)}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="parentID" label="上级部门ID">
            <InputNumber style={{ width: '100%' }} disabled />
          </Form.Item>
          <Form.Item name="name" label="部门名称" rules={[{ required: true, message: '请输入部门名称' }]}>
            <Input placeholder="如：研发部" />
          </Form.Item>
          <Form.Item name="code" label="部门编码">
            <Input placeholder="选填，如 dev" />
          </Form.Item>
          <Form.Item name="sort" label="排序">
            <InputNumber style={{ width: '100%' }} placeholder="数字越小越靠前" />
          </Form.Item>
          <Form.Item name="leaderUserID" label="负责人用户ID">
            <InputNumber style={{ width: '100%' }} placeholder="选填" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

function modalParentTitle(parentID: number) {
  return parentID === 0 ? '新建根部门' : '新建子部门'
}
