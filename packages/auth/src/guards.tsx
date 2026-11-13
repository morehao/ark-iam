import { Spin } from 'antd'

// 注：auth 包处于 ui 下层（ui → auth 依赖方向），无法引用 @ark-iam/ui tokens，
// 这里的中性色与 DESIGN.md「冷白工程台」灰阶保持一致，改动需手动同步。
const LOADING_BG = '#f6f7f9'
const LOADING_TEXT = '#94a3b8'

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
