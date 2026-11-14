import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

// App 已是自托管入口（<IamConsoleApp variant="tenant"/>：内部含 AppShell + AuthProvider + BrowserRouter），
// main.tsx 只负责挂载。
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
