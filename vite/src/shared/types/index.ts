/**
 * Shared Type Definitions
 */

// User
export interface User {
  id: string
  email: string
  username: string
  firstName: string
  lastName: string
  avatar: string
  role: string
  provider: string
}

// Face Detection (for selecting faces in uploaded images)
export interface DetectedFace {
  index: number
  bbox_x: number
  bbox_y: number
  bbox_width: number
  bbox_height: number
  confidence: number
}

export interface DetectedFacesData {
  faces: DetectedFace[]
  count: number
}

// Face Search
export interface FaceSearchResult {
  face_id: string
  photo_id: string
  shared_folder_id: string
  drive_file_id: string
  drive_folder_id: string
  file_name: string
  thumbnail_url: string
  web_view_url: string
  folder_path: string
  bbox_x: number
  bbox_y: number
  bbox_width: number
  bbox_height: number
  similarity: number
}

export interface FaceSearchData {
  results: FaceSearchResult[]
  count: number
  limit: number
  threshold: number
}

export interface FaceProcessingStats {
  total_photos: number
  processed_photos: number
  pending_photos: number
  failed_photos: number
  total_faces: number
}

export interface Face {
  id: string
  photo_id: string
  bbox_x: number
  bbox_y: number
  bbox_width: number
  bbox_height: number
  confidence: number
  person_id: string | null
  created_at: string
}

export interface FaceListData {
  faces: Face[]
  total: number
  page: number
  limit: number
}

// Photo
export interface Photo {
  id: string
  user_id: string
  drive_file_id: string
  file_name: string
  mime_type: string
  thumbnail_url: string
  web_view_url: string
  drive_folder_path: string
  face_status: 'pending' | 'processing' | 'completed' | 'failed'
  face_count: number
  created_at: string
}

// Request Types
export interface FaceSearchByImageRequest {
  imageFile: File
  limit?: number
  threshold?: number
}

export interface FaceSearchByFaceRequest {
  face_id: string
  limit?: number
  threshold?: number
}

export interface PaginationRequest {
  page?: number
  limit?: number
}

// Drive
export interface DriveStatus {
  connected: boolean
  rootFolder: DriveFolder | null
}

export interface DriveFolder {
  id: string
  name: string
  parents?: string[]
}

export interface SyncJob {
  id: string
  user_id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  total_files: number
  processed_files: number
  failed_files: number
  started_at: string | null
  completed_at: string | null
  error_message: string | null
  created_at: string
}

export interface PhotoListData {
  photos: Photo[]
  total: number
  page: number
  limit: number
}

// Sub-folder info (child folder within a shared folder)
export interface SubFolderInfo {
  path: string
  name: string
  photo_count: number
}

// Shared Folder
export interface SharedFolder {
  id: string
  drive_folder_id: string
  drive_folder_name: string
  drive_folder_path: string
  sync_status: 'idle' | 'syncing' | 'completed' | 'failed'
  last_sync_at: string | null
  last_sync_error: string | null
  photo_count: number
  user_count: number
  children?: SubFolderInfo[]
  created_at: string
  // Webhook status
  webhook_status: 'active' | 'expiring' | 'expired' | 'inactive'
  webhook_expiry: string | null
}

export interface SharedFolderListData {
  folders: SharedFolder[]
}
