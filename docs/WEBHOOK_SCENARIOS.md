# Google Drive Webhook Scenarios

## Overview

Google Drive Push Notifications (webhooks) ใช้สำหรับรับการแจ้งเตือนแบบ real-time เมื่อมีการเปลี่ยนแปลงใน Google Drive
Webhook มีอายุสูงสุด **7 วัน** และ Google **ไม่ส่งการแจ้งเตือน** ก่อนหมดอายุหรือถูก revoke

---

## Webhook Lifecycle

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Register       │────▶│  Active         │────▶│  Expired/       │
│  (WatchChanges) │     │  (Max 7 days)   │     │  Revoked        │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │                        │
                              ▼                        ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │  Auto-Renewal   │     │  Manual         │
                        │  (Every 6 hrs)  │     │  Reconnect      │
                        └─────────────────┘     └─────────────────┘
```

---

## Scenarios ที่ทำให้ Webhook ใช้งานไม่ได้

### 1. Webhook Expiration (หมดอายุปกติ)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | Webhook ครบอายุ 7 วัน |
| **การตรวจจับ** | เช็คจาก `webhook_expiry` ใน database |
| **การแก้ไข** | Auto-renewal (ทำงานทุก 6 ชม.) |
| **ผลกระทบ** | DB ไม่อัพเดทจนกว่าจะ renew สำเร็จ |

**วิธีทดสอบ:**
```sql
-- ตั้ง webhook_expiry เป็นเวลาในอดีต
UPDATE shared_folders
SET webhook_expiry = NOW() - INTERVAL '1 day'
WHERE id = '<folder_id>';
```
รอ scheduler รัน (ทุก 6 ชม.) หรือ trigger manual

---

### 2. User Revokes App Access (ผู้ใช้ถอน permission)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | User ไปที่ Google Account → Security → Third-party apps → ลบ app |
| **การตรวจจับ** | API call fail ด้วย error `invalid_grant` หรือ `Token has been revoked` |
| **การแก้ไข** | User ต้อง Reconnect (OAuth flow ใหม่) |
| **ผลกระทบ** | DB ไม่อัพเดท, Sync fail, Webhook หยุดทำงาน |

**วิธีทดสอบ:**
1. ไปที่ https://myaccount.google.com/permissions
2. หา app "KU Directory" (หรือชื่อ app)
3. คลิก "Remove Access"
4. กลับมาที่ app แล้วลอง sync หรือรอ webhook renewal

**Expected Behavior:**
- Sync fail พร้อม error message
- `sync_status` = `failed`
- แสดง Reconnect button ใน UI

---

### 3. Refresh Token Expired (Token หมดอายุถาวร)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | ไม่ได้ใช้งาน > 6 เดือน, User เปลี่ยน password, หรือ Google revoke |
| **การตรวจจับ** | Refresh token fail ด้วย `invalid_grant` |
| **การแก้ไข** | User ต้อง Reconnect |
| **ผลกระทบ** | เหมือน scenario 2 |

**วิธีทดสอบ:**
```sql
-- ทำให้ refresh token ใช้ไม่ได้
UPDATE shared_folders
SET drive_refresh_token = 'invalid_token_12345'
WHERE id = '<folder_id>';
```
จากนั้น trigger sync หรือรอ webhook renewal

---

### 4. Watched Folder Deleted (โฟลเดอร์ถูกลบ)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | User ลบ Google Drive folder ที่ watch อยู่ |
| **การตรวจจับ** | API call fail ด้วย `File not found` (404) |
| **การแก้ไข** | ลบ folder จากระบบ หรือเลือก folder ใหม่ |
| **ผลกระทบ** | Sync fail, Webhook ยังอยู่แต่ไม่มี notification มา |

**วิธีทดสอบ:**
1. สร้าง test folder ใน Google Drive
2. Add folder ในระบบ
3. ลบ folder ใน Google Drive (ย้ายไปถังขยะ)
4. ลอง sync

**Expected Behavior:**
- Sync fail พร้อม error "Folder not found"
- แสดง error ใน UI

---

### 5. Permission Revoked (สิทธิ์ถูกถอน)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | เจ้าของ folder ถอดสิทธิ์ user ออกจาก shared folder |
| **การตรวจจับ** | API call fail ด้วย `The user does not have sufficient permissions` (403) |
| **การแก้ไข** | ขอสิทธิ์ใหม่ หรือใช้ account ที่มีสิทธิ์ |
| **ผลกระทบ** | Sync fail |

**วิธีทดสอบ:**
1. Share folder กับ test account
2. Add folder ในระบบด้วย test account
3. ถอด test account ออกจาก shared folder
4. ลอง sync

---

### 6. Google API Quota Exceeded

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | เรียก API เกิน quota ต่อวัน |
| **การตรวจจับ** | API fail ด้วย `Rate Limit Exceeded` (429) หรือ `User Rate Limit Exceeded` |
| **การแก้ไข** | รอ quota reset (24 ชม.) หรือ request quota เพิ่ม |
| **ผลกระทบ** | Sync/Webhook registration fail ชั่วคราว |

**วิธีทดสอบ:**
- ยากที่จะทดสอบโดยตรง
- อาจ mock API response

---

### 7. Manual Channel Stop (หยุด webhook เอง)

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | Code เรียก `channels.stop()` |
| **การตรวจจับ** | ควบคุมได้จาก code |
| **การแก้ไข** | Register webhook ใหม่ |
| **ผลกระทบ** | ไม่ร้ายแรงถ้าตั้งใจทำ |

---

### 8. Network/Server Issues

| รายละเอียด | |
|------------|--|
| **สาเหตุ** | Server down, Webhook endpoint unreachable |
| **การตรวจจับ** | Google จะ retry แต่หยุดหลังจาก fail หลายครั้ง |
| **การแก้ไข** | แก้ไข server/network แล้ว register ใหม่ |
| **ผลกระทบ** | พลาด notifications ระหว่าง downtime |

---

## Current System Handling

### Auto-Renewal Mechanism
```
Schedule: ทุก 6 ชั่วโมง (0 */6 * * *)
Threshold: Renew เมื่อเหลือไม่ถึง 48 ชั่วโมง

