# ระบบ Sync Google Drive

## ภาพรวมการทำงาน

ระบบ Sync ทำหน้าที่ซิงค์รูปภาพจาก Google Drive มายังฐานข้อมูล โดยมี 2 โหมดการทำงาน:

### 1. Incremental Sync (ซิงค์แบบเพิ่มเติม)
- ใช้ Google Drive Changes API
- ดึงเฉพาะไฟล์ที่มีการเปลี่ยนแปลงตั้งแต่ครั้งล่าสุด
- **เร็วมาก** - ไม่ต้องสแกนไฟล์ทั้งหมด
- ใช้ `PageToken` เก็บตำแหน่งที่อ่านล่าสุด (เก็บใน DB)
- ทำงานเมื่อ: มี PageToken อยู่แล้ว (ไม่ใช่ครั้งแรก)

### 2. Full Sync (ซิงค์แบบเต็ม)
- สแกนไฟล์ทั้งหมดใน Root Folder
- เปรียบเทียบกับฐานข้อมูล
- **ลบรูปที่ไม่มีใน Drive แล้ว** (Cleanup orphaned photos)
- ทำงานเมื่อ: ครั้งแรก หรือ PageToken ไม่ถูกต้อง

---

## Flow การทำงาน

```
Server Start
    │
    ├── Auto Sync on Startup (สำหรับ user ที่เชื่อมต่อ Drive แล้ว)
    │
    ▼
Sync Worker รับ Job
    │
    ├── มี PageToken? ──Yes──► Incremental Sync
    │                              │
    │                              ├── ดึง Changes จาก Drive
    │                              ├── เพิ่ม/อัพเดท/ลบรูป
    │                              └── บันทึก PageToken ใหม่
    │
    └── ไม่มี PageToken ──► Full Sync
                               │
                               ├── ดึงรูปทั้งหมดจาก Drive
                               ├── เพิ่ม/อัพเดทรูปในฐานข้อมูล
                               ├── ลบรูปที่ไม่มีใน Drive (Cleanup)
                               └── บันทึก PageToken สำหรับครั้งหน้า
```

---

## การจัดการกรณี Server ดับ

### ปัญหาเดิม
เมื่อ Server ดับ และมีการลบไฟล์/โฟลเดอร์ใน Google Drive:
- Changes API จะ "consume" การเปลี่ยนแปลงไปแล้ว
- Incremental Sync จะเห็น 0 changes
- รูปยังคงอยู่ในฐานข้อมูล (ไม่ถูกลบ)

### วิธีแก้ไข
Full Sync จะทำ **Cleanup** โดย:
1. รวบรวม Drive File ID ทั้งหมดที่ยังมีอยู่
2. ลบรูปในฐานข้อมูลที่ไม่อยู่ในรายการนี้
3. ลบ Faces ที่เกี่ยวข้องก่อน (เพื่อไม่ให้ติด Foreign Key)

---

## ไฟล์ที่เกี่ยวข้อง

| ไฟล์ | หน้าที่ |
|------|--------|
| `infrastructure/worker/sync_worker.go` | Logic หลักของการ Sync |
| `infrastructure/postgres/photo_repository_impl.go` | CRUD รูปภาพ + Cleanup |
| `pkg/di/container.go` | Auto Sync on Startup |
| `domain/models/user.go` | เก็บ `DrivePageToken` |

---

## Functions สำคัญ

### sync_worker.go

```go
// ตัดสินใจว่าจะใช้ Sync แบบไหน
if user.DrivePageToken != "" {
    processIncrementalSync(...)  // มี token = incremental
} else {
    processFullSync(...)         // ไม่มี token = full
}
```

### processFullSync()
```go
// 1. ดึงรูปทั้งหมดจาก Drive
files := ListAllImagesRecursive(rootFolderID)

// 2. เก็บ Drive File ID ทั้งหมด
driveFileIDs := []string{}
for _, file := range files {
    driveFileIDs = append(driveFileIDs, file.ID)
}

// 3. Process แต่ละไฟล์ (เพิ่ม/อัพเดท)
...

// 4. Cleanup - ลบรูปที่ไม่มีใน Drive แล้ว
deletedCount := DeleteNotInDriveIDs(userID, driveFileIDs)
```

