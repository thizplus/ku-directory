/**
 * Shared Folders Service Layer
 */
import { apiClient, type ApiResponse } from '@/shared/lib/api'
import { FOLDERS_API } from '@/shared/lib/api/constants/api'
import type { SharedFolder, SharedFolderListData, PhotoListData } from '@/shared/types'

/**
 * List user's shared folders
 */
async function listFolders(): Promise<ApiResponse<SharedFolderListData>> {
  const response = await apiClient.get<ApiResponse<SharedFolderListData>>(FOLDERS_API.LIST)
  return response.data
}

/**
 * Get folder by ID
 */
async function getFolder(folderId: string): Promise<ApiResponse<SharedFolder>> {
  const response = await apiClient.get<ApiResponse<SharedFolder>>(FOLDERS_API.GET(folderId))
  return response.data
}

/**
 * Add a new shared folder (join existing or create new)
 * Returns immediately - photos sync in background via WebSocket
 */
async function addFolder(driveFolderId: string): Promise<ApiResponse<SharedFolder>> {
  const response = await apiClient.post<ApiResponse<SharedFolder>>(
    FOLDERS_API.ADD,
    { drive_folder_id: driveFolderId }
  )
  return response.data
}

/**
 * Remove user's access to a folder (leave folder)
 */
async function removeFolder(folderId: string): Promise<ApiResponse<null>> {
  const response = await apiClient.delete<ApiResponse<null>>(FOLDERS_API.REMOVE(folderId))
  return response.data
}

/**
 * Trigger sync for a folder
 * @param force - If true, forces a full re-sync of all photos
 */
async function triggerSync(folderId: string, force = false): Promise<ApiResponse<null>> {
  const url = force ? `${FOLDERS_API.SYNC(folderId)}?force=true` : FOLDERS_API.SYNC(folderId)
  const response = await apiClient.post<ApiResponse<null>>(url)
  return response.data
}

/**
 * Get photos from a folder
 */
async function getPhotos(
  folderId: string,
  page = 1,
  limit = 20,
  search?: string,
  folderPath?: string
): Promise<ApiResponse<PhotoListData>> {
  const params: Record<string, string | number> = { page, limit }
  if (search) {
    params.search = search
  }
  if (folderPath) {
    params.folder_path = folderPath
  }
  const response = await apiClient.get<ApiResponse<PhotoListData>>(FOLDERS_API.PHOTOS(folderId), { params })
  return response.data
}

/**
 * Sub-folder info from API
 */
export interface SubFolder {
  path: string
  name: string
  photo_count: number
}

export interface SubFoldersData {
  subfolders: SubFolder[]
  total: number
}

/**
 * Get sub-folders within a shared folder
 */
async function getSubFolders(folderId: string): Promise<ApiResponse<SubFoldersData>> {
  const response = await apiClient.get<ApiResponse<SubFoldersData>>(FOLDERS_API.SUBFOLDERS(folderId))
  return response.data
}

/**
 * Reconnect Google Drive to a folder (refresh tokens)
 */
async function reconnectFolder(folderId: string): Promise<ApiResponse<null>> {
  const response = await apiClient.post<ApiResponse<null>>(`${FOLDERS_API.GET(folderId)}/reconnect`)
  return response.data
}

export const foldersService = {
  listFolders,
  getFolder,
  addFolder,
  removeFolder,
  triggerSync,
  getPhotos,
  getSubFolders,
  reconnectFolder,
}
