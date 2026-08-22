import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tree,
  TreeSelect,
  message,
} from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import { PageContainer } from '@ark-iam/ui'
import type { OrganizationItem } from '@ark-iam/types'
import {
  createOrganization,
  deleteOrganization,
  getOrganizationTree,
  updateOrganization,
  updateOrganizationStatus,
} from '../../api/organization'

function buildTree(list: OrganizationItem[]): DataNode[] {
  return list.map((n) => ({
    key: n.organizationID,
    title: `${n.name}${n.status === 'inactive' ? '（停用）' : ''}`,
    children: n.children?.length ? buildTree(n.children) : undefined,
  }))
}

// 收集目标节点及其全部子孙 ID（移动/删除时排除自身与后代）
function collectSelfAndDescendants(list: OrganizationItem[], id: string): Set<string> {
  const result = new Set<string>()
  const node = findNode(list, id)
  if (!node) return result
  const collect = (n: OrganizationItem) => {
    result.add(n.organizationID)
    if (n.children?.length) n.children.forEach(collect)
  }
  collect(node)
  return result
}

export default function OrganizationPage() {
  const [tree, setTree] = useState<DataNode[]>([])
  const [orgList, setOrgList] = useState<OrganizationItem[]>([])
  const [selectedID, setSelectedID] = useState<string>('')
  const [loading, setLoading] = useState(false)

  const [nodeModalOpen, setNodeModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<OrganizationItem | null>(null)
  const [nodeForm] = Form.useForm()

  const [detailOpen, setDetailOpen] = useState(false)
  const [detailNode, setDetailNode] = useState<OrganizationItem | null>(null)

  const loadTree = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationTree()
      setOrgList(resp.list || [])
      setTree(buildTree(resp.list || []))
      if (!selectedID && resp.list?.length) {
        setSelectedID(resp.list[0].organizationID)
      }
    } finally {
      setLoading(false)
    }
  }, [selectedID])

  useEffect(() => {
    void loadTree()
  }, [loadTree])

  const selectedNode = useMemo(() => (selectedID ? findNode(orgList, selectedID) : null), [orgList, selectedID])

  // 节点信息：面包屑 + 子节点数
  const ancestors = useMemo(() => {
    if (!selectedNode?.orgPath) return []
    const ids = selectedNode.orgPath.split('/').filter(Boolean)
    const nameMap = new Map<string, string>()
    const walk = (items: OrganizationItem[]) => {
      for (const n of items) {
        nameMap.set(n.organizationID, n.name)
        if (n.children?.length) walk(n.children)
      }
    }
    walk(orgList)
    return ids.map((id) => ({ id, name: nameMap.get(id) || id }))
  }, [selectedNode, orgList])

  const childCount = useMemo(() => selectedNode?.children?.length ?? 0, [selectedNode])

  const openCreateNode = (parentID?: string) => {
    setEditingNode(null)
    nodeForm.resetFields()
    if (parentID) nodeForm.setFieldsValue({ parentID })
    setNodeModalOpen(true)
  }

  const openEditNode = (node: OrganizationItem) => {
    setEditingNode(node)
    nodeForm.setFieldsValue({
      parentID: node.parentID || undefined,
      name: node.name,
      code: node.code,
      sort: node.sort,
      status: node.status,
    })
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

  const removeNode = async (cascade: boolean) => {
    if (!selectedID) return
    try {
      await deleteOrganization(selectedID, cascade)
      message.success('删除成功')
      setSelectedID('')
      void loadTree()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const toggleStatus = async () => {
    if (!selectedNode) return
    const next = selectedNode.status === 'active' ? 'inactive' : 'active'
    await updateOrganizationStatus(selectedNode.organizationID, next)
    message.success('状态已更新')
    void loadTree()
  }

  const parentTreeData = useMemo(() => {
    if (!selectedID) return []
    const exclude = collectSelfAndDescendants(orgList, selectedID)
    const toTree = (items: OrganizationItem[]): any[] =>
      items
        .filter((n) => !exclude.has(n.organizationID))
        .map((n) => ({
          title: n.name,
          value: n.organizationID,
          disabled: n.organizationID === selectedID,
          children: n.children?.length ? toTree(n.children) : undefined,
        }))
    return toTree(orgList)
  }, [orgList, selectedID])

  // 详情弹窗中目标节点的路径面包屑
  const detailAncestors = useMemo(() => {
    if (!detailNode?.orgPath) return []
    const ids = detailNode.orgPath.split('/').filter(Boolean)
    const nameMap = new Map<string, string>()
    const walk = (items: OrganizationItem[]) => {
      for (const n of items) {
        nameMap.set(n.organizationID, n.name)
        if (n.children?.length) walk(n.children)
      }
    }
    walk(orgList)
    return ids.map((id) => ({ id, name: nameMap.get(id) || id }))
  }, [detailNode, orgList])

  // 直属子组织列表列（点击行进入下级；操作需阻止行点击冒泡）
  const childColumns: ColumnsType<OrganizationItem> = [
    { title: '组织名称', dataIndex: 'name', key: 'name' },
    { title: '组织编码', dataIndex: 'code', key: 'code', render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (v: string) => (v === 'active' ? <Tag color="green">启用</Tag> : <Tag color="orange">停用</Tag>),
    },
    { title: '同级排序', dataIndex: 'sort', key: 'sort', width: 90, render: (v?: number) => v ?? 0 },
    {
      title: '子组织数',
      key: 'childCount',
      width: 90,
      render: (_, r) => r.children?.length ?? 0,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_, r) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation()
            setDetailNode(r)
            setDetailOpen(true)
          }}
        >
          详情
        </Button>
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
        <Card title="组织树" style={{ width: 260, borderRadius: 12 }} styles={{ body: { maxHeight: 680, overflow: 'auto' } }}>
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

        <Card style={{ flex: 1, borderRadius: 12 }} styles={{ body: { padding: '16px 24px 24px' } }}>
          {!selectedID || !selectedNode ? (
            <div style={{ padding: 60, textAlign: 'center', color: '#94a3b8' }}>请先选择左侧组织节点</div>
          ) : (
            <>
              <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }} align="start" wrap>
                <Space direction="vertical" size={4}>
                  <Space align="center" wrap>
                    <span style={{ fontSize: 18, fontWeight: 600 }}>{selectedNode.name}</span>
                    {selectedNode.status === 'active' ? <Tag color="green">启用</Tag> : <Tag color="orange">停用</Tag>}
                    <Tag color="blue">子组织 {childCount}</Tag>
                  </Space>
                  <Space size={4} wrap style={{ color: '#94a3b8', fontSize: 13 }}>
                    {ancestors.map((a, i) => (
                      <span key={a.id}>
                        {i > 0 && <span style={{ margin: '0 4px' }}>/</span>}
                        {a.name}
                      </span>
                    ))}
                  </Space>
                </Space>
                <Space wrap>
                  <Button size="small" icon={<PlusOutlined />} onClick={() => openCreateNode(selectedID)}>
                    新建子组织
                  </Button>
                  <Button size="small" onClick={() => openEditNode(selectedNode)}>
                    编辑 / 移动
                  </Button>
                  <Popconfirm title={selectedNode.status === 'active' ? '停用该节点？' : '启用该节点？'} onConfirm={() => void toggleStatus()}>
                    <Button size="small">{selectedNode.status === 'active' ? '停用' : '启用'}</Button>
                  </Popconfirm>
                  <Popconfirm title="确认删除该节点？有子组织/成员需勾选级联" onConfirm={() => void removeNode(false)} okText="仅删本节点" cancelText="取消">
                    <Button size="small" danger>
                      删除
                    </Button>
                  </Popconfirm>
                  <Popconfirm title="级联删除该节点及全部子组织/成员？" onConfirm={() => void removeNode(true)}>
                    <Button size="small" danger>
                      级联删除
                    </Button>
                  </Popconfirm>
                </Space>
              </Space>

              <Table<OrganizationItem>
                rowKey={(r) => r.organizationID}
                size="middle"
                columns={childColumns}
                dataSource={selectedNode.children || []}
                pagination={false}
                locale={{ emptyText: '暂无下级组织' }}
                onRow={(record) => ({
                  onClick: () => setSelectedID(record.organizationID),
                  style: { cursor: 'pointer' },
                })}
              />
            </>
          )}
        </Card>
      </Space>

      <Modal
        title="组织详情"
        open={detailOpen}
        onOk={() => setDetailOpen(false)}
        onCancel={() => setDetailOpen(false)}
        footer={<Button onClick={() => setDetailOpen(false)}>关闭</Button>}
      >
        {detailNode && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="组织名称">{detailNode.name}</Descriptions.Item>
            <Descriptions.Item label="组织编码">{detailNode.code || '-'}</Descriptions.Item>
            <Descriptions.Item label="同级排序">{detailNode.sort ?? 0}</Descriptions.Item>
            <Descriptions.Item label="子组织数">{detailNode.children?.length ?? 0}</Descriptions.Item>
            <Descriptions.Item label="状态">{detailNode.status === 'active' ? '启用' : '停用'}</Descriptions.Item>
            <Descriptions.Item label="路径">
              <Space size={4} wrap>
                {detailAncestors.map((da, i) => (
                  <span key={da.id}>
                    {i > 0 && <span style={{ margin: '0 4px', color: '#94a3b8' }}>/</span>}
                    {da.name}
                  </span>
                ))}
              </Space>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
      

      <Modal title={editingNode ? '编辑组织（可改父组织实现移动）' : '新建组织'} open={nodeModalOpen} onOk={() => void submitNode()} onCancel={() => setNodeModalOpen(false)} destroyOnClose>
        <Form form={nodeForm} layout="vertical">
          {editingNode && (
            <Form.Item name="parentID" label="父组织（不选为根节点；移动会级联更新子组织路径）">
              <TreeSelect allowClear treeDefaultExpandAll treeData={parentTreeData} placeholder="选择父组织" />
            </Form.Item>
          )}
          {!editingNode && (
            <Form.Item name="parentID" label="父组织（不选为根节点）">
              <TreeSelect allowClear treeDefaultExpandAll treeData={orgList.length ? toTreeSelect(orgList) : []} placeholder="选择父组织" />
            </Form.Item>
          )}
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

// toTreeSelect 组织树 -> TreeSelect 数据
function toTreeSelect(list: OrganizationItem[]): any[] {
  return list.map((n) => ({
    title: n.name,
    value: n.organizationID,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }))
}
