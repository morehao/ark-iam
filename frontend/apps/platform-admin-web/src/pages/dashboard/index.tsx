import { Card, Row, Col } from 'antd'

const Dashboard = () => {
  return (
    <div>
      <h1>仪表盘</h1>
      <Row gutter={16}>
        <Col span={6}>
          <Card title="用户总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="角色总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="部门总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
        <Col span={6}>
          <Card title="应用总数">
            <p style={{ fontSize: 24, textAlign: 'center' }}>-</p>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard