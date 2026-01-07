/**
 * Shared Folders Hooks - React Query hooks for shared folder feature
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { foldersService } from '@/services'
import { CACHE_TIMES } from '@/shared/config/constants'
import { getErrorMessage } from '@/shared/lib/api'
import type { SharedFolder, SharedFolderListData, PhotoListData } from '@/shared/types'

/**
 * Query Keys
 */
export const foldersKeys = {
  all: ['folders'] as const,
  list: () => [...foldersKeys.all, 'list'] as const,
  detail: (id: string) => [...foldersKeys.all, 'detail', id] as const,
  photos: (folderId: string, page: number, limit: number, search?: string, folderPath?: string) =>
    [...foldersKeys.all, 'photos', { folderId, page, limit, search, folderPath }] as const,
  subfolders: (folderId: string) => [...foldersKeys.all, 'subfolders', folderId] as const,
}

/**
 * List user's shared folders
 */
export function useSharedFolders(enabled = true) {
  return useQuery({
    queryKey: foldersKeys.list(),
    queryFn: async () => {
      const response = await foldersService.listFolders()
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as SharedFolderListData
    },
    enabled,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}

/**
 * Get folder by ID
 */
export function useSharedFolder(folderId: string, enabled = true) {
  return useQuery({
    queryKey: foldersKeys.detail(folderId),
    queryFn: async () => {
      const response = await foldersService.getFolder(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as SharedFolder
    },
    enabled: enabled && !!folderId,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}

/**
 * Add a new shared folder
 */
export function useAddFolder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (driveFolderId: string) => {
      const response = await foldersService.addFolder(driveFolderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as SharedFolder
    },
    onSuccess: (folder) => {
      queryClient.invalidateQueries({ queryKey: foldersKeys.list() })
      queryClient.invalidateQueries({ queryKey: ['faces', 'stats'] }) // Invalidate stats for new folder access
      toast.success(`เพิ่มโฟลเดอร์ "${folder.drive_folder_name}" สำเร็จ`)
    },
    onError: (error) => {
      toast.error(`เพิ่มโฟลเดอร์ไม่สำเร็จ: ${getErrorMessage(error)}`)
    },
  })
}

/**
 * Remove user's access to a folder
 */
export function useRemoveFolder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (folderId: string) => {
      const response = await foldersService.removeFolder(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return folderId
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: foldersKeys.list() })
      toast.success('ออกจากโฟลเดอร์สำเร็จ')
    },
    onError: (error) => {
      toast.error(`ออกจากโฟลเดอร์ไม่สำเร็จ: ${getErrorMessage(error)}`)
    },
  })
}

/**
 * Trigger sync for a folder
 */
export function useTriggerFolderSync() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (folderId: string) => {
      const response = await foldersService.triggerSync(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return folderId
    },
    onSuccess: (folderId) => {
      queryClient.invalidateQueries({ queryKey: foldersKeys.detail(folderId) })
      queryClient.invalidateQueries({ queryKey: foldersKeys.list() })
      toast.success('เริ่ม Sync โฟลเดอร์แล้ว')
    },
    onError: (error) => {
      toast.error(`Sync ไม่สำเร็จ: ${getErrorMessage(error)}`)
    },
  })
}

/**
 * Get photos from a folder
 */
export function useFolderPhotos(folderId: string, page = 1, limit = 20, search?: string, enabled = true, folderPath?: string) {
  return useQuery({
    queryKey: foldersKeys.photos(folderId, page, limit, search, folderPath),
    queryFn: async () => {
      const response = await foldersService.getPhotos(folderId, page, limit, search, folderPath)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data as PhotoListData
    },
    enabled: enabled && !!folderId,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}

/**
 * Get sub-folders within a shared folder
 */
export function useSubFolders(folderId: string, enabled = true) {
  return useQuery({
    queryKey: foldersKeys.subfolders(folderId),
    queryFn: async () => {
      const response = await foldersService.getSubFolders(folderId)
      if (!response.success) {
        throw new Error(response.message)
      }
      return response.data
    },
    enabled: enabled && !!folderId,
    staleTime: CACHE_TIMES.DEFAULT,
  })
}
