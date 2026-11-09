import '@testing-library/jest-dom/vitest'

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

// localStorage 兜底：部分 jsdom/node 组合下 window.localStorage 未挂载（读它直接 undefined），
// 页面用 localStorage 记忆「上次选择的所属应用」，缺失会让渲染期抛错、整套用例连锁失败。
// 这里提供内存实现，行为与浏览器 Storage 一致（getItem/setItem/removeItem/clear/key/length）。
if (typeof globalThis.localStorage === 'undefined') {
  const store: Record<string, string> = {}
  const memoryStorage: Storage = {
    getItem: (key: string) => (key in store ? store[key] : null),
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      for (const key of Object.keys(store)) delete store[key]
    },
    key: (index: number) => Object.keys(store)[index] ?? null,
    get length() {
      return Object.keys(store).length
    },
  }
  Object.defineProperty(globalThis, 'localStorage', { value: memoryStorage, writable: true, configurable: true })
  Object.defineProperty(window, 'localStorage', { value: memoryStorage, writable: true, configurable: true })
}

// antd6 的 rc-resize-observer / rc-virtual-list 依赖全局 ResizeObserver（jsdom 未实现），
// 缺失会在渲染表格/下拉时抛 `ResizeObserver is not defined`。这里提供空实现兜底。
if (typeof globalThis.ResizeObserver === 'undefined') {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  ;(globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = ResizeObserverStub
}

// antd6 的静态 `Modal.confirm` / `message` 渲染到 body 上的独立 holder，RTL 的 cleanup
// 不会卸载这些 portal，导致跨用例 DOM 泄漏（下一条用例会误以为"弹窗还在"）。
// 每个用例后清空 antd 静态 holder + 残留 portal，保证用例隔离。
import { afterEach } from 'vitest'
afterEach(() => {
  document.querySelectorAll('.ant-modal-root, .ant-modal-wrap, .ant-message, .ant-notification').forEach((el) => {
    el.parentNode?.removeChild(el)
  })
})
