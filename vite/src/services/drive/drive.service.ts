/**
 * Google Drive Service Layer
 */
import { apiClient, type ApiResponse } from '@/shared/lib/api'
import { DRIVE_API } from '@/shared/lib/api/constants/api'
import type { DriveStatus, DriveFolder, SyncJob, PhotoListData } from '@/shared/types'

/**
 * Get the OAuth connect URL from backend
 * Backend will set cookies and return the Google OAuth URL
 */
async function getConnectUrl(): Promise<ApiResponse<{ authUrl: string }>> {
  const response = await apiClient.get<ApiResponse<{ authUrl: string }>>(DRIVE_API.CONNECT)
  return response.data
}

/**
 * Get connection status
 */
async function getStatus(): Promise<ApiResponse<DriveStatus>> {
  const response = await apiClient.get<ApiResponse<DriveStatus>>(DRIVE_API.STATUS)
  return response.data
}

/**
 * Disconnect Google Drive
 */
async function disconnect(): Promise<ApiResponse<null>> {
  const response = await apiClient.post<ApiResponse<null>>(DRIVE_API.DISCONNECT)
  return response.data
}

/**
 * List folders in Google Drive
 */
async function listFolders(parentId?: string): Promise<ApiResponse<DriveFolder[]>> {
  const params = parentId ? { parent: parentId } : {}
  const response = await apiClient.get<ApiResponse<DriveFolder[]>>(DRIVE_API.FOLDERS, { params })
  return response.data
}

/**
 * Set root folder for sync
 */
async function setRootFolder(folderId: string): Promise<ApiResponse<null>> {
  const response = await apiClient.post<ApiResponse<null>>(DRIVE_API.SET_ROOT_FOLDER, {
    folderId,
  })
  return response.data
}

/**
 * Start sync job
 */
async function startSync(): Promise<ApiResponse<SyncJob>> {
  const response = await apiClient.post<ApiResponse<SyncJob>>(DRIVE_API.SYNC)
  return response.data
}

/**
 * Get sync status
 */
async function getSyncStatus(): Promise<ApiResponse<SyncJob | null>> {
  const response = await apiClient.get<ApiResponse<SyncJob | null>>(DRIVE_API.SYNC_STATUS)
  return response.data
}

/**
 * Get photos
 */
async function getPhotos(
  page = 1,
  limit = 20,
  folder?: string,
  search?: string
): Promise<ApiResponse<PhotoListData>> {
  const params: Record<string, string | number> = { page, limit }
  if (folder) {
    params.folder = folder
  }
  if (search) {
    params.search = search
  }
  const response = await apiClient.get<ApiResponse<PhotoListData>>(DRIVE_API.PHOTOS, { params })
  return response.data
}

/**
 * Download multiple photos as a zip file
 * Returns a Blob for download
 * @param onProgress - Optional callback for download progress (0-100)
 */
async function downloadPhotos(
  driveFileIds: string[],
  onProgress?: (progress: number) => void
): Promise<Blob> {
  const response = await apiClient.post(
    DRIVE_API.DOWNLOAD,
    { driveFileIds },
    {
      responseType: 'blob',
      timeout: 5 * 60 * 1000, // 5 minutes for large downloads
      onDownloadProgress: (progressEvent) => {
        if (progressEvent.total && onProgress) {
          const percent = Math.round((progressEvent.loaded / progressEvent.total) * 100)
          onProgress(percent)
        }
      },
    }
  )
  return response.data
}

export const driveService = {
  getConnectUrl,
  getStatus,
  disconnect,
  listFolders,
  setRootFolder,
  startSync,
  getSyncStatus,
  getPhotos,
  downloadPhotos,
}
