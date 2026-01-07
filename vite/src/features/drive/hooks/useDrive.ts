/**
 * Google Drive Hooks - React Query hooks for Google Drive feature
 */
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { driveService } from '@/services'
import { CACHE_TIMES } from '@/shared/config/constants'
import { getErrorMessage } from '@/shared/lib/api'
import type { DriveStatus, DriveFolder, SyncJob, PhotoListData } from '@/shared/types'

/**
 * Query Keys
 */
export const driveKeys = {
  all: ['drive'] as const,
  status: () => [...driveKeys.all, 'status'] as const,
  folders: (parentId?: string) => [...driveKeys.all, 'folders', parentId] as const,
  syncStatus: () => [...driveKeys.all, 'sync-status'] as const,
  photos: (page: number, limit: number, folder?: string, search?: string) =>
    [...driveKeys.all, 'photos', { page, limit, folder, search }] as const,
}

/**
 * Get Drive connection status
 */
export function useDriveStatus(enabled = true) {
  return useQuery({
    queryKey: driveKeys.status(),
    queryFn: async () => {
      const response = await driveService.getStatus()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as DriveStatus
    },
    enabled,
    staleTime: CACHE_TIMES.STATS,
  })
}

/**
 * Disconnect Google Drive
 */
export function useDisconnectDrive() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      const response = await driveService.disconnect()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    onSuccess: () => {
      // Invalidate all drive-related queries
      queryClient.invalidateQueries({ queryKey: driveKeys.all })
    },
    onError: (error) => {
      console.error('Disconnect Drive error:', getErrorMessage(error))
    },
  })
}

/**
 * List folders in Google Drive
 */
export function useDriveFolders(parentId?: string, enabled = true) {
  return useQuery({
    queryKey: driveKeys.folders(parentId),
    queryFn: async () => {
      const response = await driveService.listFolders(parentId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as DriveFolder[]
    },
    enabled,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}

/**
 * Set root folder for sync
 */
export function useSetRootFolder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (folderId: string) => {
      const response = await driveService.setRootFolder(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: driveKeys.status() })
    },
    onError: (error) => {
      console.error('Set root folder error:', getErrorMessage(error))
    },
  })
}

/**
 * Start sync job
 */
export function useStartSync() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      const response = await driveService.startSync()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as SyncJob
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: driveKeys.syncStatus() })
    },
    onError: (error) => {
      console.error('Start sync error:', getErrorMessage(error))
    },
  })
}

/**
 * Get sync status
 * No polling - WebSocket handles real-time updates
 */
export function useSyncStatus(enabled = true) {
  return useQuery({
    queryKey: driveKeys.syncStatus(),
    queryFn: async () => {
      const response = await driveService.getSyncStatus()
      if (!response.success) {
        throw new Error(response.message)
      }
      // Return null instead of undefined if no sync job
      return (response.data as SyncJob | null) ?? null
    },
    enabled,
    staleTime: 5000, // 5 seconds
  })
}

/**
 * Get photos
 */
export function useDrivePhotos(page = 1, limit = 20, folder?: string, enabled = true, search?: string) {
  return useQuery({
    queryKey: driveKeys.photos(page, limit, folder, search),
    queryFn: async () => {
      const response = await driveService.getPhotos(page, limit, folder, search)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as PhotoListData
    },
    enabled,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}

/**
 * Connect to Google Drive (get OAuth URL and redirect)
 */
export function useConnectDrive() {
  return useMutation({
    mutationFn: async () => {
      const response = await driveService.getConnectUrl()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as { authUrl: string }
    },
    onSuccess: (data) => {
      // Redirect to Google OAuth
      window.location.href = data.authUrl
    },
    onError: (error) => {
      console.error('Connect Drive error:', getErrorMessage(error))
    },
  })
}

/**
 * Download multiple photos as a zip file
 * Returns mutation with progress tracking
 */
export function useDownloadPhotos() {
  const [progress, setProgress] = useState(0)

  const mutation = useMutation({
    mutationFn: async (driveFileIds: string[]) => {
      setProgress(0)
      const blob = await driveService.downloadPhotos(driveFileIds, (percent) => {
        setProgress(percent)
      })
      return { blob, count: driveFileIds.length }
    },
    onSuccess: ({ blob, count }) => {
      // Create download link
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `photos_${new Date().toISOString().slice(0, 10)}.zip`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      setProgress(0)
      toast.success(`ดาวน์โหลดเสร็จแล้ว ${count} ไฟล์`)
    },
    onError: (error) => {
      console.error('Download photos error:', getErrorMessage(error))
      setProgress(0)
    },
  })

  return { ...mutation, progress }
}
