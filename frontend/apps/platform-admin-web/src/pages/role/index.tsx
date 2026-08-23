import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Input, Table, Tag, Typography } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, IDCell, PageContainer } from '@ark-iam/ui'
import { getRolePageList, getRoleUsers } from '@ark-iam/api'
import type { RoleItem, RoleUserItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

/** 系统管理等级展示：super→超管，member→成员 */
function adminLevelText(level?: string) {
  switch (level) {
    case 'super':
      return <Tag color="red">超管</Tag>
    case 'member':
    default:
      return <Tag>成员</Tag>
  }
}

export default function RoleList() {
  const [data, setData] = useState<RoleItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  // 详情 Drawer（只读成员列表）
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRole, setDetailRole] = useState<RoleItem | null>(null)
  const [members, setMembers] = useState<RoleUserItem[]>([])
  const [membersTotal, setMembersTotal] = useState(0)
  const [membersLoading, setMembersLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getRolePageList({ page, pageSize, name: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取角色列表失败:', error)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const fetchMembers = useCallback(async (roleID: string) => {
    setMembersLoading(true)
    try {
      const resp = await getRoleUsers(roleID)
      setMembers(resp?.users || [])
      setMembersTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取角色成员失败:', error)
    } finally {
      setMembersLoading(false)
    }
  }, [])

  const handleOpenDetail = (record: RoleItem) => {
    setDetailRole(record)
    setDetailOpen(true)
    void fetchMembers(record.roleID)
  }

  const columns: ColumnsType<RoleItem> = [
    { title: '角色ID', dataIndex: 'roleID', key: 'roleID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '角色名称', dataIndex: 'name', key: 'name', width: 160, render: (v: string) => v || '-' },
    { title: '角色编码', dataIndex: 'code', key: 'code', width: 160, render: (v: string) => v || '-' },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (v: string) => (v === 'builtin' ? <Tag color="gold">内置</Tag> : <Tag color="blue">自定义</Tag>),
    },
    {
      title: '系统管理',
      dataIndex: 'adminLevel',
      key: 'adminLevel',
      width: 110,
      render: (v: string) => adminLevelText(v),
    },
    { title: '描述', dataIndex: 'description', key: 'description', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime(r.createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_, r) => (
        <Button type="link" size="small" onClick={() => handleOpenDetail(r)}>
          详情
        </Button>
      ),
    },
  ]

  const readOnlyMemberColumns: ColumnsType<RoleUserItem> = [
    { title: '用户ID', dataIndex: 'userID', key: 'userID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '姓名', dataIndex: 'name', key: 'name', width: 120, render: (v: string) => v || '-' },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120, render: (v: string) => v || '-' },
    { title: '邮箱', dataIndex: 'email', key: 'email', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '加入时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: number) => fmtTime(v),
    },
  ]

  return (
    <PageContainer
      title="角色管理（平台视角）"
      description="平台角色排查：种子角色与跨租户角色只读查看；租户内角色 CRUD 与授权（成员/菜单）请使用租户自服务控制台"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
          刷新
        </Button>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按角色名称搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>

      <Table<RoleItem>
        rowKey="roleID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1140 }}
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

      {/* 角色详情 Drawer（只读成员列表） */}
      <Drawer title={`角色详情 - ${detailRole?.name || ''}`} width={520} open={detailOpen} onClose={() => setDetailOpen(false)}>
        {detailRole && (
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="角色ID"><IDCell value={detailRole.roleID} /></Descriptions.Item>
            <Descriptions.Item label="角色名称">{detailRole.name}</Descriptions.Item>
            <Descriptions.Item label="角色编码">{detailRole.code}</Descriptions.Item>
            <Descriptions.Item label="来源">
              {detailRole.source === 'builtin' ? <Tag color="gold">内置</Tag> : <Tag color="blue">自定义</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="系统管理">{adminLevelText(detailRole.adminLevel)}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{fmtTime(detailRole.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {detailRole.description || '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
        <div style={{ margin: '16px 0 12px' }}>
          <Typography.Text strong>成员列表</Typography.Text>
          <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
            共 {membersTotal} 位成员
          </Typography.Text>
        </div>
        <Table<RoleUserItem>
          rowKey="userID"
          columns={readOnlyMemberColumns}
          dataSource={members}
          loading={membersLoading}
          size="small"
          pagination={false}
          scroll={{ x: 520 }}
        />
      </Drawer>
    </PageContainer>
  )
}
