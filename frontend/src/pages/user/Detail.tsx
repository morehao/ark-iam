import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Descriptions, Button, Spin } from 'antd'
import { getUserDetail } from '../../api/user'

const UserDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return
      setLoading(true)
      try {
        const resp = await getUserDetail(Number(id))
        setUser(resp.data)
      } catch (error) {
        console.error('获取用户详情失败:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [id])

  if (loading) return <Spin />

  if (!user) return null

  return (
    <div>
      <Button onClick={() => navigate('/user')}>返回</Button>
      <Card title="用户详情" style={{ marginTop: 16 }}>
        <Descriptions column={2} bordered>
          <Descriptions.Item label="ID">{user.id}</Descriptions.Item>
          <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
          <Descriptions.Item label="姓名">{user.name}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
          <Descriptions.Item label="手机号">{user.phone}</Descriptions.Item>
          <Descriptions.Item label="状态">{user.status}</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}

export default UserDetail