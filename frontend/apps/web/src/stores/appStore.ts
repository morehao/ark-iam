import { create } from 'zustand'

interface AppState {
  collapsed: boolean
  toggleCollapsed: () => void
}

export const useAppStore = create<AppState>((set) => ({
  collapsed: false,
  toggleCollapsed: () => set((state) => ({ collapsed: !state.collapsed })),
}))