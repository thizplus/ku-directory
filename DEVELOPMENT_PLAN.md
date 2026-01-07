# DangKu - University Photo Directory System
## Development Plan

---

## Project Overview

ระบบ Web Application สำหรับมหาวิทยาลัย เพื่อจัดการและค้นหารูปภาพกิจกรรมจาก Google Drive พร้อมระบบ AI ช่วยเขียนข่าวประชาสัมพันธ์ และค้นหาด้วยใบหน้า

### Tech Stack (Confirmed)
| Layer | Technology |
|-------|------------|
| Backend | GoFiber + PostgreSQL + Redis |
| Frontend | Vite + React 19 + TypeScript + TailwindCSS |
| AI | Google Gemini 2.0 Flash |
| Face Recognition | Python + InsightFace (GPU-accelerated) |
| Auth | Google OAuth + LINE Login |
| Hosting | DigitalOcean App Platform |

### Development Machine Specs
| Component | Spec | Use Case |
|-----------|------|----------|
| CPU | Ryzen 9 5950x (16-core) | Parallel batch processing |
| GPU | RTX 3060 Ti (8GB VRAM) | CUDA face recognition |
| RAM | 64GB | Large batch processing |

---

## Deployment Strategy

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Development Phase (Local)                        │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────────┐│
│  │ PostgreSQL   │   │ Redis        │   │ Face Indexing            ││
│  │ (Docker)     │   │ (Docker)     │   │ (GPU - RTX 3060 Ti)      ││
│  └──────────────┘   └──────────────┘   └──────────────────────────┘│
│         │                  │                      │                 │
│         └──────────────────┼──────────────────────┘                 │
│                            ▼                                        │
│              ┌─────────────────────────┐                           │
│              │ pg_dump (export data)   │                           │
│              └─────────────────────────┘                           │
└─────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                 Production (DigitalOcean App Platform)              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────────┐│
│  │ Managed      │   │ Managed      │   │ GoFiber API              ││
│  │ PostgreSQL   │◄──│ Redis        │◄──│ + Vite Static            ││
│  │ (pg_restore) │   │              │   │                          ││
│  └──────────────┘   └──────────────┘   └──────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

**ข้อดีของแนวคิดนี้:**
- ประหยัดค่า server ช่วง development
- Face indexing ใช้ GPU local = เร็วกว่า cloud CPU มาก
- หลายหมื่นรูป index ที่เครื่องตัวเอง ใช้เวลาไม่นาน
- Production server ไม่ต้องมี GPU (แค่ vector search)

---

## Current Starter Structure

### Backend (GoFiber) - Clean Architecture
```
gofiber/
├── cmd/api/main.go              # Entry point
├── domain/
│   ├── models/                  # Entities (User, File, Job, Task)
│   ├── dto/                     # Data Transfer Objects
│   ├── repositories/            # Repository interfaces
│   └── services/                # Service interfaces
├── application/serviceimpl/     # Service implementations
├── infrastructure/
│   ├── postgres/                # Repository implementations
│   ├── redis/                   # Cache layer
│   ├── storage/                 # Bunny CDN storage
│   └── websocket/               # WebSocket handler
├── interfaces/api/
│   ├── handlers/                # HTTP handlers
│   ├── middleware/              # Auth, CORS, Logger
│   └── routes/                  # Route definitions
└── pkg/
    ├── config/                  # Configuration
    ├── di/                      # Dependency injection
    ├── scheduler/               # Job scheduler (gocron)
    └── utils/                   # JWT, Validator, Response
```

### Frontend (Vite) - Component-based
```
vite/src/
├── components/
│   ├── ui/                      # shadcn/ui components
│   └── page/                    # Page-specific components
├── layouts/                     # App layout, Sidebar, Nav
├── page/                        # Route pages (admin, agent, sales)
├── routes/                      # React Router config
├── hooks/                       # Custom hooks
├── lib/                         # Utilities
└── theme/                       # Theme provider
```

---

## 3 Main Features

### Feature 1: Google Drive Sync System

**Purpose:** Sync Google Drive folders/files to PostgreSQL for fast searching

#### New Models Required
```go
// Folder - represents Google Drive folder
type DriveFolder struct {
    ID            uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    DriveFolderID string    `gorm:"uniqueIndex;not null"` // Google Drive folder ID
    Name          string    `gorm:"not null"`
    ParentID      *uuid.UUID
    Parent        *DriveFolder `gorm:"foreignKey:ParentID"`
    Path          string       // Full path: "กิจกรรม/ปี2567/กีฬาสี"
    ThumbnailURL  string
    PhotoCount    int          // Cached count
    CreatedAt     time.Time
    UpdatedAt     time.Time
    SyncedAt      time.Time
}

// Photo - represents image in Google Drive
type DrivePhoto struct {
    ID            uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    DriveFileID   string    `gorm:"uniqueIndex;not null"`
    FolderID      uuid.UUID `gorm:"not null"`
    Folder        DriveFolder `gorm:"foreignKey:FolderID"`
    FileName      string    `gorm:"not null"`
    ThumbnailURL  string    // Google Drive thumbnail
    FullURL       string    // Direct link or proxy
    Width         int
    Height        int
    FileSize      int64
    TakenAt       *time.Time // EXIF date
    FaceIndexed   bool      `gorm:"default:false"` // Has been processed
    FaceCount     int       `gorm:"default:0"`     // Number of faces found
    CreatedAt     time.Time
    SyncedAt      time.Time
}
```

#### Implementation Steps
1. **Google Drive Integration**
   - สร้าง `infrastructure/gdrive/client.go` สำหรับ Google Drive API
   - ใช้ Service Account (ง่ายกว่า OAuth สำหรับ backend)
   - Methods: ListFolders, ListFiles, GetThumbnail, WatchChanges

2. **Sync Service**
   - สร้าง `domain/services/sync_service.go`
   - Initial full sync (recursive folder scan)
   - Incremental sync via polling หรือ webhook

3. **Background Job Queue**
   - ใช้ Redis + gocron ที่มีอยู่แล้ว
   - Job types: `full_sync`, `incremental_sync`, `process_new_folder`

4. **API Endpoints**
   ```
   POST   /api/v1/sync/init          # Start initial sync
   GET    /api/v1/sync/status        # Get sync status
   POST   /api/v1/webhook/gdrive     # Google Drive webhook (optional)
   GET    /api/v1/folders            # List folders (with pagination)
   GET    /api/v1/folders/:id        # Get folder detail
   GET    /api/v1/folders/:id/photos # Get photos in folder
   GET    /api/v1/search/folders     # Search folders by name
   ```

---

### Feature 2: AI News Writer Assistant (Gemini)

**Purpose:** ช่วยเขียนข่าวประชาสัมพันธ์จากรูปภาพกิจกรรม

#### Why Gemini?
- **Gemini 2.0 Flash** - เร็ว, ราคาถูก, อ่านภาพได้ดี
- **Free tier** - 15 RPM, 1M tokens/day (เพียงพอสำหรับ development)
- **Image generation** - Imagen 3 สร้างภาพได้ (ถ้าต้องการ)

#### Implementation Steps
1. **Gemini Integration**
   ```go
   // infrastructure/ai/gemini_client.go
   type GeminiClient struct {
       apiKey string
       model  string // "gemini-2.0-flash-exp"
   }

   func (c *GeminiClient) GenerateNewsFromImages(
       images []string,  // base64 or URLs
       context string,   // activity name, date
       style string,     // formal, casual
   ) (*NewsResponse, error)
   ```

2. **News Draft Model**
   ```go
   type NewsDraft struct {
       ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
       FolderID    uuid.UUID
       Folder      DriveFolder `gorm:"foreignKey:FolderID"`
       Title       string
       Content     string      `gorm:"type:text"`
       CoverImage  string      // Selected cover photo
       PhotoIDs    pq.StringArray `gorm:"type:text[]"` // Selected photos
       Status      string      `gorm:"default:'draft'"` // draft, published
       AIPrompt    string      // Prompt used
       CreatedBy   uuid.UUID
       User        User `gorm:"foreignKey:CreatedBy"`
       CreatedAt   time.Time
       UpdatedAt   time.Time
   }
   ```

3. **API Endpoints**
   ```
   POST   /api/v1/ai/generate-news   # Generate news from images
   Body: { folder_id, photo_ids[], prompt?, style? }

   GET    /api/v1/news               # List news drafts
   POST   /api/v1/news               # Create draft manually
   GET    /api/v1/news/:id           # Get news detail
   PUT    /api/v1/news/:id           # Edit news draft
   DELETE /api/v1/news/:id           # Delete draft
   POST   /api/v1/news/:id/regenerate # Re-generate with new prompt
   ```

