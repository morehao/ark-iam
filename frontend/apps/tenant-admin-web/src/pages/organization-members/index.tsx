import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  Card,
  Drawer,
  Input,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  TreeSelect,
  message,
} from 'antd'
import { ReloadOutlined, SearchOutlined, UserAddOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import type { OrganizationItem, OrganizationUserItem, TenantUserItem } from '@ark-iam/types'
import {
  createOrganizationUser,
  deleteOrganizationUser,
  getOrganizationTree,
  getOrganizationUserPage,
  updateOrganizationUser,
} from '../../api/organization'
import { getTenantUserPageList } from '../../api/user'

const RELATION_OPTIONS = [
  { label: '成员', value: 'member' },
  { label: '负责人', value: 'leader' },
]

export default function OrganizationMembersPage() {
  const [orgTree, setOrgTree] = useState<OrganizationItem[]>([])
  const [orgLoading, setOrgLoading] = useState(false)
  const [selectedID, setSelectedID] = useState<string>('')

  const [members, setMembers] = useState<OrganizationUserItem[]>([])
  const [memberTotal, setMemberTotal] = useState(0)
  const [memberPage, setMemberPage] = useState(1)
  const [memberPageSize, setMemberPageSize] = useState(10)
  const [memberLoading, setMemberLoading] = useState(false)
  const [relationFilter, setRelationFilter] = useState<string | undefined>()
  const [keyword, setKeyword] = useState('')

  const [addDrawerOpen, setAddDrawerOpen] = useState(false)
  const [userOptions, setUserOptions] = useState<TenantUserItem[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [userKeyword, setUserKeyword] = useState('')
  const [selectedUserIDs, setSelectedUserIDs] = useState<string[]>([])
  const [memberRelation, setMemberRelation] = useState('member')
  const [memberIsPrimary, setMemberIsPrimary] = useState(false)
  const [adding, setAdding] = useState(false)

  const loadTree = useCallback(async () => {
    setOrgLoading(true)
    try {
      const resp = await getOrganizationTree()
      setOrgTree(resp.list || [])
      if (!selectedID && resp.list?.length) {
        setSelectedID(resp.list[0].organizationID)
      }
    } finally {
      setOrgLoading(false)
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

  const selectedNode = useMemo(() => (selectedID ? findNode(orgTree, selectedID) : null), [orgTree, selectedID])

  // 当前组织路径面包屑
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
    walk(orgTree)
    return ids.map((id) => ({ id, name: nameMap.get(id) || id }))
  }, [selectedNode, orgTree])

  const treeSelectData = useMemo(() => toTreeSelect(orgTree), [orgTree])

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

  return (
    <PageContainer
      title="成员管理"
      description="按组织查看与管理成员关系（成员/负责人，主归属唯一）"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void loadTree()}>
          刷新
        </Button>
      }
    >
      <Card style={{ borderRadius: 12 }} styles={{ body: { padding: '16px 24px 24px' } }}>
        <Space style={{ width: '100%', marginBottom: 16 }} align="center" wrap>
          <span style={{ fontWeight: 500, whiteSpace: 'nowrap' }}>组织</span>
          <TreeSelect
            style={{ width: 280 }}
            treeData={treeSelectData}
            value={selectedID || undefined}
            onChange={(v) => {
              setSelectedID(String(v ?? ''))
              setMemberPage(1)
            }}
            treeDefaultExpandAll
            placeholder="请选择组织"
            loading={orgLoading}
            allowClear
          />
          {ancestors.length > 0 && (
            <span style={{ color: '#94a3b8', fontSize: 13 }}>
              {ancestors.map((a, i) => (
                <span key={a.id}>
                  {i > 0 && <span style={{ margin: '0 4px' }}>/</span>}
                  {a.name}
                </span>
              ))}
            </span>
          )}
          {selectedID && <Tag color="blue">共 {memberTotal} 名</Tag>}
        </Space>

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
      </Card>

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
