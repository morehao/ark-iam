import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  Card,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  Tree,
  TreeSelect,
  message,
} from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined, UserAddOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { DataNode } from 'antd/es/tree'
import { PageContainer } from '@ark-iam/ui'
import type { OrganizationItem, OrganizationUserItem, TenantUserItem } from '@ark-iam/types'
import {
  createOrganization,
  createOrganizationUser,
  deleteOrganization,
  deleteOrganizationUser,
  getOrganizationTree,
  getOrganizationUserPage,
  updateOrganization,
  updateOrganizationStatus,
  updateOrganizationUser,
} from '../../api/organization'
import { getTenantUserPageList } from '../../api/user'

const RELATION_OPTIONS = [
  { label: '成员', value: 'member' },
  { label: '负责人', value: 'leader' },
]

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

  const [members, setMembers] = useState<OrganizationUserItem[]>([])
  const [memberTotal, setMemberTotal] = useState(0)
  const [memberPage, setMemberPage] = useState(1)
  const [memberPageSize, setMemberPageSize] = useState(10)
  const [memberLoading, setMemberLoading] = useState(false)
  const [relationFilter, setRelationFilter] = useState<string | undefined>()
  const [keyword, setKeyword] = useState('')

  const [nodeModalOpen, setNodeModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<OrganizationItem | null>(null)
  const [nodeForm] = Form.useForm()

  const [addDrawerOpen, setAddDrawerOpen] = useState(false)
  const [userOptions, setUserOptions] = useState<TenantUserItem[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [userKeyword, setUserKeyword] = useState('')
  const [selectedUserIDs, setSelectedUserIDs] = useState<string[]>([])
  const [memberRelation, setMemberRelation] = useState('member')
  const [memberIsPrimary, setMemberIsPrimary] = useState(false)
  const [adding, setAdding] = useState(false)

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

  const loadMembers = useCallback(async () => {
    if (!selectedID) return
    setMemberLoading(true)
    try {
      const resp = await getOrganizationUserPage(selectedID, {
        page: memberPage,
        pageSize: memberPageSize,
        relationType: relationFilter,
        keyword: keyword || undefined,
      })
      setMembers(resp.list || [])
      setMemberTotal(resp.total || 0)
    } finally {
      setMemberLoading(false)
    }
  }, [selectedID, memberPage, memberPageSize, relationFilter, keyword])

  useEffect(() => {
    void loadMembers()
  }, [loadMembers])

  const selectedNode = useMemo(() => (selectedID ? findNode(orgList, selectedID) : null), [orgList, selectedID])

  // 节点信息：面包屑 + 子节点数 + 成员数
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

  const loadUserOptions = useCallback(async (kw: string) => {
    setUsersLoading(true)
    try {
      const resp = await getTenantUserPageList({ page: 1, pageSize: 50, keyword: kw || undefined })
      setUserOptions(resp.list || [])
    } finally {
      setUsersLoading(false)
    }
  }, [])

  const openAddDrawer = () => {
    setSelectedUserIDs([])
    setMemberRelation('member')
    setMemberIsPrimary(false)
    setUserKeyword('')
    setAddDrawerOpen(true)
    void loadUserOptions('')
  }

  const submitAddMembers = async () => {
    if (!selectedID || selectedUserIDs.length === 0) {
      message.warning('请选择要添加的用户')
      return
    }
    setAdding(true)
    try {
      for (let i = 0; i < selectedUserIDs.length; i++) {
        await createOrganizationUser(selectedID, {
          userID: selectedUserIDs[i],
          relationType: memberRelation,
          isPrimary: memberIsPrimary && i === 0,
        })
      }
      message.success('添加成功')
      setAddDrawerOpen(false)
      void loadMembers()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setAdding(false)
    }
  }

  const removeMember = async (userID: string) => {
    if (!selectedID) return
    await deleteOrganizationUser(selectedID, userID)
    message.success('移除成功')
    void loadMembers()
  }

  const setPrimary = async (userID: string) => {
    if (!selectedID) return
    await updateOrganizationUser(selectedID, userID, { relationType: 'member', isPrimary: true })
    message.success('已设为主归属')
    void loadMembers()
  }

  const changeRelation = async (userID: string, relationType: string) => {
    if (!selectedID) return
    await updateOrganizationUser(selectedID, userID, { relationType })
    message.success('关系已更新')
    void loadMembers()
  }

  const memberColumns: ColumnsType<OrganizationUserItem> = [
    {
      title: '用户',
      key: 'user',
      width: 200,
      render: (_, r) => (
        <Space>
          <Avatar size={28}>{r.userName?.charAt(0)?.toUpperCase() || 'U'}</Avatar>
          <Space direction="vertical" size={0}>
            <span style={{ fontWeight: 500 }}>{r.userName || '-'}</span>
            <span style={{ fontSize: 12, color: '#94a3b8' }}>@{r.username || '-'}</span>
          </Space>
        </Space>
      ),
    },
    { title: '邮箱', dataIndex: 'primaryEmail', key: 'primaryEmail', render: (v: string) => v || '-' },
    { title: '手机号', dataIndex: 'primaryPhone', key: 'primaryPhone', width: 130, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="red">挂起</Tag> : <Tag color="green">正常</Tag>),
    },
    {
      title: '关系',
      dataIndex: 'relationType',
      key: 'relationType',
      width: 110,
      render: (v: string, r) => (
        <Select
          size="small"
          value={v}
          style={{ width: 92 }}
          options={RELATION_OPTIONS}
          onChange={(next) => void changeRelation(r.userID, next)}
        />
      ),
    },
    {
      title: '主归属',
      dataIndex: 'isPrimary',
      key: 'isPrimary',
      width: 90,
      render: (v: boolean, r) =>
        v ? <Tag color="gold">主</Tag> : r.relationType === 'member' ? <Button type="link" size="small" onClick={() => void setPrimary(r.userID)}>设为主</Button> : <Tag>否</Tag>,
    },
    {
      title: '加入时间',
      dataIndex: 'joinedAt',
      key: 'joinedAt',
      width: 160,
      render: (v?: number) => fmtTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_, r) => (
        <Popconfirm title="确认移除该关系？" onConfirm={() => void removeMember(r.userID)}>
          <Button type="link" size="small" danger>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ]

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

        <Card style={{ flex: 1, borderRadius: 12 }} styles={{ body: { padding: 0 } }}>
          {!selectedID || !selectedNode ? (
            <div style={{ padding: 60, textAlign: 'center', color: '#94a3b8' }}>请先选择左侧组织节点</div>
          ) : (
            <Tabs
              defaultActiveKey="info"
              items={[
                {
                  key: 'info',
                  label: '节点信息',
                  children: (
                    <div style={{ padding: 16 }}>
                      <Descriptions column={2} bordered size="small">
                        <Descriptions.Item label="组织名称">{selectedNode.name}</Descriptions.Item>
                        <Descriptions.Item label="组织编码">{selectedNode.code || '-'}</Descriptions.Item>
                        <Descriptions.Item label="同级排序">{selectedNode.sort ?? 0}</Descriptions.Item>
                        <Descriptions.Item label="状态">
                          {selectedNode.status === 'active' ? <Tag color="green">启用</Tag> : <Tag color="orange">停用</Tag>}
                        </Descriptions.Item>
                        <Descriptions.Item label="子组织数">{childCount}</Descriptions.Item>
                        <Descriptions.Item label="成员数">{memberTotal}</Descriptions.Item>
                        <Descriptions.Item label="路径" span={2}>
                          <Space size={4} wrap>
                            {ancestors.map((a, i) => (
                              <span key={a.id}>
                                {i > 0 && <span style={{ margin: '0 4px', color: '#94a3b8' }}>/</span>}
                                {a.name}
                              </span>
                            ))}
                          </Space>
                        </Descriptions.Item>
                      </Descriptions>
                      <Space style={{ marginTop: 16 }}>
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
                    </div>
                  ),
                },
                {
                  key: 'members',
                  label: `成员管理${memberTotal ? `（${memberTotal}）` : ''}`,
                  children: (
                    <div style={{ padding: 16 }}>
                      <Space style={{ marginBottom: 12 }} wrap>
                        <Select
                          allowClear
                          placeholder="关系类型"
                          style={{ width: 130 }}
                          options={RELATION_OPTIONS}
                          value={relationFilter}
                          onChange={(v) => {
                            setRelationFilter(v)
                            setMemberPage(1)
                          }}
                        />
                        <Input.Search
                          allowClear
                          placeholder="姓名/用户名/邮箱/手机"
                          prefix={<SearchOutlined />}
                          style={{ width: 240 }}
                          onSearch={(v) => {
                            setKeyword(v)
                            setMemberPage(1)
                          }}
                        />
                        <Button type="primary" icon={<UserAddOutlined />} onClick={openAddDrawer}>
                          添加成员/负责人
                        </Button>
                      </Space>
                      <Table<OrganizationUserItem>
                        rowKey={(r) => `${r.organizationID}-${r.userID}-${r.relationType}`}
                        loading={memberLoading}
                        columns={memberColumns}
                        dataSource={members}
                        scroll={{ x: 1100 }}
                        pagination={{
                          current: memberPage,
                          pageSize: memberPageSize,
                          total: memberTotal,
                          showSizeChanger: true,
                          showTotal: (t) => `共 ${t} 条`,
                          onChange: (p, ps) => {
                            setMemberPage(p)
                            setMemberPageSize(ps)
                          },
                        }}
                      />
                    </div>
                  ),
                },
              ]}
            />
          )}
        </Card>
      </Space>

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

      <Drawer
        title="添加成员 / 负责人"
        width={480}
        open={addDrawerOpen}
        onClose={() => setAddDrawerOpen(false)}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => setAddDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={adding} onClick={() => void submitAddMembers()}>
              添加
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" style={{ width: '100%' }} size={16}>
          <div>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>选择用户（从租户用户目录搜索）</div>
            <Select
              mode="multiple"
              allowClear
              showSearch
              filterOption={false}
              onSearch={(v) => {
                setUserKeyword(v)
                void loadUserOptions(v)
              }}
              placeholder="输入姓名/用户名/邮箱/手机搜索"
              loading={usersLoading}
              style={{ width: '100%' }}
              value={selectedUserIDs}
              onChange={setSelectedUserIDs}
              options={userOptions.map((u) => ({ label: `${u.name}（@${u.username || '-'}）`, value: u.userID }))}
              notFoundContent={userKeyword ? '无匹配用户' : '输入关键词搜索'}
            />
          </div>
          <div>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>关系类型</div>
            <Select style={{ width: '100%' }} value={memberRelation} onChange={setMemberRelation} options={RELATION_OPTIONS} />
          </div>
          <div>
            <Switch checked={memberIsPrimary} onChange={setMemberIsPrimary} disabled={memberRelation !== 'member'} />
            <span style={{ marginLeft: 8, color: '#64748b' }}>设为首个为主归属（仅成员关系，且仅对第一个用户生效）</span>
          </div>
        </Space>
      </Drawer>
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

// fmtTime 时间戳渲染（秒级兼容）
function fmtTime(value?: number | null): string {
  if (value == null || value === 0) return '-'
  const ms = value < 1e12 ? value * 1000 : value
  const d = new Date(ms)
  if (Number.isNaN(d.getTime())) return String(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
