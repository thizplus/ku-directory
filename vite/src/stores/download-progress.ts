/**
 * Download Progress Store - Tracks download progress from WebSocket
 */
import { create } from 'zustand'

interface DownloadProgress {
  current: number
  total: number
  fileName: string
}

interface DownloadProgressState {
  progress: DownloadProgress | null
  isDownloading: boolean
  setProgress: (progress: DownloadProgress) => void
  setCompleted: () => void
  reset: () => void
}

export const useDownloadProgressStore = create<DownloadProgressState>((set) => ({
  progress: null,
  isDownloading: false,
  setProgress: (progress) => set({ progress, isDownloading: true }),
  setCompleted: () => set({ progress: null, isDownloading: false }),
  reset: () => set({ progress: null, isDownloading: false }),
}))
