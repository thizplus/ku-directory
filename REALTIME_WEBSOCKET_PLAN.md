# Real-time Updates with WebSocket

## Current Issues

### 1. Photo Status Not Updating
- Photos sync จาก Google Drive สำเร็จ แสดงสถานะ `pending` (face detection)
- Face worker ประมวลผลเสร็จ (ตัวเลข stats อัพเดท)
- แต่ UI ยังแสดง `pending` ไม่เปลี่ยนเป็น `completed` (เขียว)
- ต้อง refresh หน้าหรือสลับโฟลเดอร์ถึงจะเห็นสถานะใหม่

### 2. Polling Inefficiency
- ปัจจุบันใช้ polling ทุก 10 วินาที (idle) / 2 วินาที (syncing)
- เปลือง CPU และ network bandwidth
- ไม่ real-time - delay สูงสุด 10 วินาที

## Proposed Solution: WebSocket Events

### Backend มี WebSocket Infrastructure อยู่แล้ว
- `gofiber/infrastructure/websocket/websocket.go` - WebSocket Manager
- `BroadcastToUser(userID, messageType, data)` - ส่งข้อความไปยัง user เฉพาะ
- WebSocket endpoint: `ws://localhost:8080/ws`

### Events ที่ควรส่งผ่าน WebSocket

```go
// Event Types
const (
    EventSyncStarted    = "sync:started"     // Sync job เริ่มทำงาน
    EventSyncProgress   = "sync:progress"    // อัพเดท progress (files processed)
    EventSyncCompleted  = "sync:completed"   // Sync job เสร็จสิ้น
    EventPhotoAdded     = "photo:added"      // รูปใหม่ถูกเพิ่ม
    EventPhotoUpdated   = "photo:updated"    // รูปถูกอัพเดท (face status changed)
    EventFaceCompleted  = "face:completed"   // Face detection เสร็จสำหรับรูป
)
```

### Implementation Steps

#### 1. Backend - Add WebSocket Events

**File: `gofiber/application/workers/sync_worker.go`**
```go
// เมื่อ sync เริ่ม
websocket.Manager.BroadcastToUser(job.UserID, "sync:started", map[string]interface{}{
    "jobId": job.ID,
})

// เมื่อเพิ่มรูปใหม่
websocket.Manager.BroadcastToUser(userID, "photo:added", map[string]interface{}{
    "photo": photo,
})

// เมื่อ sync เสร็จ
websocket.Manager.BroadcastToUser(job.UserID, "sync:completed", map[string]interface{}{
    "jobId":          job.ID,
    "totalFiles":     job.TotalFiles,
    "processedFiles": job.ProcessedFiles,
})
```

**File: `gofiber/application/workers/face_worker.go`**
```go
// เมื่อ face detection เสร็จสำหรับรูป
websocket.Manager.BroadcastToUser(photo.UserID, "photo:updated", map[string]interface{}{
    "photoId":    photo.ID,
    "faceStatus": photo.FaceStatus,
    "faceCount":  photo.FaceCount,
})
```

#### 2. Frontend - WebSocket Hook

**File: `vite/src/hooks/useWebSocket.ts`**
```typescript
import { useEffect, useRef, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth } from './use-auth'
import { driveKeys } from '@/features/drive'
import { faceKeys } from '@/features/face-search'

type WebSocketMessage = {
  type: string
  data: any
}

export function useGalleryWebSocket() {
  const { token, user } = useAuth()
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)

  const handleMessage = useCallback((event: MessageEvent) => {
    const message: WebSocketMessage = JSON.parse(event.data)

    switch (message.type) {
      case 'sync:started':
      case 'sync:progress':
        queryClient.invalidateQueries({ queryKey: driveKeys.syncStatus() })
        break

      case 'sync:completed':
        queryClient.invalidateQueries({ queryKey: driveKeys.all })
        queryClient.invalidateQueries({ queryKey: faceKeys.stats() })
        break

      case 'photo:added':
        queryClient.invalidateQueries({ queryKey: driveKeys.photos })
        break

      case 'photo:updated':
        // Update specific photo in cache or invalidate
        queryClient.invalidateQueries({ queryKey: driveKeys.photos })
        queryClient.invalidateQueries({ queryKey: faceKeys.stats() })
        break
    }
  }, [queryClient])

  useEffect(() => {
    if (!token || !user) return

    const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`)

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = handleMessage

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      // Auto reconnect after 3 seconds
      setTimeout(() => {
        // Reconnect logic
      }, 3000)
    }

    wsRef.current = ws

    return () => {
      ws.close()
    }
  }, [token, user, handleMessage])

  return wsRef.current
}
```

#### 3. Frontend - Use in Gallery Page

**File: `vite/src/page/gallery/page.tsx`**
```typescript
import { useGalleryWebSocket } from '@/hooks/useWebSocket'

export default function GalleryPage() {
  // Connect to WebSocket for real-time updates
  useGalleryWebSocket()

  // Remove polling - no longer needed
  const { data: syncStatus } = useSyncStatus(driveStatus?.connected)

  // ... rest of component
}
```

## Benefits

| Aspect | Polling | WebSocket |
|--------|---------|-----------|
| Latency | 2-10 seconds | Instant (<100ms) |
| CPU Usage | High (constant requests) | Low (event-driven) |
| Network | Many HTTP requests | Single persistent connection |
| Battery | Drains faster | Efficient |
| Scalability | Poor | Good |

## Files to Modify

### Backend
1. `gofiber/application/workers/sync_worker.go` - Add WebSocket broadcasts
2. `gofiber/application/workers/face_worker.go` - Add WebSocket broadcasts
3. `gofiber/interfaces/api/handlers/drive_handler.go` - Broadcast on webhook

### Frontend
1. Create `vite/src/hooks/useWebSocket.ts` - WebSocket hook
2. Update `vite/src/page/gallery/page.tsx` - Remove polling, use WebSocket
3. Update `vite/src/features/drive/hooks/useDrive.ts` - Remove refetchInterval

## Priority

**High Priority** - This affects user experience significantly
- Users expect real-time updates when photos are processed
- Current polling approach is inefficient and has noticeable delay
