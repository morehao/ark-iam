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
