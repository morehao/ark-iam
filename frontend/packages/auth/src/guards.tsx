import { Spin } from 'antd'

export function FullPageSpinner() {
  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 16,
        background: 'linear-gradient(135deg, #eef2ff 0%, #f5f0ff 100%)',
      }}
    >
      <Spin size="large" />
      <span style={{ color: 'rgba(17, 24, 39, 0.45)', fontSize: 13 }}>正在加载 IAM 平台…</span>
    </div>
  )
}
