import { Spin } from 'antd'

// 注：auth 包处于 ui 下层（ui → auth 依赖方向），无法引用 @ark-iam/ui tokens，
// 这里的中性色与 DESIGN.md「Ant Design 原生」灰阶保持一致（画布 #f5f5f5 / 占位色 rgba(0,0,0,0.45)），改动需手动同步。
const LOADING_BG = '#f5f5f5'
const LOADING_TEXT = 'rgba(0,0,0,0.45)'

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
        background: LOADING_BG,
      }}
    >
      <Spin size="large" />
      <span style={{ color: LOADING_TEXT, fontSize: 13 }}>正在加载 IAM 平台…</span>
    </div>
  )
}
