/**
 * Activity Logs Service Layer
 */
import { apiClient, type ApiResponse } from '@/shared/lib/api'
import { ACTIVITY_LOGS_API } from '@/shared/lib/api/constants/api'

/**
 * Activity Log Response Types
 */
export interface ActivityLogDetails {
  count?: number
  total_new?: number
  total_updated?: number
  total_deleted?: number
  total_failed?: number
  total_trashed?: number
  total_restored?: number
  file_names?: string[]
  folder_name?: string
  folder_path?: string
  old_folder_name?: string
  new_folder_name?: string
  drive_file_id?: string
  drive_folder_id?: string
  job_id?: string
  is_incremental?: boolean
  duration_ms?: number
  error_message?: string
  error_code?: string
  channel_id?: string
  resource_state?: string
}

export interface ActivityLog {
  id: string
  sharedFolderId: string
  activityType: string
  message: string
  details?: ActivityLogDetails
  rawData?: Record<string, unknown>
  createdAt: string
}

export interface ActivityLogMeta {
  total: number
  page: number
  limit: number
  totalPages: number
  hasNext: boolean
  hasPrev: boolean
}

export interface ActivityLogsResponse {
  data: ActivityLog[]
  meta: ActivityLogMeta
}

export interface ActivityType {
  value: string
  label: string
  category: string
}

export interface GetActivityLogsParams {
  folderId: string
  page?: number
  limit?: number
  activityType?: string
}

/**
 * Get activity logs for a folder
 */
async function getActivityLogs(params: GetActivityLogsParams): Promise<ApiResponse<ActivityLogsResponse>> {
  const { folderId, page = 1, limit = 50, activityType } = params
  const queryParams: Record<string, string | number> = {
    folderId,
    page,
    limit,
  }
  if (activityType) {
    queryParams.activityType = activityType
  }
  const response = await apiClient.get<ApiResponse<ActivityLogsResponse>>(ACTIVITY_LOGS_API.LIST, {
    params: queryParams,
  })
  // แปลงเป็น flat response
  return {
    success: response.data.success,
    message: '',
    data: {
      data: (response.data.data as unknown as ActivityLog[]) ?? [],
      meta: (response.data as unknown as { meta: ActivityLogMeta }).meta ?? {
        total: 0,
        page: 1,
        limit: 50,
        totalPages: 0,
        hasNext: false,
        hasPrev: false,
      },
    },
  }
}

/**
 * Get recent activity logs across all folders
 */
async function getRecentActivityLogs(limit = 50): Promise<ApiResponse<ActivityLog[]>> {
  const response = await apiClient.get<ApiResponse<ActivityLog[]>>(ACTIVITY_LOGS_API.RECENT, {
    params: { limit },
  })
  return response.data
}

/**
 * Get all activity types
 */
async function getActivityTypes(): Promise<ApiResponse<ActivityType[]>> {
  const response = await apiClient.get<ApiResponse<ActivityType[]>>(ACTIVITY_LOGS_API.TYPES)
  return response.data
}

export const activityLogsService = {
  getActivityLogs,
  getRecentActivityLogs,
  getActivityTypes,
}