Flow:
1. Query folders ที่ webhook_expiry < NOW() + 48 hours
2. สำหรับแต่ละ folder:
   a. Refresh OAuth token (ถ้าจำเป็น)
   b. Stop webhook เก่า
   c. Register webhook ใหม่
   d. Update webhook_expiry ใน DB
```

### Error Detection Points
| จุดตรวจจับ | Scenarios ที่จับได้ |
|-----------|-------------------|
| Sync Worker | 2, 3, 4, 5, 6 |
| Webhook Renewal | 2, 3 |
| API Calls | 4, 5, 6 |

### Current Gaps (ช่องโหว่)
1. **ไม่มี proactive health check** - รู้ตัวเมื่อ fail เท่านั้น
2. **ไม่แสดง webhook status ใน UI** - User ไม่รู้ว่า webhook ทำงานอยู่หรือไม่
3. **ไม่มี notification เมื่อ renewal fail** - ต้องดู logs

---

## Implemented Features

### 1. Webhook Status in Settings Page (DONE)
แสดงข้อมูลในหน้า `/settings`:
- **active** (สีเขียว): Webhook ทำงานปกติ + วันหมดอายุ
- **expiring** (สีเหลือง): Webhook ใกล้หมดอายุ (< 48 ชม.) + วันหมดอายุ
- **expired** (สีแดง): Webhook หมดอายุแล้ว - รอ auto-renewal หรือกด Sync
- **inactive** (สีเทา): ยังไม่มี webhook - กด Sync เพื่อลงทะเบียน

**API Response:**
```json
{
  "folders": [{
    "id": "uuid",
    "drive_folder_name": "KU TEST",
    "webhook_status": "active",
    "webhook_expiry": "2026-01-15T10:30:00Z",
    ...
  }]
}
```

---

## Recommended Improvements

### 2. Proactive Health Check
- เพิ่ม scheduled job ทดสอบ token validity
- ส่ง test API call ทุกวัน
- แจ้งเตือนถ้า token ใกล้หมดอายุหรือ invalid

### 3. User Notifications
- WebSocket notification เมื่อ webhook fail
- Email notification (optional)
- In-app banner warning

### 4. Database Fields to Track
```sql
-- เพิ่ม fields สำหรับ tracking
ALTER TABLE shared_folders ADD COLUMN webhook_status VARCHAR(20) DEFAULT 'unknown';
-- Values: 'active', 'expired', 'failed', 'unknown'

ALTER TABLE shared_folders ADD COLUMN webhook_last_notification TIMESTAMP;
-- บันทึกเวลาล่าสุดที่ได้รับ notification

ALTER TABLE shared_folders ADD COLUMN webhook_error_message TEXT;
-- บันทึก error message ล่าสุด
```

---

## Testing Checklist

| Scenario | วิธีทดสอบ | Expected Result | Status |
|----------|----------|-----------------|--------|
| 1. Expiration | แก้ไข DB หรือรอ 7 วัน | Auto-renew สำเร็จ | ⬜ |
| 2. User Revoke | ลบ app จาก Google Account | Reconnect button แสดง | ⬜ |
| 3. Token Expired | แก้ไข refresh_token ใน DB | Reconnect button แสดง | ⬜ |
| 4. Folder Deleted | ลบ folder ใน Google Drive | Error message แสดง | ⬜ |
| 5. Permission Revoked | ถอดสิทธิ์จาก shared folder | Error message แสดง | ⬜ |
| 6. Quota Exceeded | Mock API response | Retry หรือแจ้ง error | ⬜ |

---

## API Error Codes Reference

| Error | HTTP Code | สาเหตุ | การจัดการ |
|-------|-----------|--------|----------|
| `invalid_grant` | 400 | Token revoked/expired | Reconnect |
| `Token has been revoked` | 401 | User revoked access | Reconnect |
| `File not found` | 404 | Folder deleted | Remove folder |
| `insufficientPermissions` | 403 | No access to folder | Request access |
| `userRateLimitExceeded` | 429 | Too many requests | Retry with backoff |
| `quotaExceeded` | 429 | Daily quota exceeded | Wait 24 hours |

---

## Files Reference

| File | Purpose |
|------|---------|
| `gofiber/application/serviceimpl/shared_folder_service_impl.go` | Webhook registration, renewal |
| `gofiber/infrastructure/googledrive/drive_client.go` | Google Drive API calls |
| `gofiber/infrastructure/worker/sync_worker.go` | Sync with error detection |
| `gofiber/pkg/di/container.go` | Scheduler setup |
| `vite/src/page/settings/page.tsx` | Settings UI |
