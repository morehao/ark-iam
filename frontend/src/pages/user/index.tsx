import { useEffect, useState } from 'react'
import { Table, Button, Space, Input } from 'antd'
import { useNavigate } from 'react-router'
import type { ColumnsType } from 'antd/es/table'
import { getUserPageList, User } from '../../api/user'

interface UserTableItem extends User {}

const UserList = () => {
  const navigate = useNavigate()
  const [data, setData] = useState<UserTableItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getUserPageList({ page, pageSize, keyword })
      setData(resp.data?.list || [])
      setTotal(resp.data?.total || 0)
    } catch (error) {
      console.error('获取用户列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const columns: ColumnsType<UserTableItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '用户名', dataIndex: 'username', key: 'username' },
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    { title: '手机号', dataIndex: 'phone', key: 'phone' },
    { title: '状态', dataIndex: 'status', key: 'status' },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => navigate(`/user/${record.id}`)}>
            详情
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <h1>用户管理</h1>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索用户"
          style={{ width: 200 }}
          onSearch={(value) => setKeyword(value)}
        />
      </div>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />
    </div>
  )
}

export default UserList