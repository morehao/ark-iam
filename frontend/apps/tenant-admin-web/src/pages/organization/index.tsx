import { useCallback, useEffect, useState } from 'react'
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Spin, Switch, Table, Tag, Tree, message } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import { IDCell, PageContainer } from '@ark-iam/ui'
import {
  createOrganization,
  createOrganizationUser,
  deleteOrganization,
  deleteOrganizationUser,
  getOrganizationTree,
  getOrganizationUserPage,
  updateOrganization,
  updateOrganizationStatus,
} from '../../api/organization'
import type { OrganizationItem, OrganizationUserItem } from '@ark-iam/types'

const RELATION_TEXT: Record<string, string> = { member: '成员', leader: '负责人' }

function buildTree(list: OrganizationItem[]): DataNode[] {
  return list.map((n) => ({
    key: n.organizationID,
    title: `${n.name}${n.status === 'inactive' ? '（停用）' : ''}`,
    children: n.children?.length ? buildTree(n.children) : undefined,
  }))
}

export default function OrganizationPage() {
  const [tree, setTree] = useState<DataNode[]>([])
  const [orgList, setOrgList] = useState<OrganizationItem[]>([])
  const [selectedID, setSelectedID] = useState<string>('')
  const [loading, setLoading] = useState(false)

  const [members, setMembers] = useState<OrganizationUserItem[]>([])
  const [memberLoading, setMemberLoading] = useState(false)

  const [nodeModalOpen, setNodeModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<OrganizationItem | null>(null)
  const [nodeForm] = Form.useForm()

  const [memberModalOpen, setMemberModalOpen] = useState(false)
  const [memberForm] = Form.useForm()

  const loadTree = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationTree()
      setOrgList(resp.list || [])
      setTree(buildTree(resp.list || []))
      if (!selectedID && resp.list?.length) {
        const first = resp.list[0]
        setSelectedID(first.organizationID)
      }
    } finally {
      setLoading(false)
    }
  }, [selectedID])

  useEffect(() => {
    void loadTree()
  }, [loadTree])

  const loadMembers = useCallback(async () => {
    if (!selectedID) return
    setMemberLoading(true)
    try {
      const resp = await getOrganizationUserPage(selectedID, { page: 1, pageSize: 100 })
      setMembers(resp.list || [])
    } finally {
      setMemberLoading(false)
    }
  }, [selectedID])

  useEffect(() => {
    void loadMembers()
  }, [loadMembers])

  const openCreateNode = (parentID?: string) => {
    setEditingNode(null)
    nodeForm.resetFields()
    if (parentID) nodeForm.setFieldsValue({ parentID })
    setNodeModalOpen(true)
  }

  const openEditNode = (node: OrganizationItem) => {
    setEditingNode(node)
    nodeForm.setFieldsValue({ name: node.name, code: node.code, sort: node.sort, status: node.status })
    setNodeModalOpen(true)
  }

  const submitNode = async () => {
    try {
      const values = await nodeForm.validateFields()
      if (editingNode) {
        await updateOrganization({ organizationID: editingNode.organizationID, ...values })
        message.success('保存成功')
      } else {
        await createOrganization(values)
        message.success('创建成功')
      }
      setNodeModalOpen(false)
      void loadTree()
    } catch {
      /* 校验或请求失败 */
    }
  }

  const removeNode = async (id: string, cascade: boolean) => {
    try {
      await deleteOrganization(id, cascade)
      message.success('删除成功')
      setSelectedID('')
      void loadTree()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const toggleStatus = async (node: OrganizationItem) => {
    const next = node.status === 'active' ? 'inactive' : 'active'
    await updateOrganizationStatus(node.organizationID, next)
    message.success('状态已更新')
    void loadTree()
  }

  const submitMember = async () => {
    try {
      const values = await memberForm.validateFields()
      await createOrganizationUser(selectedID, { ...values, relationType: values.relationType || 'member' })
      message.success('添加成功')
      setMemberModalOpen(false)
      memberForm.resetFields()
      void loadMembers()
    } catch {
      /* 校验或请求失败 */
    }
  }

  const removeMember = async (userID: string) => {
    await deleteOrganizationUser(selectedID, userID)
    message.success('移除成功')
    void loadMembers()
  }

  const memberColumns: ColumnsType<OrganizationUserItem> = [
    { title: '用户ID', dataIndex: 'userID', key: 'userID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '姓名', dataIndex: 'userName', key: 'userName' },
    { title: '关系', dataIndex: 'relationType', key: 'relationType', width: 90, render: (v: string) => <Tag color={v === 'leader' ? 'geekblue' : 'blue'}>{RELATION_TEXT[v] || v}</Tag> },
    { title: '主归属', dataIndex: 'isPrimary', key: 'isPrimary', width: 90, render: (v: boolean) => (v ? <Tag color="gold">主</Tag> : <Tag>否</Tag>) },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, r) => (
        <Space size={4}>
          <Popconfirm title="确认移除该关系？" onConfirm={() => void removeMember(r.userID)}>
            <Button type="link" size="small" danger>
              移除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="组织架构"
      description="租户下的组织树与成员关系（成员/负责人，主归属唯一）"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadTree()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreateNode()}>
            新建根组织
          </Button>
        </Space>
      }
    >
      <Space align="start" size={16} style={{ width: '100%' }}>
        <Card title="组织树" style={{ width: 360, borderRadius: 12 }} styles={{ body: { maxHeight: 640, overflow: 'auto' } }}>
          <Spin spinning={loading}>
            {tree.length === 0 ? (
              <Button type="dashed" block icon={<PlusOutlined />} onClick={() => openCreateNode()}>
                创建组织
              </Button>
            ) : (
              <Tree
                treeData={tree}
                selectedKeys={selectedID ? [selectedID] : []}
                onSelect={(keys) => keys.length && setSelectedID(String(keys[0]))}
                defaultExpandAll
                blockNode
              />
            )}
          </Spin>
        </Card>

        <Card
          title="节点操作与成员"
          style={{ flex: 1, borderRadius: 12 }}
          extra={
            selectedID && (
              <Space>
                <Button size="small" onClick={() => openCreateNode(selectedID)}>
                  新建子组织
                </Button>
                <Button size="small" onClick={() => openEditNode(findNode(orgList, selectedID)!)}>
                  编辑
                </Button>
                <Popconfirm title="停用/启用该节点？" onConfirm={() => void toggleStatus(findNode(orgList, selectedID)!)}>
                  <Button size="small">{findNode(orgList, selectedID)?.status === 'active' ? '停用' : '启用'}</Button>
                </Popconfirm>
                <Popconfirm title="确认删除？有子节点/成员需先移除" onConfirm={() => void removeNode(selectedID, false)}>
                  <Button size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            )
          }
        >
          {!selectedID ? (
            <div style={{ padding: 40, textAlign: 'center', color: '#94a3b8' }}>请先选择左侧组织节点</div>
          ) : (
            <>
              <Space style={{ marginBottom: 12 }}>
                <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setMemberModalOpen(true)}>
                  添加成员/负责人
                </Button>
              </Space>
              <Table<OrganizationUserItem>
                rowKey={(r) => `${r.organizationID}-${r.userID}-${r.relationType}`}
                loading={memberLoading}
                columns={memberColumns}
                dataSource={members}
                pagination={false}
                size="small"
              />
            </>
          )}
        </Card>
      </Space>

      <Modal title={editingNode ? '编辑组织' : '新建组织'} open={nodeModalOpen} onOk={() => void submitNode()} onCancel={() => setNodeModalOpen(false)} destroyOnClose>
        <Form form={nodeForm} layout="vertical">
          <Form.Item name="parentID" label="父组织" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="name" label="组织名称" rules={[{ required: true, message: '请输入组织名称' }]}>
            <Input placeholder="如：产品研发部" />
          </Form.Item>
          <Form.Item name="code" label="组织编码">
            <Input placeholder="可空，外部系统同步用" />
          </Form.Item>
          <Form.Item name="sort" label="同级排序" initialValue={0}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select
              options={[
                { label: '启用', value: 'active' },
                { label: '停用', value: 'inactive' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="添加成员/负责人" open={memberModalOpen} onOk={() => void submitMember()} onCancel={() => setMemberModalOpen(false)} destroyOnClose>
        <Form form={memberForm} layout="vertical">
          <Form.Item name="userID" label="用户ID" rules={[{ required: true, message: '请输入用户ID' }]}>
            <Input placeholder="租户内用户ID" />
          </Form.Item>
          <Form.Item name="relationType" label="关系类型" initialValue="member">
            <Select
              options={[
                { label: '成员（归属）', value: 'member' },
                { label: '负责人', value: 'leader' },
              ]}
            />
          </Form.Item>
          <Form.Item name="isPrimary" label="主归属" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

// findNode 在树列表中按 ID 查找节点
function findNode(list: OrganizationItem[], id: string): OrganizationItem | null {
  for (const n of list) {
    if (n.organizationID === id) return n
    if (n.children?.length) {
      const found = findNode(n.children, id)
      if (found) return found
    }
  }
  return null
}
