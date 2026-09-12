import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
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
import { actionColumn, NAME_COL_WIDTH, nameColumn, PageContainer, STATUS_COL_WIDTH, EnableTag, tableScrollX, timeColumn, tokens } from '@ark-iam/ui'
import type { DepartmentChildItem, DepartmentItem, DeptNodeStatus } from '@ark-iam/types'
import {
  createDepartment,
  deleteDepartment,
  getDepartmentChildren,
  getDepartmentTree,
  updateDepartment,
  updateDepartmentStatus,
} from '../../api/department'

// 树节点：纯部门名称
function buildTree(list: DepartmentItem[]): DataNode[] {
  return list.map((n) => ({
    key: n.departmentID,
    title: n.name,
    children: n.children?.length ? buildTree(n.children) : undefined,
  }))
}

// 按关键字在树上过滤节点（命中节点保留其祖先链）
function filterTree(list: DepartmentItem[], keyword: string): DepartmentItem[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return list
  const hit = (n: DepartmentItem) => n.name.toLowerCase().includes(kw)
  const walk = (items: DepartmentItem[]): DepartmentItem[] => {
    const out: DepartmentItem[] = []
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
function collectSelfAndDescendants(list: DepartmentItem[], id: string): Set<string> {
  const result = new Set<string>()
  const node = findNode(list, id)
  if (!node) return result
  const collect = (n: DepartmentItem) => {
    result.add(n.departmentID)
    if (n.children?.length) n.children.forEach(collect)
  }
  collect(node)
  return result
}

// findNode 在树列表中按 ID 查找节点
function findNode(list: DepartmentItem[], id: string): DepartmentItem | null {
  for (const n of list) {
    if (n.departmentID === id) return n
    if (n.children?.length) {
      const found = findNode(n.children, id)
      if (found) return found
    }
  }
  return null
}

// toTreeSelect 部门树 -> TreeSelect 数据
function toTreeSelect(list: DepartmentItem[]): any[] {
  return list.map((n) => ({
    title: n.name,
    value: n.departmentID,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }))
}

export default function DepartmentPage() {
  const [deptList, setDeptList] = useState<DepartmentItem[]>([])
  // 左侧树当前被选中的部门；为空表示未选中，右侧展示顶级部门的下级部门
  const [selectedID, setSelectedID] = useState<string>('')
  const [treeLoading, setTreeLoading] = useState(false)

  // 左侧树搜索关键字（仅影响树展示）
  const [treeKeyword, setTreeKeyword] = useState('')

  // ---------- 右侧下级部门列表 ----------
  const [childrenList, setChildrenList] = useState<DepartmentChildItem[]>([])
  const [childrenTotal, setChildrenTotal] = useState(0)
  const [childrenLoading, setChildrenLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [filterForm] = Form.useForm()
  const [query, setQuery] = useState<{ name?: string; status?: DeptNodeStatus }>({})

  // 顶级部门（根节点）：未选中任何部门时，右侧默认展示其下级部门
  const defaultRootID = useMemo(() => deptList[0]?.departmentID || '', [deptList])

  const [nodeModalOpen, setNodeModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<{ departmentID: string; parentID?: string; name: string; sort?: number; status: DeptNodeStatus } | null>(null)
  const [nodeForm] = Form.useForm()

  // 左侧树加载
  const loadTree = useCallback(async () => {
    setTreeLoading(true)
    try {
      const resp = await getDepartmentTree()
      setDeptList(resp.list || [])
    } finally {
      setTreeLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadTree()
  }, [loadTree])

  // 右侧下级部门列表加载（入参：选中部门或顶级部门 + 筛选 + 分页）
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
        const resp = await getDepartmentChildren(parentID, {
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

  const visibleTree = useMemo(() => filterTree(deptList, treeKeyword), [deptList, treeKeyword])
  const visibleTreeData = useMemo(() => buildTree(visibleTree), [visibleTree])

  const openCreateNode = (parentID?: string) => {
    setEditingNode(null)
    nodeForm.resetFields()
    if (parentID) nodeForm.setFieldsValue({ parentID })
    setNodeModalOpen(true)
  }

  const openEditNode = (node: { departmentID: string; parentID?: string; name: string; sort?: number; status: DeptNodeStatus }) => {
    setEditingNode(node)
    nodeForm.setFieldsValue({
      parentID: node.parentID || undefined,
      name: node.name,
      sort: node.sort,
      status: node.status,
    })
    setNodeModalOpen(true)
  }

  const submitNode = async () => {
    try {
      const values = await nodeForm.validateFields()
      if (editingNode) {
        await updateDepartment({ departmentID: editingNode.departmentID, ...values })
        message.success('保存成功')
      } else {
        await createDepartment(values)
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
      await deleteDepartment(id, true)
      message.success('删除成功')
      if (selectedID === id) setSelectedID('')
      if (page > 1 && childrenList.length === 1) setPage(page - 1)
      void loadTree()
      void loadChildren(page)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const toggleStatus = async (node: { departmentID: string; status: DeptNodeStatus }) => {
    const next = node.status === 'enable' ? 'disable' : 'enable'
    await updateDepartmentStatus(node.departmentID, next)
    message.success('状态已更新')
    void loadTree()
    void loadChildren(page)
  }

  // 编辑时可选的父部门：排除当前节点及其子孙，避免环路
  const parentTreeData = useMemo(() => {
    const baseID = editingNode?.departmentID
    if (!baseID) return []
    const exclude = collectSelfAndDescendants(deptList, baseID)
    const toTree = (items: DepartmentItem[]): any[] =>
      items
        .filter((n) => !exclude.has(n.departmentID))
        .map((n) => ({
          title: n.name,
          value: n.departmentID,
          children: n.children?.length ? toTree(n.children) : undefined,
        }))
    return toTree(deptList)
  }, [deptList, editingNode])

  // 右侧下级部门列
  const childrenColumns: ColumnsType<DepartmentChildItem> = [
    nameColumn<DepartmentChildItem>({
      title: '部门名称',
      dataIndex: 'name',
      width: NAME_COL_WIDTH,
      onClick: (r) => setSelectedID(r.departmentID),
    }),
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: STATUS_COL_WIDTH,
      render: (v: string) => <EnableTag value={v} />,
    },
    timeColumn<DepartmentChildItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<DepartmentChildItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<DepartmentChildItem>({
      max: 3,
      actions: (r) => [
        { key: 'edit', label: '编辑', onClick: () => openEditNode(r) },
        {
          key: 'toggle',
          label: r.status === 'enable' ? '停用' : '启用',
          onClick: () => void toggleStatus(r),
        },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          confirm: '确认删除该部门及其子部门？',
          onClick: () => void removeNode(r.departmentID),
        },
      ],
    }),
  ]

  return (
    <PageContainer title="部门管理" description="部门树维护：部门层级与部门节点管理">
      <style>{`
        #dept-tree-card .ant-tree-treenode {
          padding-inline: 0;
        }
        #dept-tree-card .ant-tree-switcher {
          width: 0;
          overflow: hidden;
        }
      `}</style>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16 }}>
        <Card id="dept-tree-card" style={{ width: 280, flexShrink: 0, borderRadius: 12 }} styles={{ body: { padding: '16px 16px', maxHeight: 680, overflow: 'auto' } }}>
          <Input placeholder="搜索部门" allowClear value={treeKeyword} onChange={(e) => setTreeKeyword(e.target.value)} style={{ marginBottom: 12 }} />
          <Spin spinning={treeLoading}>
            {visibleTreeData.length === 0 ? (
              <Button type="dashed" block icon={<PlusOutlined />} onClick={() => openCreateNode()}>
                创建部门
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
            onFinish={(v: { name?: string; status?: DeptNodeStatus }) => {
              setQuery({ name: v.name, status: v.status })
            }}
          >
            <Form.Item name="name" label="部门名称">
              <Input placeholder="请输入部门名称" allowClear style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                allowClear
                placeholder="请选择状态"
                style={{ width: 160 }}
                options={[
                  { label: '启用', value: 'enable' },
                  { label: '停用', value: 'disable' },
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
              新增部门
            </Button>
          </div>
          <div style={{ borderBottom: `1px solid ${tokens.border}`, marginBottom: 16 }} />
          <Table<DepartmentChildItem>
            rowKey={(r) => r.departmentID}
            size="middle"
            columns={childrenColumns}
            dataSource={childrenList}
            loading={childrenLoading}
            locale={{ emptyText: '暂无下级部门' }}
            scroll={tableScrollX(childrenColumns)}
            tableLayout="fixed"
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

      {/* 新建 / 编辑部门 */}
      <Modal title={editingNode ? '编辑部门（可改父部门实现移动）' : '新建部门'} open={nodeModalOpen} onOk={() => void submitNode()} onCancel={() => setNodeModalOpen(false)} destroyOnClose>
        <Form form={nodeForm} layout="vertical">
          {editingNode && (
            <Form.Item name="parentID" label="父部门（不选为根节点；移动会级联更新子部门路径）">
              <TreeSelect allowClear treeDefaultExpandAll treeData={parentTreeData} placeholder="选择父部门" />
            </Form.Item>
          )}
          {!editingNode && (
            <Form.Item name="parentID" label="父部门（不选为根节点）">
              <TreeSelect allowClear treeDefaultExpandAll treeData={deptList.length ? toTreeSelect(deptList) : []} placeholder="选择父部门" />
            </Form.Item>
          )}
          <Form.Item name="name" label="部门名称" rules={[{ required: true, message: '请输入部门名称' }]}>
            <Input placeholder="如：产品研发部" />
          </Form.Item>
          <Form.Item name="sort" label="同级排序" initialValue={0}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="enable">
            <Select
              options={[
                { label: '启用', value: 'enable' },
                { label: '停用', value: 'disable' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
