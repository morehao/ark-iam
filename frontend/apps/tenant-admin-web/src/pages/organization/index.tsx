import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tree,
  TreeSelect,
  message,
} from 'antd'
import { PlusOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import { fmtTime, PageContainer, StatusTag, tokens } from '@ark-iam/ui'
import type { OrganizationChildItem, OrganizationItem } from '@ark-iam/types'
import {
  createOrganization,
  deleteOrganization,
  getOrganizationChildren,
  getOrganizationTree,
  updateOrganization,
  updateOrganizationStatus,
} from '../../api/organization'

// 树节点：纯组织名称
function buildTree(list: OrganizationItem[]): DataNode[] {
  return list.map((n) => ({
    key: n.organizationID,
    title: n.name,
    children: n.children?.length ? buildTree(n.children) : undefined,
  }))
}

// 按关键字在树上过滤节点（命中节点保留其祖先链）
function filterTree(list: OrganizationItem[], keyword: string): OrganizationItem[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  const hit = (n: OrganizationItem) => n.name.toLowerCase().includes(kw)
  const walk = (items: OrganizationItem[]): OrganizationItem[] => {
    const out: OrganizationItem[] = []
    for (const n of items) {
      const children = n.children?.length ? walk(n.children) : []
      if (hit(n) || children.length) {
        out.push({ ...n, children })
      }
    }
    return out
  }
  return walk(list)
}

// 收集目标节点及其全部子孙 ID
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

export default function OrganizationPage() {
  const [orgList, setOrgList] = useState<OrganizationItem[]>([])
  // 左侧树当前被选中的组织；为空表示未选中，右侧展示顶级组织的下级组织
  const [selectedID, setSelectedID] = useState<string>('')
  const [treeLoading, setTreeLoading] = useState(false)

  // 左侧树搜索关键字（仅影响树展示）
  const [treeKeyword, setTreeKeyword] = useState('')

  // ---------- 右侧下级组织列表 ----------
  const [childrenList, setChildrenList] = useState<OrganizationChildItem[]>([])
  const [childrenTotal, setChildrenTotal] = useState(0)
  const [childrenLoading, setChildrenLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [filterForm] = Form.useForm()
  const [query, setQuery] = useState<{ name?: string; status?: string }>({})

  // 顶级组织（根节点）：未选中任何组织时，右侧默认展示其下级组织
  const defaultRootID = useMemo(() => orgList[0]?.organizationID || '', [orgList])

  const [nodeModalOpen, setNodeModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<{ organizationID: string; parentID?: string; name: string; code?: string; sort?: number; status: string } | null>(null)
  const [nodeForm] = Form.useForm()

  // 左侧树加载
  const loadTree = useCallback(async () => {
    setTreeLoading(true)
    try {
      const resp = await getOrganizationTree()
      setOrgList(resp.list || [])
    } finally {
      setTreeLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadTree()
  }, [loadTree])

  // 右侧下级组织列表加载（入参：选中组织或顶级组织 + 筛选 + 分页）
  const loadChildren = useCallback(
    async (p: number) => {
      const parentID = selectedID || defaultRootID
      if (!parentID) {
        setChildrenList([])
        setChildrenTotal(0)
        return
      }
      setChildrenLoading(true)
      try {
        const resp = await getOrganizationChildren(parentID, {
          page: p,
          pageSize,
          name: query.name,
          status: query.status,
        })
        setChildrenList(resp.list || [])
        setChildrenTotal(resp.total || 0)
      } finally {
        setChildrenLoading(false)
      }
    },
    [selectedID, defaultRootID, query],
  )

  useEffect(() => {
    setPage(1)
    void loadChildren(1)
  }, [loadChildren])

  const visibleTree = useMemo(() => filterTree(orgList, treeKeyword), [orgList, treeKeyword])
  const visibleTreeData = useMemo(() => buildTree(visibleTree), [visibleTree])

  const openCreateNode = (parentID?: string) => {
    setEditingNode(null)
    nodeForm.resetFields()
    if (parentID) nodeForm.setFieldsValue({ parentID })
    setNodeModalOpen(true)
  }

  const openEditNode = (node: { organizationID: string; parentID?: string; name: string; code?: string; sort?: number; status: string }) => {
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
      void loadChildren(page)
    } catch {
      /* 校验或请求失败 */
    }
  }

  const removeNode = async (id: string) => {
    try {
      await deleteOrganization(id, true)
      message.success('删除成功')
      if (selectedID === id) setSelectedID('')
      if (page > 1 && childrenList.length === 1) setPage(page - 1)
      void loadTree()
      void loadChildren(page)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const toggleStatus = async (node: { organizationID: string; status: string }) => {
    const next = node.status === 'active' ? 'inactive' : 'active'
    await updateOrganizationStatus(node.organizationID, next)
    message.success('状态已更新')
    void loadTree()
    void loadChildren(page)
  }

  // 编辑时可选的父组织：排除当前节点及其子孙，避免环路
  const parentTreeData = useMemo(() => {
    const baseID = editingNode?.organizationID
    if (!baseID) return []
    const exclude = collectSelfAndDescendants(orgList, baseID)
    const toTree = (items: OrganizationItem[]): any[] =>
      items
        .filter((n) => !exclude.has(n.organizationID))
        .map((n) => ({
          title: n.name,
          value: n.organizationID,
          children: n.children?.length ? toTree(n.children) : undefined,
        }))
    return toTree(orgList)
  }, [orgList, editingNode])

  // 右侧下级组织列
  const childrenColumns: ColumnsType<OrganizationChildItem> = [
    {
      title: '组织名称',
      dataIndex: 'name',
      key: 'name',
      render: (v: string, r) => <a onClick={() => setSelectedID(r.organizationID)}>{v}</a>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (v: string) => <StatusTag value={v} />,
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v?: number) => (v ? fmtTime(v * 1000) : '-') },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, r) => (
        <Space size={0}>
          <Button type="link" size="small" onClick={() => openEditNode(r)}>
            编辑
          </Button>
          <Button type="link" size="small" onClick={() => void toggleStatus(r)}>
            {r.status === 'active' ? '停用' : '启用'}
          </Button>
          <Popconfirm title="确认删除该组织及其子组织？" onConfirm={() => void removeNode(r.organizationID)}>
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer title="组织管理" description="组织树维护：组织层级与组织节点管理">
      <style>{`
        #org-tree-card .ant-tree-treenode {
          padding-inline: 0;
        }
        #org-tree-card .ant-tree-switcher {
          width: 0;
          overflow: hidden;
        }
      `}</style>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16 }}>
        <Card id="org-tree-card" style={{ width: 280, flexShrink: 0, borderRadius: 12 }} styles={{ body: { padding: '16px 16px', maxHeight: 680, overflow: 'auto' } }}>
          <Input placeholder="搜索组织" allowClear value={treeKeyword} onChange={(e) => setTreeKeyword(e.target.value)} style={{ marginBottom: 12 }} />
          <Spin spinning={treeLoading}>
            {visibleTreeData.length === 0 ? (
              <Button type="dashed" block icon={<PlusOutlined />} onClick={() => openCreateNode()}>
                创建组织
              </Button>
            ) : (
              <Tree
                treeData={visibleTreeData}
                selectedKeys={selectedID ? [selectedID] : []}
                onSelect={(keys) => keys.length && setSelectedID(String(keys[0]))}
                defaultExpandAll
                blockNode
              />
            )}
          </Spin>
        </Card>

        <Card style={{ flex: 1, borderRadius: 12 }} styles={{ body: { padding: '16px 24px 24px' } }}>
          {/* 查询筛选栏 */}
          <Form
            form={filterForm}
            layout="inline"
            style={{ marginBottom: 16, rowGap: 12 }}
            onFinish={(v: { name?: string; status?: string }) => {
              setQuery({ name: v.name, status: v.status })
            }}
          >
            <Form.Item name="name" label="组织名称">
              <Input placeholder="请输入组织名称" allowClear style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                allowClear
                placeholder="请选择状态"
                style={{ width: 160 }}
                options={[
                  { label: '启用', value: 'active' },
                  { label: '停用', value: 'inactive' },
                ]}
              />
            </Form.Item>
            <Form.Item style={{ marginLeft: 'auto', marginRight: 0 }}>
              <Space>
                <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                  查询
                </Button>
                <Button
                  onClick={() => {
                    filterForm.resetFields()
                    setQuery({})
                  }}
                >
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <div style={{ borderBottom: `1px solid ${tokens.border}`, marginBottom: 16 }} />
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreateNode(selectedID || defaultRootID || undefined)}>
              新增组织
            </Button>
          </div>
          <div style={{ borderBottom: `1px solid ${tokens.border}`, marginBottom: 16 }} />
          <Table<OrganizationChildItem>
            rowKey={(r) => r.organizationID}
            size="middle"
            columns={childrenColumns}
            dataSource={childrenList}
            loading={childrenLoading}
            locale={{ emptyText: '暂无下级组织' }}
            pagination={{
              current: page,
              pageSize,
              total: childrenTotal,
              showTotal: (t) => `共 ${t} 条`,
              onChange: (p) => {
                setPage(p)
                void loadChildren(p)
              },
            }}
          />
        </Card>
      </div>

      {/* 新建 / 编辑组织 */}
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