4. **Frontend Components**
   - Photo grid selector (select from folder)
   - Prompt customization form
   - Generated preview with edit capability
   - Markdown/Rich text editor

---

### Feature 3: Face Search System (High Accuracy)

**Purpose:** ค้นหารูปภาพจากใบหน้าคน ความแม่นยำสูง

#### Recommended: InsightFace + ArcFace

เนื่องจากต้องการความแม่นยำสูง และมี GPU (RTX 3060 Ti) แนะนำใช้ **InsightFace** แทน face_recognition เพราะ:

| Library | Accuracy (LFW) | Speed (GPU) | Embedding Size |
|---------|----------------|-------------|----------------|
| face_recognition (dlib) | 99.38% | Slow | 128-d |
| **InsightFace (ArcFace)** | **99.83%** | **Fast** | 512-d |

#### Architecture
```
┌─────────────────────────────────────────────────────────────────────┐
│                  Face Processing Service (Python)                   │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │                    FastAPI Server                              │ │
│  │  POST /detect    - Detect faces, return bounding boxes        │ │
│  │  POST /encode    - Generate 512-d embedding                   │ │
│  │  POST /batch     - Batch process multiple images              │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                              │                                      │
│  ┌───────────────────────────┴───────────────────────────────────┐ │
│  │              InsightFace + ONNX Runtime (CUDA)                │ │
│  │  - buffalo_l model (high accuracy)                            │ │
│  │  - RetinaFace detection                                       │ │
│  │  - ArcFace recognition                                        │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

#### New Models
```go
// FaceEmbedding - stores face vector data
type FaceEmbedding struct {
    ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    PhotoID     uuid.UUID `gorm:"not null;index"`
    Photo       DrivePhoto `gorm:"foreignKey:PhotoID"`
    FaceIndex   int       // Face number in photo (0, 1, 2...)
    BoundingBox string    `gorm:"type:jsonb"` // {x, y, width, height, confidence}
    Embedding   pgvector.Vector `gorm:"type:vector(512)"` // 512-d ArcFace
    CreatedAt   time.Time
}

// FaceSearchLog - optional, for analytics
type FaceSearchLog struct {
    ID          uuid.UUID
    SearcherIP  string
    ResultCount int
    CreatedAt   time.Time
}
```

#### Python Face Service Structure
```
face-service/
├── main.py                 # FastAPI entry point
├── models/
│   └── face_analyzer.py    # InsightFace wrapper
├── services/
│   ├── detector.py         # Face detection
│   ├── encoder.py          # Embedding generation
│   └── batch_processor.py  # Batch processing worker
├── api/
│   └── routes.py           # API endpoints
├── config.py               # Settings
├── requirements.txt
└── Dockerfile              # For production (CPU only)
```

#### Batch Indexing Flow (Local Development)
```
┌─────────────────────────────────────────────────────────────────────┐
│                    Local Batch Indexing                             │
│                                                                     │
│  1. Run: python batch_index.py --start                             │
│                                                                     │
│  2. Flow:                                                          │
│     ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│     │ PostgreSQL  │───▶│ Fetch photos│───▶│ Download from       │  │
│     │ (unindexed) │    │ batch=100   │    │ Google Drive        │  │
│     └─────────────┘    └─────────────┘    └─────────────────────┘  │
│                                                   │                 │
│     ┌─────────────┐    ┌─────────────┐    ┌──────▼──────────────┐  │
│     │ Update      │◀───│ Save        │◀───│ InsightFace         │  │
│     │ face_indexed│    │ embeddings  │    │ (GPU - RTX 3060 Ti) │  │
│     └─────────────┘    └─────────────┘    └─────────────────────┘  │
│                                                                     │
│  3. Progress: WebSocket updates to dashboard                       │
│                                                                     │
│  Estimated speed: ~50-100 images/sec with RTX 3060 Ti             │
│  10,000 photos ≈ 2-3 minutes                                       │
│  50,000 photos ≈ 10-15 minutes                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Vector Search with pgvector
```sql
-- Enable extension
CREATE EXTENSION vector;

-- Add embedding column
ALTER TABLE face_embeddings
ADD COLUMN embedding vector(512);

-- Create index for fast similarity search
CREATE INDEX ON face_embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Search query (find similar faces)
SELECT
    fe.id,
    fe.photo_id,
    dp.file_name,
    dp.thumbnail_url,
    fe.bounding_box,
    1 - (fe.embedding <=> $1) as similarity
FROM face_embeddings fe
JOIN drive_photos dp ON fe.photo_id = dp.id
WHERE 1 - (fe.embedding <=> $1) > 0.6  -- threshold
ORDER BY fe.embedding <=> $1
LIMIT 50;
```

