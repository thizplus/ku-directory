# Server-Centric Sync Refactor Plan

## Overview

เปลี่ยนระบบ sync จาก **User-centric** (user เป็นเจ้าของ photos) เป็น **Server-centric** (server sync แล้ว share ให้ทุกคน)

---

## Current State (ปัจจุบัน)

### Architecture
```
User A ลงทะเบียน folder
    ↓
User A sync → photos.user_id = User A
    ↓
User B ลงทะเบียน folder เดียวกัน
    ↓
User B ดู photos ผ่าน folder_path (แต่ user_id ยังเป็น User A)
    ↓
Webhook → trigger sync เฉพาะ User A
    ↓
WebSocket → แจ้งเฉพาะ User A
```

### Problems
1. Photos มี `user_id` ผูกกับคนที่ sync ก่อน
2. Webhook ผูกกับ user คนเดียว
3. WebSocket broadcast เฉพาะ user ที่ sync
4. ถ้า User A ลบ account → photos หาย?
5. Logic ซับซ้อน (บางที่ใช้ user_id, บางที่ใช้ folder_path)

---

## Target State (เป้าหมาย)

### Architecture
```
Admin/System ตั้งค่า shared folder
    ↓
Server sync folder (background job)
    ↓
Photos เก็บใน DB (ไม่มี user_id หรือเป็น system_user)
    ↓
Users ลงทะเบียน folder ที่ตัวเองมีสิทธิ์
    ↓
Webhook → Server sync → broadcast to folder room
    ↓
WebSocket → แจ้งทุก user ที่อยู่ใน folder room
```

### Benefits
- Sync ครั้งเดียว ได้ทุกคน
- ไม่มี duplicate data
- Real-time updates สำหรับทุกคน
- ไม่มีปัญหา ownership
- Code cleaner (ใช้ folder_path เป็น key หลัก)

---

## Database Changes

### Option A: Remove user_id from photos (Recommended)

```sql
-- photos table
ALTER TABLE photos DROP COLUMN user_id;
-- หรือเปลี่ยนเป็น nullable และไม่ใช้
ALTER TABLE photos ALTER COLUMN user_id DROP NOT NULL;
```

### Option B: Keep user_id but use system user

```sql
-- สร้าง system user สำหรับ server-owned photos
INSERT INTO users (id, email, name) VALUES
  ('00000000-0000-0000-0000-000000000000', 'system@local', 'System');
```

### New Table: shared_folders

```sql
CREATE TABLE shared_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drive_folder_id VARCHAR(255) NOT NULL UNIQUE,
    drive_folder_name VARCHAR(255) NOT NULL,
    drive_folder_path VARCHAR(1024) NOT NULL,

    -- Webhook info
    webhook_channel_id VARCHAR(255),
    webhook_resource_id VARCHAR(255),
    webhook_token VARCHAR(255),
    webhook_expiry TIMESTAMP,

    -- Sync info
    page_token VARCHAR(255),
    last_synced_at TIMESTAMP,
    sync_status VARCHAR(50) DEFAULT 'idle', -- idle, syncing, error

    -- OAuth tokens (for sync - from first user who added)
    drive_access_token TEXT,
    drive_refresh_token TEXT,
    drive_token_expiry TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for quick lookup
CREATE INDEX idx_shared_folders_path ON shared_folders(drive_folder_path);
```

### New Table: user_folder_access

```sql
CREATE TABLE user_folder_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_folder_id UUID NOT NULL REFERENCES shared_folders(id) ON DELETE CASCADE,

    -- User's root within this folder (for sub-folder access)
    root_path VARCHAR(1024),

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(user_id, shared_folder_id)
);

CREATE INDEX idx_user_folder_access_user ON user_folder_access(user_id);
```

### Update photos table

```sql
-- Add reference to shared_folder instead of user
ALTER TABLE photos ADD COLUMN shared_folder_id UUID REFERENCES shared_folders(id);

-- Migrate existing data
UPDATE photos p
SET shared_folder_id = sf.id
FROM shared_folders sf
WHERE p.drive_folder_path LIKE sf.drive_folder_path || '%';
```

### Update faces table

```sql
-- faces ยังคง link กับ photos ผ่าน photo_id
-- ไม่ต้องเปลี่ยนอะไร แต่ต้องเอา user_id check ออก
```

---

## Backend Changes

### 1. New Domain Models

