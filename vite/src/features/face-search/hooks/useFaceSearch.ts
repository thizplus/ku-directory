/**
 * Face Search Hooks - React Query hooks for face search feature
 */
import { useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { faceService } from '@/services'
import { CACHE_TIMES } from '@/shared/config/constants'
import { getErrorMessage } from '@/shared/lib/api'
import { useFaceStatsStore } from '@/stores/face-stats'
import type { FaceSearchData, FaceProcessingStats, DetectedFacesData } from '@/shared/types'

/**
 * Query Keys
 */
export const faceKeys = {
  all: ['faces'] as const,
  stats: () => [...faceKeys.all, 'stats'] as const,
  list: (page?: number, limit?: number) => [...faceKeys.all, 'list', { page, limit }] as const,
  byPhoto: (photoId: string) => [...faceKeys.all, 'photo', photoId] as const,
  search: () => [...faceKeys.all, 'search'] as const,
}

/**
 * Get face processing statistics
 * Uses Zustand store for real-time WebSocket updates
 */
export function useFaceStats() {
  const storeStats = useFaceStatsStore((state) => state.stats)
  const setStats = useFaceStatsStore((state) => state.setStats)

  // Fetch initial stats from API
  const query = useQuery({
    queryKey: faceKeys.stats(),
    queryFn: async () => {
      const response = await faceService.getStats()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as FaceProcessingStats
    },
    staleTime: CACHE_TIMES.STATS,
    // No polling - WebSocket updates the store directly
  })

  // Sync API data to Zustand store
  useEffect(() => {
    if (query.data) {
      setStats(query.data)
    }
  }, [query.data, setStats])

  // Return store stats (updated by WebSocket) or query data as fallback
  return {
    ...query,
    data: storeStats || query.data,
  }
}

/**
 * Detect faces in an uploaded image
 * Returns all faces with bounding boxes for selection
 */
export function useDetectFaces() {
  return useMutation({
    mutationFn: async (imageFile: File) => {
      const response = await faceService.detectFaces(imageFile)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as DetectedFacesData
    },
    onError: (error) => {
      console.error('Face detection error:', getErrorMessage(error))
    },
  })
}

/**
 * Search faces by uploading an image
 * @param faceIndex - Index of the face to search (for multi-face images)
 */
export function useFaceSearchByImage() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      imageFile,
      limit = 20,
      threshold = 0.6,
      faceIndex = 0,
    }: {
      imageFile: File
      limit?: number
      threshold?: number
      faceIndex?: number
    }) => {
      const response = await faceService.searchByImage(imageFile, limit, threshold, faceIndex)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as FaceSearchData
    },
    onSuccess: () => {
      // Optionally invalidate stats after search
      queryClient.invalidateQueries({ queryKey: faceKeys.stats() })
    },
    onError: (error) => {
      console.error('Face search error:', getErrorMessage(error))
    },
  })
}

/**
 * Search faces by existing face ID
 */
export function useFaceSearchByFaceId() {
  return useMutation({
    mutationFn: async ({
      faceId,
      limit = 20,
      threshold = 0.6,
    }: {
      faceId: string
      limit?: number
      threshold?: number
    }) => {
      const response = await faceService.searchByFaceId(faceId, limit, threshold)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as FaceSearchData
    },
    onError: (error) => {
      console.error('Face search by ID error:', getErrorMessage(error))
    },
  })
}

/**
 * Get paginated list of faces
 */
export function useFaces(page = 1, limit = 50, enabled = true) {
  return useQuery({
    queryKey: faceKeys.list(page, limit),
    queryFn: async () => {
      const response = await faceService.getFaces(page, limit)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    enabled,
    staleTime: CACHE_TIMES.FACES,
  })
}

/**
 * Get faces for a specific photo
 */
export function useFacesByPhoto(photoId: string, enabled = true) {
  return useQuery({
    queryKey: faceKeys.byPhoto(photoId),
    queryFn: async () => {
      const response = await faceService.getFacesByPhoto(photoId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    enabled: enabled && !!photoId,
    staleTime: CACHE_TIMES.FACES,
  })
}

/**
 * Retry failed face processing
 * Resets failed photos to pending for reprocessing
 */
export function useRetryFailedPhotos() {
  const onRetryFailed = useFaceStatsStore((state) => state.onRetryFailed)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (folderId?: string) => {
      const response = await faceService.retryFailed(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    onSuccess: (data) => {
      // Update store with the number of reset photos
      if (data && data.reset_count > 0) {
        onRetryFailed(data.reset_count)
      }
      // Invalidate stats to refresh from server
      queryClient.invalidateQueries({ queryKey: faceKeys.stats() })
    },
    onError: (error) => {
      console.error('Retry failed photos error:', getErrorMessage(error))
    },
  })
}