#### API Endpoints
```
# Face Service (Python)
POST   /detect              # Detect faces in image
       Body: { image: base64 }
       Response: { faces: [{bbox, confidence}] }

POST   /encode              # Get embedding for face
       Body: { image: base64, bbox }
       Response: { embedding: float[512] }

# GoFiber API
POST   /api/v1/face/search  # Search by face
       Body: { image: base64, face_index: 0 }
       Response: { results: [{photo, similarity, bbox}] }

GET    /api/v1/face/status  # Indexing status
       Response: { total, indexed, pending, speed }
```

---

## Authentication (Google + LINE)

### OAuth Setup
```go
// infrastructure/oauth/providers.go

type OAuthConfig struct {
    Google GoogleConfig
    LINE   LINEConfig
}

type GoogleConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string // email, profile
}

type LINEConfig struct {
    ChannelID     string
    ChannelSecret string
    RedirectURL   string
    Scopes        []string // profile, openid, email
}
```

### User Model Update
```go
type User struct {
    ID            uuid.UUID `gorm:"primaryKey"`
    Email         string    `gorm:"uniqueIndex"`
    Name          string
    Avatar        string
    Provider      string    // "google", "line", "local"
    ProviderID    string    // OAuth provider's user ID
    Role          string    `gorm:"default:'user'"` // admin, staff, user
    LastLoginAt   *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### API Endpoints
```
GET    /api/v1/auth/google          # Redirect to Google
GET    /api/v1/auth/google/callback # Google callback
GET    /api/v1/auth/line            # Redirect to LINE
GET    /api/v1/auth/line/callback   # LINE callback
POST   /api/v1/auth/logout          # Logout
GET    /api/v1/auth/me              # Get current user
```

---

## Database Schema Summary

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- Users (update existing)
ALTER TABLE users ADD COLUMN provider VARCHAR(20);
ALTER TABLE users ADD COLUMN provider_id VARCHAR(255);
ALTER TABLE users ADD COLUMN avatar VARCHAR(500);
ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user';

-- Drive Folders
CREATE TABLE drive_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drive_folder_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(500) NOT NULL,
    parent_id UUID REFERENCES drive_folders(id),
    path VARCHAR(1000),
    thumbnail_url VARCHAR(500),
    photo_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    synced_at TIMESTAMP
);

CREATE INDEX idx_folders_parent ON drive_folders(parent_id);
CREATE INDEX idx_folders_path ON drive_folders(path);

-- Drive Photos
CREATE TABLE drive_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drive_file_id VARCHAR(255) UNIQUE NOT NULL,
    folder_id UUID NOT NULL REFERENCES drive_folders(id),
    file_name VARCHAR(500) NOT NULL,
    thumbnail_url VARCHAR(500),
    full_url VARCHAR(500),
    width INT,
    height INT,
    file_size BIGINT,
    taken_at TIMESTAMP,
    face_indexed BOOLEAN DEFAULT FALSE,
    face_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    synced_at TIMESTAMP
);

CREATE INDEX idx_photos_folder ON drive_photos(folder_id);
CREATE INDEX idx_photos_face_indexed ON drive_photos(face_indexed);

-- Face Embeddings
CREATE TABLE face_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID NOT NULL REFERENCES drive_photos(id) ON DELETE CASCADE,
    face_index INT DEFAULT 0,
    bounding_box JSONB,
    embedding vector(512),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_face_photo ON face_embeddings(photo_id);
CREATE INDEX idx_face_embedding ON face_embeddings
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- News Drafts
CREATE TABLE news_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id UUID REFERENCES drive_folders(id),
    title VARCHAR(500),
    content TEXT,
    cover_image VARCHAR(500),
    photo_ids TEXT[],
    status VARCHAR(20) DEFAULT 'draft',
    ai_prompt TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Environment Variables

```env
# ===== Existing =====
DATABASE_URL=postgres://user:pass@localhost:5432/dangku
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-jwt-secret
PORT=3000

# ===== Google Cloud =====
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_REDIRECT_URL=http://localhost:3000/api/v1/auth/google/callback

# Google Drive (Service Account)
GOOGLE_SERVICE_ACCOUNT_JSON=path/to/service-account.json
GOOGLE_DRIVE_ROOT_FOLDER_ID=1abc...xyz