**File:** `gofiber/domain/models/shared_folder.go`
```go
type SharedFolder struct {
    ID              uuid.UUID
    DriveFolderID   string
    DriveFolderName string
    DriveFolderPath string

    // Webhook
    WebhookChannelID  string
    WebhookResourceID string
    WebhookToken      string
    WebhookExpiry     *time.Time

    // Sync
    PageToken    string
    LastSyncedAt *time.Time
    SyncStatus   string

    // OAuth (for sync)
    DriveAccessToken  string
    DriveRefreshToken string
    DriveTokenExpiry  *time.Time

    CreatedAt time.Time
    UpdatedAt time.Time
}

type UserFolderAccess struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    SharedFolderID uuid.UUID
    RootPath       string
    CreatedAt      time.Time
}
```

### 2. New Repositories

**File:** `gofiber/domain/repositories/shared_folder_repository.go`
```go
type SharedFolderRepository interface {
    Create(ctx context.Context, folder *models.SharedFolder) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.SharedFolder, error)
    GetByDriveFolderID(ctx context.Context, driveFolderID string) (*models.SharedFolder, error)
    GetByWebhookToken(ctx context.Context, token string) (*models.SharedFolder, error)
    GetAll(ctx context.Context) ([]models.SharedFolder, error)
    Update(ctx context.Context, id uuid.UUID, folder *models.SharedFolder) error
    Delete(ctx context.Context, id uuid.UUID) error

    // User access
    AddUserAccess(ctx context.Context, access *models.UserFolderAccess) error
    RemoveUserAccess(ctx context.Context, userID, folderID uuid.UUID) error
    GetUsersByFolder(ctx context.Context, folderID uuid.UUID) ([]models.User, error)
    GetFoldersByUser(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error)
}
```

### 3. Update Services

#### DriveService Changes
```go
// เปลี่ยนจาก user-based เป็น folder-based
type DriveService interface {
    // Folder management
    AddSharedFolder(ctx context.Context, userID uuid.UUID, folderID string) (*models.SharedFolder, error)
    RemoveSharedFolder(ctx context.Context, folderID uuid.UUID) error
    GetUserFolders(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error)

    // Sync (server-level)
    SyncFolder(ctx context.Context, folderID uuid.UUID) error
    SyncAllFolders(ctx context.Context) error

    // Webhook (folder-level)
    HandleWebhook(ctx context.Context, token string, resourceState string) error
}
```

#### PhotoService Changes
```go
// Query by folder path instead of user_id
type PhotoService interface {
    GetPhotos(ctx context.Context, folderPath string, page, limit int) ([]models.Photo, int64, error)
    GetPhotosByFolder(ctx context.Context, folderID uuid.UUID, page, limit int) ([]models.Photo, int64, error)
    // ... etc
}
```

#### FaceService Changes
```go
// Query by folder path instead of user_id
type FaceService interface {
    SearchByImage(ctx context.Context, folderPath string, imageData []byte, ...) ([]FaceSearchResult, error)
    GetProcessingStats(ctx context.Context, folderPath string) (*FaceProcessingStats, error)
    // ... etc
}
```

### 4. Update Sync Worker

**File:** `gofiber/infrastructure/worker/sync_worker.go`

```go
// เปลี่ยนจาก sync per user เป็น sync per folder
func (w *SyncWorker) processJob(job models.SyncJob) {
    // Get shared folder instead of user
    folder, err := w.sharedFolderRepo.GetByID(ctx, job.FolderID)

    // Sync using folder's OAuth tokens
    srv, err := w.driveClient.GetDriveService(ctx,
        folder.DriveAccessToken,
        folder.DriveRefreshToken,
        folder.DriveTokenExpiry)

    // Process sync...

    // Broadcast to ALL users with access to this folder
    users, _ := w.sharedFolderRepo.GetUsersByFolder(ctx, folder.ID)
    for _, user := range users {
        websocket.Manager.BroadcastToUser(user.ID, "photos:added", data)
    }

    // OR better: Broadcast to room
    websocket.Manager.BroadcastToRoom(folder.DriveFolderPath, "photos:added", data)
}
```

### 5. Update WebSocket

**File:** `gofiber/infrastructure/websocket/websocket.go`

```go
// เพิ่ม room-based broadcast
func (m *WebSocketManager) BroadcastToRoom(roomID string, messageType string, data interface{}) {
    // roomID = folder path
    // broadcast to all clients in this room
}

// Client joins room when connecting
func (m *WebSocketManager) JoinRoom(clientID, roomID string) {
    // Add client to room
}
```

### 6. Update Handlers

#### DriveHandler
```go
// Webhook ใช้ folder token แทน user token
func (h *DriveHandler) Webhook(c *fiber.Ctx) error {
    token := payload.ChannelToken

    // Find folder by webhook token
    folder, err := h.sharedFolderRepo.GetByWebhookToken(ctx, token)

    // Trigger sync for folder
    h.driveService.SyncFolder(ctx, folder.ID)
}
```

