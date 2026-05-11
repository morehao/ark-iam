import { useEffect, useState } from 'react'
import { Table, Input } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getRolePageList, Role } from '../../api/role'

interface RoleTableItem extends Role {}

const RoleList = () => {
  const [data, setData] = useState<RoleTableItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const resp = await getRolePageList({ page, pageSize, keyword })
      setData(resp.data?.list || [])
      setTotal(resp.data?.total || 0)
    } catch (error) {
      console.error('获取角色列表失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, keyword])

  const columns: ColumnsType<RoleTableItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: '角色名称', dataIndex: 'name', key: 'name' },
    { title: '角色编码', dataIndex: 'code', key: 'code' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '状态', dataIndex: 'status', key: 'status' },
  ]

  return (
    <div>
      <h1>角色管理</h1>
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索角色"
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

export default RoleList