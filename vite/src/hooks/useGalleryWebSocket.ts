/**
 * WebSocket hook for real-time gallery updates
 * Listens for sync and photo events, invalidates queries accordingly
 * Uses singleton pattern to prevent duplicate connections in StrictMode
 */
import { useEffect, useRef, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth } from './use-auth'
import { driveKeys } from '@/features/drive'
import { foldersKeys } from '@/features/folders'
import { useDownloadProgressStore } from '@/stores/download-progress'
import { useSyncProgressStore } from '@/stores/sync-progress'
import { useFaceStatsStore } from '@/stores/face-stats'

type WebSocketMessage = {
  type: string
  data: Record<string, unknown>
}

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:3010/ws'

// Singleton WebSocket instance to prevent duplicate connections
let globalWs: WebSocket | null = null
let globalReconnectTimeout: NodeJS.Timeout | null = null
let cleanupTimeout: NodeJS.Timeout | null = null

export function useGalleryWebSocket() {
  const { token, user } = useAuth()
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)
  const maxReconnectAttempts = 5

  // Download progress store actions
  const setDownloadProgress = useDownloadProgressStore((state) => state.setProgress)
  const setDownloadCompleted = useDownloadProgressStore((state) => state.setCompleted)

  // Sync progress store actions
  const setSyncProgress = useSyncProgressStore((state) => state.setProgress)
  const clearSyncProgress = useSyncProgressStore((state) => state.clearProgress)

  // Face stats store actions
  const onPhotoProcessed = useFaceStatsStore((state) => state.onPhotoProcessed)

  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)
      console.log('WebSocket message:', message.type, message.data)

      switch (message.type) {
        case 'sync:started':
          // Sync job started - invalidate stats to get fresh data
          console.log('Sync started:', message.data)
          queryClient.invalidateQueries({ queryKey: ['faces', 'stats'] })
          queryClient.invalidateQueries({ queryKey: foldersKeys.all })
          break

        case 'sync:progress':
          // Sync progress update - only update UI store (no query invalidation needed)
          {
            const folderId = message.data.folderId as string
            const percent = message.data.percent as number
            const processedFiles = message.data.processedFiles as number
            const totalFiles = message.data.totalFiles as number
            if (folderId) {
              setSyncProgress(folderId, { percent, processedFiles, totalFiles })
            }
          }
          break

        case 'sync:completed':
          // Sync completed - clear progress and invalidate queries
          {
            const folderId = message.data.folderId as string
            if (folderId) {
              clearSyncProgress(folderId)
            }
          }
          queryClient.invalidateQueries({ queryKey: driveKeys.all })
          queryClient.invalidateQueries({ queryKey: foldersKeys.all })
          queryClient.invalidateQueries({ queryKey: ['faces', 'stats'] })
          break

        case 'photos:added':
          // Photos added during sync - invalidate stats to show correct pending count
          console.log('Photos added:', message.data.count)
          queryClient.invalidateQueries({ queryKey: ['faces', 'stats'] })
          break

        case 'photo:updated':
          // Photo updated (face detection) - update stats store directly
          {
            const faceCount = message.data.faceCount as number
            const faceStatus = message.data.faceStatus as 'completed' | 'failed'
            if (faceStatus === 'completed' || faceStatus === 'failed') {
              onPhotoProcessed(faceCount || 0, faceStatus)
            }
          }
          break

        case 'pong':
          // Heartbeat response - ignore
          break

        case 'download:progress':
          // Download progress update
          setDownloadProgress({
            current: message.data.current as number,
            total: message.data.total as number,
            fileName: message.data.fileName as string,
          })
          break

        case 'download:completed':
          // Download completed (server finished creating zip)
          // Toast is handled in the mutation onSuccess after file is saved
          setDownloadCompleted()
          break

        default:
          console.log('Unknown WebSocket message type:', message.type)
      }
    } catch (error) {
      console.error('Error parsing WebSocket message:', error)
    }
  }, [queryClient, setDownloadProgress, setDownloadCompleted, setSyncProgress, clearSyncProgress, onPhotoProcessed])

  const connect = useCallback(() => {
    if (!token || !user) return

    // Use singleton - if already connected or connecting, just update the ref
    if (globalWs && (globalWs.readyState === WebSocket.OPEN || globalWs.readyState === WebSocket.CONNECTING)) {
      wsRef.current = globalWs
      // Update message handler to use latest callback
      globalWs.onmessage = handleMessage
      return
    }

    // Clean up existing connection
    if (globalWs) {
      globalWs.close()
      globalWs = null
    }

    try {
      const ws = new WebSocket(`${WS_URL}?token=${token}`)

      ws.onopen = () => {
        console.log('WebSocket connected (singleton)')
        reconnectAttempts.current = 0
      }

      ws.onmessage = handleMessage

      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason)
        globalWs = null
        wsRef.current = null

        // Auto reconnect with exponential backoff
        if (reconnectAttempts.current < maxReconnectAttempts) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000)
          reconnectAttempts.current++
          console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`)

          globalReconnectTimeout = setTimeout(() => {
            connect()
          }, delay)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      globalWs = ws
      wsRef.current = ws
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
    }
  }, [token, user, handleMessage])

  // Setup heartbeat to keep connection alive
  useEffect(() => {
    const heartbeatInterval = setInterval(() => {
      if (globalWs?.readyState === WebSocket.OPEN) {
        globalWs.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000) // Ping every 30 seconds

    return () => clearInterval(heartbeatInterval)
  }, [])

  // Connect on mount with delayed cleanup for StrictMode
  useEffect(() => {
    // Cancel any pending cleanup (StrictMode re-mount)
    if (cleanupTimeout) {
      clearTimeout(cleanupTimeout)
      cleanupTimeout = null
    }

    connect()

    return () => {
      // Delay cleanup to handle StrictMode double-mount
      // If component re-mounts quickly, cleanup will be cancelled
      cleanupTimeout = setTimeout(() => {
        if (globalReconnectTimeout) {
          clearTimeout(globalReconnectTimeout)
          globalReconnectTimeout = null
        }
        if (globalWs) {
          globalWs.close()
          globalWs = null
        }
        cleanupTimeout = null
      }, 100)
    }
  }, [connect])

  return {
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
    reconnect: connect,
  }
}