#### PhotoHandler
```go
// Query by folder path (from user's access)
func (h *PhotoHandler) GetPhotos(c *fiber.Ctx) error {
    userID := middleware.GetUserID(c)

    // Get user's folder access
    folders, _ := h.sharedFolderRepo.GetFoldersByUser(ctx, userID)

    // Get photos from user's accessible folders
    // ...
}
```

---

## Frontend Changes

### 1. WebSocket Connection

**File:** `vite/src/hooks/use-websocket.ts`

```typescript
// Join room based on user's folder
const connectWebSocket = (folderPath: string) => {
    ws.send(JSON.stringify({
        type: 'join_room',
        room: folderPath
    }))
}

// Listen for folder-level events
ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    if (data.type === 'photos:added') {
        // Refresh gallery
        queryClient.invalidateQueries(['photos'])
    }
}
```

### 2. Gallery Page

```typescript
// ไม่ต้องเปลี่ยนมาก เพราะ API จะ return photos ตาม user's folder access
const { data: photos } = usePhotos({ page, limit })
```

### 3. Settings Page

```typescript
// เพิ่ม UI สำหรับ manage shared folders
const { data: folders } = useUserFolders()

// Add new folder
const addFolder = useMutation({
    mutationFn: (folderId: string) => driveService.addSharedFolder(folderId)
})
```

---

## Migration Plan

### Phase 1: Database Migration
1. สร้าง `shared_folders` table
2. สร้าง `user_folder_access` table
3. Migrate ข้อมูลจาก `users.drive_root_folder_*` ไป `shared_folders`
4. สร้าง `user_folder_access` records จากข้อมูลเดิม
5. Add `shared_folder_id` to `photos` table
6. Update existing photos with correct `shared_folder_id`

### Phase 2: Backend Updates
1. สร้าง new models และ repositories
2. Update services ให้ใช้ folder-based queries
3. Update sync worker ให้ sync per folder
4. Update webhook handler
5. Update WebSocket ให้ support rooms

### Phase 3: Frontend Updates
1. Update WebSocket connection ให้ join room
2. Update API calls (ถ้ามีการเปลี่ยน endpoints)
3. Add folder management UI (optional)

### Phase 4: Cleanup
1. Remove unused user fields (`drive_root_folder_*`, `drive_webhook_*`)
2. Remove `user_id` checks ที่ไม่จำเป็น
3. Update tests

---

## API Changes

### New Endpoints

```
POST   /api/v1/folders              - Add shared folder
GET    /api/v1/folders              - Get user's folders
DELETE /api/v1/folders/:id          - Remove folder access
POST   /api/v1/folders/:id/sync     - Trigger sync for folder
```

### Modified Endpoints

```
GET    /api/v1/photos               - Query by user's folder access (no change in interface)
GET    /api/v1/drive/status         - Return folder-based status
POST   /api/v1/drive/webhook        - Handle folder webhook (internal change)
```

---

## Risks & Considerations

### 1. OAuth Token Management
- Folder needs valid OAuth tokens for sync
- What if user who provided tokens revokes access?
- **Solution**: Store tokens in shared_folder, allow any user with access to refresh

### 2. Backward Compatibility
- Existing photos have `user_id`
- Migration must preserve data
- **Solution**: Keep `user_id` as nullable, migrate gradually

### 3. Permission Model
- Who can add/remove shared folders?
- Who can see which folders?
- **Solution**: Any user can add folder they have Drive access to

### 4. WebSocket Rooms
- Need to track which rooms each client is in
- Client needs to join room on connect
- **Solution**: Auto-join based on user's folder access

---

## Estimated Effort

| Task | Complexity | Files |
|------|------------|-------|
| Database migration | Medium | 1-2 migration files |
| New models/repos | Low | 3-4 new files |
| Update services | High | 5-6 files |
| Update sync worker | High | 1 file (major changes) |
| Update handlers | Medium | 3-4 files |
| Update WebSocket | Medium | 1-2 files |
| Frontend updates | Low | 2-3 files |
| Testing | High | Multiple files |

---

## Final Decisions

1. **Single folder or multiple folders per user?**
   - **ตอบ**: User สามารถเพิ่มได้หลาย folders

2. **Admin control?**
   - **ตอบ**: ไม่ต้อง - Google Drive จัดการ permission อยู่แล้ว

3. **Token refresh strategy?**
   - **ตอบ**: ใช้ Option A - tokens ของ user แรกที่เพิ่ม folder
   - ถ้า token หมดอายุ → แจ้ง users ผ่าน WebSocket
   - User คนไหนก็ได้ที่มี access กด "Reconnect" ได้

4. **Existing data?**
   - **ตอบ**: ลบทิ้งหมด แล้วทดสอบใหม่
