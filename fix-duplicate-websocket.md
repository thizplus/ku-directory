# แก้ไขปัญหา WebSocket Duplicate (StrictMode)

## ปัญหา
React StrictMode รัน useEffect 2 ครั้งใน Development mode ทำให้:
- WebSocket connect 2 ครั้ง
- ได้รับ message ซ้ำ 2 เท่า

---

## วิธีแก้ไข

### 1. Frontend: Cleanup Function ที่ถูกต้อง

```typescript
useEffect(() => {
  const socket = new WebSocket('ws://localhost:8080');

  socket.onmessage = (event) => {
    console.log("Message:", event.data);
  };

  // Cleanup: ปิด connection เก่าก่อนสร้างใหม่
  return () => {
    socket.close();
  };
}, []);
```

**Flow:**
```
รอบที่ 1 (Connect) → รอบที่ 1 (Disconnect ทันที) → รอบที่ 2 (Connect ค้างไว้)
```

ผลลัพธ์: เหลือ WebSocket แค่ **1 อัน**

---

### 2. Backend: Idempotent Design

#### 2.1 Database Constraints
```sql
-- ใช้ UNIQUE key ป้องกันข้อมูลซ้ำ
ALTER TABLE shared_folders ADD UNIQUE (drive_folder_id);
```

```go
// เช็คก่อนสร้าง
if existingFolder != nil {
    return existingFolder, nil
}
```

#### 2.2 Status Check ก่อนสร้าง Job
```go
// เช็คว่ามี Job ค้างอยู่ไหม
existingJob := syncJobRepo.FindPendingByFolder(folderID)
if existingJob != nil {
    return existingJob, nil // ไม่สร้างใหม่
}
```

#### 2.3 Upsert
```sql
INSERT INTO sync_jobs (...)
ON CONFLICT (folder_id, status)
DO UPDATE SET updated_at = NOW();
```

---

### 3. WebSocket Hub: จัดการ Connection ซ้ำ

```go
// Map connection ด้วย UserID (ไม่ใช่ connection ID)
type WebSocketManager struct {
    connections map[uuid.UUID]*websocket.Conn // key = UserID
}

func (m *WebSocketManager) Register(userID uuid.UUID, conn *websocket.Conn) {
    // ถ้ามี connection เก่า → ปิดทิ้ง
    if oldConn, exists := m.connections[userID]; exists {
        oldConn.Close()
    }
    m.connections[userID] = conn
}
```

---

## วิธีตรวจสอบว่าพลาด

1. **Network Tab**: เห็น WebSocket 2 เส้น
2. **Console**: Log พ่นซ้ำ 2 ครั้งต่อ 1 message

---

## สรุป

| Layer | วิธีแก้ |
|-------|--------|
| Frontend | Cleanup function ปิด connection เก่า |
| Backend | UNIQUE constraint + Status check + Upsert |
| WebSocket Hub | Map ด้วย UserID ไม่ใช่ Connection ID |

> "อันเก่าถูกปิด อันใหม่มาแทน" + Backend เช็คความซ้ำซ้อน = ข้อมูลถูกต้อง
