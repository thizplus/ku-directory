# Plan: Async Folder Sync with WebSocket Progress

## Overview
เปลี่ยนจาก Synchronous AddFolder (รอจนเสร็จ) เป็น Asynchronous (return ทันที + sync background)

## Current Flow (Synchronous)
```
User กด Add Folder
    ↓
API รอ 8+ วินาที (ดึง folders, images, save DB)
    ↓
Return response
    ↓
User เห็นข้อมูล
```

**ปัญหา**: ถ้ามี 1,000 รูป อาจใช้เวลา 1-2 นาที user ต้องรอ

---

## New Flow (Asynchronous + WebSocket)

```
User กด Add Folder
    ↓
API สร้าง shared_folder + user_access + sync_job (status: pending)
    ↓
Return response ทันที (< 1 วินาที)
    ↓
Background: SyncWorker หยิบ job ไป process
    ↓
WebSocket: ส่ง progress updates → Frontend แสดง "กำลัง sync 10/100..."
    ↓
เสร็จ: อัพเดท sync_job status = completed
    ↓
WebSocket: ส่ง sync_complete → Frontend refresh ข้อมูล
```

---

## Implementation Steps

### Step 1: Modify AddFolder Service
**File**: `gofiber/application/serviceimpl/shared_folder_service_impl.go`

```go
func (s *SharedFolderServiceImpl) AddFolder(ctx, userID, driveFolderID) (*SharedFolder, error) {
    // 1. Get Drive service
    // 2. Get folder info from Drive
    // 3. Check if folder exists → create or get existing
    // 4. Create user_folder_access
    // 5. Create sync_job with status "pending" ← สำคัญ!
    // 6. Trigger SyncWorker immediately
    // 7. Return folder ทันที (ไม่รอ sync)
}
```

**เปลี่ยนแปลง**:
- ลบ code ที่ list images และ save photos ออก
- สร้าง sync_job แล้ว return ทันที
- trigger worker ให้เริ่มทำงาน

### Step 2: Modify SyncWorker to Send WebSocket Progress
**File**: `gofiber/infrastructure/worker/sync_worker.go`

```go
func (w *SyncWorker) processJob(job *SyncJob) {
    // 1. List all folders (build path map)
    // 2. List all images

    // 3. Loop save photos พร้อมส่ง progress
    for i, photo := range photos {
        // Save photo

        // Send WebSocket progress ทุก 10 รูป หรือทุก 5%
        if i % 10 == 0 {
            w.sendProgress(job.SharedFolderID, i, len(photos))
        }
    }

    // 4. Mark job complete
    // 5. Send WebSocket: sync_complete
}
```

### Step 3: Add WebSocket Progress Events
**File**: `gofiber/infrastructure/websocket/hub.go`

เพิ่ม event types:
```go
const (
    EventSyncProgress  = "sync_progress"   // { folder_id, current, total, percent }
    EventSyncComplete  = "sync_complete"   // { folder_id, total_photos }
    EventSyncError     = "sync_error"      // { folder_id, error }
)
```

### Step 4: Frontend Handle WebSocket Events
**File**: `vite/src/page/gallery/page.tsx`

```tsx
useEffect(() => {
    // Subscribe to WebSocket
    ws.on('sync_progress', (data) => {
        // แสดง progress bar หรือ toast
        // "กำลัง sync รูปภาพ 50/100 (50%)"
    })

    ws.on('sync_complete', (data) => {
        // Refresh folder data
        queryClient.invalidateQueries(['folders'])
        // แสดง toast "Sync เสร็จแล้ว!"
    })
}, [])
```

---

## Data Flow Diagram

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │     │   Backend   │     │   Worker    │
└─────────────┘     └─────────────┘     └─────────────┘
      │                    │                    │
      │ POST /folders      │                    │
      │───────────────────>│                    │
      │                    │                    │
      │                    │ Create folder      │
      │                    │ Create sync_job    │
      │                    │ (status: pending)  │
      │                    │                    │
      │ Return { folder }  │                    │
      │<───────────────────│                    │
      │                    │                    │
      │                    │ Trigger worker     │
      │                    │───────────────────>│
      │                    │                    │
      │                    │                    │ Process job
      │                    │                    │ List folders
      │                    │                    │ List images
      │                    │                    │
      │ WS: sync_progress  │                    │
      │<───────────────────│<───────────────────│ Save batch
      │ (10/100)           │                    │
      │                    │                    │
      │ WS: sync_progress  │                    │
      │<───────────────────│<───────────────────│ Save batch
      │ (20/100)           │                    │
      │                    │                    │
      │        ...         │                    │
      │                    │                    │
      │ WS: sync_complete  │                    │
      │<───────────────────│<───────────────────│ Done!
      │                    │                    │
      │ Refresh UI         │                    │
      │                    │                    │
```

---

## Files to Modify

### Backend
| File | Changes |
|------|---------|
| `serviceimpl/shared_folder_service_impl.go` | ลบ sync code, สร้าง job แล้ว return ทันที |
| `worker/sync_worker.go` | เพิ่ม WebSocket progress, ใช้ folder path map |
| `websocket/hub.go` | เพิ่ม SendToFolder() method |

### Frontend
| File | Changes |
|------|---------|
| `page/gallery/page.tsx` | Handle sync_progress, sync_complete events |
| `services/folders/folders.service.ts` | ลบ timeout 2 นาที (ไม่จำเป็นแล้ว) |

---

## WebSocket Message Format

### sync_progress
```json
{
    "type": "sync_progress",
    "data": {
        "folder_id": "uuid",
        "current": 50,
        "total": 100,
        "percent": 50,
        "message": "กำลัง sync รูปภาพ..."
    }
}
```

### sync_complete
```json
{
    "type": "sync_complete",
    "data": {
        "folder_id": "uuid",
        "total_photos": 100,
        "message": "Sync เสร็จสมบูรณ์"
    }
}
```

### sync_error
```json
{
    "type": "sync_error",
    "data": {
        "folder_id": "uuid",
        "error": "Failed to access Google Drive"
    }
}
```

---

## Benefits

1. **User Experience**: ไม่ต้องรอ, เห็น progress แบบ real-time
2. **Scalability**: รองรับ folder ที่มีรูปเยอะๆ ได้
3. **Reliability**: ถ้า sync fail สามารถ retry ได้
4. **Visibility**: User รู้ว่า backend กำลังทำอะไร

---

## Questions / Decisions

1. ส่ง progress ทุกกี่รูป? (แนะนำ: ทุก 10 รูป หรือทุก 5%)
2. แสดง progress แบบไหนใน UI? (progress bar, toast, หรือ badge บน folder)
3. ถ้า sync fail ให้ทำอะไร? (retry auto หรือให้ user กด manual)
