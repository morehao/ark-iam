import { Card, Avatar } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { useAuth } from 'react-oidc-context'

const Home = () => {
  const auth = useAuth()
  const profile = auth.user?.profile as Record<string, any> | undefined

  return (
    <Card title="用户信息">
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <Avatar size={64} src={profile?.avatar} icon={<UserOutlined />} />
        <div>
          <div style={{ fontSize: 18, fontWeight: 500 }}>{profile?.name ?? '-'}</div>
          <div style={{ color: '#888', marginTop: 4 }}>
            用户ID: {profile?.sub ?? '-'}
          </div>
        </div>
      </div>
    </Card>
  )
}

export default Home
