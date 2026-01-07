/**
 * Store for tracking folder sync progress (real-time via WebSocket)
 */
import { create } from 'zustand'

interface SyncProgress {
  folderId: string
  percent: number
  processedFiles: number
  totalFiles: number
}

interface SyncProgressStore {
  // Map of folderId -> progress
  progress: Record<string, SyncProgress>

  // Actions
  setProgress: (folderId: string, data: Omit<SyncProgress, 'folderId'>) => void
  clearProgress: (folderId: string) => void
  clearAll: () => void
}

export const useSyncProgressStore = create<SyncProgressStore>((set) => ({
  progress: {},

  setProgress: (folderId, data) =>
    set((state) => ({
      progress: {
        ...state.progress,
        [folderId]: { folderId, ...data },
      },
    })),

  clearProgress: (folderId) =>
    set((state) => {
      const { [folderId]: _, ...rest } = state.progress
      return { progress: rest }
    }),

  clearAll: () => set({ progress: {} }),
}))
