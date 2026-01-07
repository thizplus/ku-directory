/**
 * Store for face processing stats (real-time via WebSocket)
 */
import { create } from 'zustand'

export interface FaceStats {
  total_photos: number
  processed_photos: number
  pending_photos: number
  failed_photos: number
  total_faces: number
}

interface FaceStatsStore {
  stats: FaceStats | null

  // Actions
  setStats: (stats: FaceStats) => void

  // Update from photo:updated WebSocket event
  onPhotoProcessed: (faceCount: number, status: 'completed' | 'failed') => void

  // Reset
  reset: () => void
}

export const useFaceStatsStore = create<FaceStatsStore>((set) => ({
  stats: null,

  setStats: (stats) => set({ stats }),

  onPhotoProcessed: (faceCount, status) =>
    set((state) => {
      if (!state.stats) return state

      if (status === 'completed') {
        return {
          stats: {
            ...state.stats,
            processed_photos: state.stats.processed_photos + 1,
            pending_photos: Math.max(0, state.stats.pending_photos - 1),
            total_faces: state.stats.total_faces + faceCount,
          },
        }
      } else {
        // failed
        return {
          stats: {
            ...state.stats,
            failed_photos: state.stats.failed_photos + 1,
            pending_photos: Math.max(0, state.stats.pending_photos - 1),
          },
        }
      }
    }),

  reset: () => set({ stats: null }),
}))