### processIncrementalSync()
```go
// 1. ดึง Changes ตั้งแต่ PageToken ล่าสุด
changes := GetChanges(pageToken)

// 2. Process แต่ละ Change
for _, change := range changes {
    if change.Removed || change.File.Trashed {
        // ลบรูป + ลบรูปในโฟลเดอร์นั้น
        DeleteByDriveFileID(change.FileId)
        DeleteByFolderID(userID, change.FileId)
    } else {
        // เพิ่ม/อัพเดทรูป
    }
}

// 3. บันทึก PageToken ใหม่
SavePageToken(newPageToken)
```

---

## การ Cleanup รูปที่ถูกลบ

### DeleteNotInDriveIDs()
```go
func DeleteNotInDriveIDs(userID, driveFileIDs) {
    // Transaction เพื่อลบ Faces ก่อน แล้วค่อยลบ Photos
    Transaction(func(tx) {
        // 1. หา Photo IDs ที่จะลบ
        photoIDs := SELECT id FROM photos
                    WHERE user_id = ? AND drive_file_id NOT IN (?)

        // 2. ลบ Faces ก่อน (Foreign Key)
        DELETE FROM faces WHERE photo_id IN (photoIDs)

        // 3. ลบ Photos
        DELETE FROM photos WHERE id IN (photoIDs)
    })
}
```

### DeleteByFolderID()
```go
func DeleteByFolderID(userID, folderID) {
    // เมื่อโฟลเดอร์ถูกลบ ต้องลบรูปทั้งหมดในโฟลเดอร์นั้น
    Transaction(func(tx) {
        photoIDs := SELECT id FROM photos
                    WHERE user_id = ? AND drive_folder_id = ?

        DELETE FROM faces WHERE photo_id IN (photoIDs)
        DELETE FROM photos WHERE id IN (photoIDs)
    })
}
```

---

## Auto Sync on Startup

```go
// pkg/di/container.go
func autoSyncOnStartup() {
    users := GetAllUsers()

    for _, user := range users {
        // เช็คว่า user เชื่อมต่อ Drive แล้ว
        if user.DriveRefreshToken != "" && user.DriveRootFolderID != "" {
            // เช็คว่าไม่มี job pending/running อยู่
            existingJob := GetLatestJob(userID)
            if existingJob.Status == "pending" || "running" {
                continue
            }

            // สร้าง Sync Job ใหม่
            CreateSyncJob(userID)
        }
    }
}
```

---

## สถานะ Sync Job

| Status | ความหมาย |
|--------|----------|
| `pending` | รอ Worker มารับ |
| `running` | กำลังทำงาน |
| `completed` | เสร็จสมบูรณ์ |
| `failed` | ล้มเหลว |

---

## WebSocket Events

| Event | ส่งเมื่อ |
|-------|---------|
| `sync:started` | เริ่ม Sync |
| `sync:progress` | อัพเดท Progress |
| `sync:completed` | Sync เสร็จ |
| `sync:failed` | Sync ล้มเหลว |
| `photos:added` | มีรูปใหม่ |
| `photos:deleted` | รูปถูกลบ |

---

## ตัวอย่าง Log

```
✓ Sync worker started
Processing 1 sync jobs
Starting sync job xxx for user yyy
Using full sync for job xxx (no page token)
Found 30 images to sync for job xxx
Sync job xxx progress: 30/30 (new: 0, failed: 0)
Cleaned up 4 orphaned photos for user yyy
Full sync job xxx completed: 30 processed, 0 new, 4 deleted, 0 failed
```

---

## การบังคับ Full Sync (Manual)

หากต้องการบังคับให้ทำ Full Sync:

```sql
-- ล้าง PageToken ของ user
UPDATE users SET drive_page_token = '' WHERE id = 'user-id';

-- สร้าง Sync Job ใหม่
INSERT INTO sync_jobs (id, user_id, job_type, status, ...)
VALUES (gen_random_uuid(), 'user-id', 'drive_sync', 'pending', ...);
```

---

## สรุป

| Feature | สถานะ |
|---------|-------|
| Incremental Sync (Changes API) | ✅ ทำงานได้ |
| Full Sync with Cleanup | ✅ ทำงานได้ |
| Auto Sync on Startup | ✅ ทำงานได้ |
| ลบโฟลเดอร์ขณะ Server ดับ | ✅ แก้ไขแล้ว |
| ลบ Faces ก่อนลบ Photos | ✅ แก้ไขแล้ว |
