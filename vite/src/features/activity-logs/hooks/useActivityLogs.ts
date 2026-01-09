/**
 * Activity Logs Hooks - React Query hooks for activity logs feature
 */
import { useQuery } from '@tanstack/react-query'
import { activityLogsService } from '@/services'
import { CACHE_TIMES } from '@/shared/config/constants'
import type { GetActivityLogsParams } from '@/services/activity-logs'

/**
 * Query Keys
 */
export const activityLogsKeys = {
  all: ['activity-logs'] as const,
  list: (params: GetActivityLogsParams) => [...activityLogsKeys.all, 'list', params] as const,
  recent: (limit: number) => [...activityLogsKeys.all, 'recent', limit] as const,
  types: () => [...activityLogsKeys.all, 'types'] as const,
}

/**
 * Get activity logs for a folder
 */
export function useActivityLogs(params: GetActivityLogsParams, enabled = true) {
  return useQuery({
    queryKey: activityLogsKeys.list(params),
    queryFn: async () => {
      const response = await activityLogsService.getActivityLogs(params)
      if (!response.success) {
        throw new Error('Failed to fetch activity logs')
      }
      return response.data
    },
    enabled: enabled && !!params.folderId,
    staleTime: CACHE_TIMES.STATS, // Activity logs should refresh more frequently
  })
}

/**
 * Get recent activity logs across all folders
 */
export function useRecentActivityLogs(limit = 50, enabled = true) {
  return useQuery({
    queryKey: activityLogsKeys.recent(limit),
    queryFn: async () => {
      const response = await activityLogsService.getRecentActivityLogs(limit)
      if (!response.success) {
        throw new Error('Failed to fetch recent activity logs')
      }
      return response.data
    },
    enabled,
    staleTime: CACHE_TIMES.STATS,
  })
}

/**
 * Get all activity types
 */
export function useActivityTypes() {
  return useQuery({
    queryKey: activityLogsKeys.types(),
    queryFn: async () => {
      const response = await activityLogsService.getActivityTypes()
      if (!response.success) {
        throw new Error('Failed to fetch activity types')
      }
      return response.data
    },
    staleTime: CACHE_TIMES.PHOTOS, // Activity types don't change often
  })
}