# ===== LINE Login =====
LINE_CHANNEL_ID=xxx
LINE_CHANNEL_SECRET=xxx
LINE_REDIRECT_URL=http://localhost:3000/api/v1/auth/line/callback

# ===== Gemini AI =====
GEMINI_API_KEY=xxx
GEMINI_MODEL=gemini-2.0-flash-exp

# ===== Face Service =====
FACE_SERVICE_URL=http://localhost:8001

# ===== Production (DigitalOcean) =====
# DATABASE_URL=${database.DATABASE_URL}
# REDIS_URL=${redis.REDIS_URL}
```

---

## Development Phases

### Phase 1: Foundation & Auth
1. Setup Google Cloud Project
   - Create project
   - Enable Drive API
   - Create Service Account
   - Create OAuth credentials (Google Login)
2. Setup LINE Developers
   - Create LINE Login channel
3. Implement OAuth login (Google + LINE)
4. Update User model & migration
5. Create login page UI

### Phase 2: Google Drive Sync
1. Implement Google Drive client (Service Account)
2. Create DriveFolder & DrivePhoto models
3. Build sync service (recursive scan)
4. Create folder browsing UI
5. Implement folder search

### Phase 3: AI News Writer
1. Setup Gemini API key
2. Implement Gemini client
3. Build news draft CRUD
4. Create image selector component
5. Build news editor with preview

### Phase 4: Face Search
1. Setup Python environment with CUDA
2. Install InsightFace + onnxruntime-gpu
3. Build FastAPI face service
4. Enable pgvector extension
5. Create batch indexing script
6. Run initial indexing (local GPU)
7. Build face search UI
8. Export database for production

### Phase 5: Deployment
1. Setup DigitalOcean App Platform
2. Configure managed PostgreSQL (import data)
3. Configure managed Redis
4. Deploy GoFiber + Vite build
5. Setup domain & SSL
6. Configure webhooks for new photos

---

## File Structure (Final)

```
_dang_ku/
├── gofiber/                          # Backend
│   ├── domain/
│   │   ├── models/
│   │   │   ├── user.go               # Updated with OAuth
│   │   │   ├── drive_folder.go       # NEW
│   │   │   ├── drive_photo.go        # NEW
│   │   │   ├── face_embedding.go     # NEW
│   │   │   └── news_draft.go         # NEW
│   │   └── ...
│   ├── infrastructure/
│   │   ├── gdrive/
│   │   │   └── client.go             # NEW - Google Drive API
│   │   ├── ai/
│   │   │   └── gemini_client.go      # NEW - Gemini API
│   │   ├── oauth/
│   │   │   ├── google.go             # NEW
│   │   │   └── line.go               # NEW
│   │   └── ...
│   └── ...
│
├── vite/                             # Frontend
│   └── src/
│       ├── page/
│       │   ├── auth/
│       │   │   └── login/            # Login page
│       │   ├── folders/              # Folder browser
│       │   ├── photos/               # Photo gallery
│       │   ├── face-search/          # Face search
│       │   └── news/                 # News editor
│       └── ...
│
├── face-service/                     # NEW - Python service
│   ├── main.py
│   ├── batch_index.py                # Batch indexing script
│   ├── requirements.txt
│   └── ...
│
├── docker-compose.yml                # Local dev (PostgreSQL + Redis)
├── DEVELOPMENT_PLAN.md               # This file
└── README.md
```

---

## Quick Start Commands

```bash
# 1. Start local services
docker-compose up -d  # PostgreSQL + Redis

# 2. Run GoFiber backend
cd gofiber
go run cmd/api/main.go

# 3. Run Vite frontend
cd vite
npm run dev

# 4. Setup Python face service
cd face-service
python -m venv venv
source venv/bin/activate  # or venv\Scripts\activate on Windows
pip install -r requirements.txt
python main.py

# 5. Run batch face indexing (after Drive sync)
python batch_index.py --start
```

---

## Cost Estimation (DigitalOcean)

| Service | Spec | Monthly Cost |
|---------|------|--------------|
| App Platform (Basic) | 1 vCPU, 512MB | $5 |
| Managed PostgreSQL | 1GB RAM, 10GB | $15 |
| Managed Redis | 1GB RAM | $15 |
| **Total** | | **~$35/month** |

*Note: Face processing ทำที่เครื่อง local ไม่ต้องจ่ายค่า GPU server*
