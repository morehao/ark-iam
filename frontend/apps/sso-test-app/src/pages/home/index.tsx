import { Card, Avatar } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'

const Home = () => {
  const personInfo = useAuthStore((state) => state.personInfo)

  return (
    <Card title="用户信息">
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <Avatar
          size={64}
          src={personInfo?.avatar}
          icon={!personInfo?.avatar ? <UserOutlined /> : undefined}
        />
        <div>
          <div style={{ fontSize: 18, fontWeight: 500 }}>{personInfo?.name ?? '-'}</div>
          <div style={{ color: '#888', marginTop: 4 }}>
            用户ID: {personInfo?.personID ?? '-'}
          </div>
        </div>
      </div>
    </Card>
  )
}

export default Home
