import { useState, useEffect, useCallback } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { PhotoProvider, PhotoView } from "react-photo-view"
import "react-photo-view/dist/react-photo-view.css"
import {
  Images,
  Folder,
  FolderOpen,
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Clock,
  ImageOff,
  Users,
  ArrowRight,
  Download,
  Check,
  ZoomIn,
  Newspaper,
  RotateCcw,
  ChevronRight,
  Home,
  LayoutGrid,
  List,
  Settings,
  Search,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import {
  useDriveStatus,
  useDownloadPhotos,
} from "@/features/drive"
import {
  useSharedFolders,
  useFolderPhotos,
  useTriggerFolderSync,
} from "@/features/folders"
import { useFaceStats, useRetryFailedPhotos } from "@/features/face-search"
import { useAuth } from "@/hooks/use-auth"
import { useDownloadProgressStore } from "@/stores/download-progress"
import { useSyncProgressStore } from "@/stores/sync-progress"
import { getThumbnailUrl } from "@/shared/config/constants"
import { cn } from "@/lib/utils"
import type { Photo, SharedFolder } from "@/shared/types"

// View mode type
type ViewMode = "grid" | "list"

// Breadcrumb item
interface BreadcrumbItem {
  id: string
  name: string
  path?: string
}

// Sub-folder info from API
interface SubFolderInfo {
  path: string
  name: string
  photo_count: number
}

// Photo card component - Grid view
function PhotoCardGrid({
  photo,
  token,
  isSelected,
  onToggleSelect
}: {
  photo: Photo
  token: string | null
  isSelected: boolean
  onToggleSelect: () => void
}) {
  const [imageError, setImageError] = useState(false)
  const thumbnailUrl = token ? getThumbnailUrl(photo.drive_file_id, token) : ""
  const fullSizeUrl = token ? getThumbnailUrl(photo.drive_file_id, token, 1600) : ""

  const getStatusIcon = () => {
    switch (photo.face_status) {
      case "completed":
        return <CheckCircle2 className="h-3 w-3 text-green-500" />
      case "processing":
        return <Loader2 className="h-3 w-3 animate-spin text-blue-500" />
      case "pending":
        return <Clock className="h-3 w-3 text-yellow-500" />
      case "failed":
        return <AlertCircle className="h-3 w-3 text-red-500" />
      default:
        return null
    }
  }

  return (
    <div className="group relative">
      <PhotoView src={fullSizeUrl}>
        <div className={cn(
          "relative aspect-square bg-muted rounded-lg overflow-hidden cursor-pointer transition-all",
          isSelected && "ring-2 ring-primary"
        )}>
          {imageError || !token ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <ImageOff className="h-6 w-6 text-muted-foreground/50" />
            </div>
          ) : (
            <img
              src={thumbnailUrl}
              alt={photo.file_name}
              className="w-full h-full object-cover transition-transform group-hover:scale-105"
              onError={() => setImageError(true)}
              loading="lazy"
            />
          )}
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center pointer-events-none">
            <ZoomIn className="h-8 w-8 text-white opacity-0 group-hover:opacity-70 transition-opacity" />
          </div>
          <div className="absolute top-1.5 left-1.5">{getStatusIcon()}</div>
          {photo.face_count > 0 && (
            <div className="absolute bottom-1.5 right-1.5 bg-black/60 text-white text-xs px-1.5 py-0.5 rounded flex items-center gap-1">
              <Users className="h-3 w-3" />
              {photo.face_count}
            </div>
          )}
        </div>
      </PhotoView>

      <button
        className="absolute top-1.5 right-1.5 z-10"
        onClick={(e) => {
          e.stopPropagation()
          onToggleSelect()
        }}
      >
        <div className={cn(
          "h-5 w-5 rounded border flex items-center justify-center transition-all shadow-sm",
          isSelected
            ? "bg-primary border-primary text-white"
            : "bg-white/90 border-gray-300 hover:border-primary hover:bg-white"
        )}>
          {isSelected && <Check className="h-3.5 w-3.5" strokeWidth={3} />}
        </div>
      </button>

      <p className="mt-1.5 text-xs truncate text-muted-foreground" title={photo.file_name}>
        {photo.file_name}
      </p>
    </div>
  )
}

// Photo card component - List view
function PhotoCardList({
  photo,
  token,
  isSelected,
  onToggleSelect
}: {
  photo: Photo
  token: string | null
  isSelected: boolean
  onToggleSelect: () => void
}) {
  const [imageError, setImageError] = useState(false)
  const thumbnailUrl = token ? getThumbnailUrl(photo.drive_file_id, token, 100) : ""
  const fullSizeUrl = token ? getThumbnailUrl(photo.drive_file_id, token, 1600) : ""

  const getStatusBadge = () => {
    switch (photo.face_status) {
      case "completed":
        return <span className="text-xs text-green-600 flex items-center gap-1"><CheckCircle2 className="h-3 w-3" /> เสร็จแล้ว</span>
      case "processing":
        return <span className="text-xs text-blue-600 flex items-center gap-1"><Loader2 className="h-3 w-3 animate-spin" /> กำลังประมวลผล</span>
      case "pending":
        return <span className="text-xs text-yellow-600 flex items-center gap-1"><Clock className="h-3 w-3" /> รอประมวลผล</span>
      case "failed":
        return <span className="text-xs text-red-600 flex items-center gap-1"><AlertCircle className="h-3 w-3" /> ล้มเหลว</span>
      default:
        return null
    }
  }

  return (
    <div className={cn(
      "flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors group",
      isSelected && "bg-primary/5 ring-1 ring-primary/20"
    )}>
      <button
        className="flex-shrink-0"
        onClick={(e) => {
          e.stopPropagation()
          onToggleSelect()
        }}
      >
        <div className={cn(
          "h-5 w-5 rounded border flex items-center justify-center transition-all",
          isSelected
            ? "bg-primary border-primary text-white"
            : "border-gray-300 hover:border-primary"
        )}>
          {isSelected && <Check className="h-3.5 w-3.5" strokeWidth={3} />}
        </div>
      </button>

      <PhotoView src={fullSizeUrl}>
        <div className="w-12 h-12 rounded overflow-hidden bg-muted flex-shrink-0 cursor-pointer">
          {imageError || !token ? (
            <div className="w-full h-full flex items-center justify-center">
              <ImageOff className="h-4 w-4 text-muted-foreground/50" />
            </div>
          ) : (
            <img
              src={thumbnailUrl}
              alt={photo.file_name}
              className="w-full h-full object-cover"
              onError={() => setImageError(true)}
              loading="lazy"
            />
          )}
        </div>
      </PhotoView>

      <div className="flex-1 min-w-0">
        <p className="text-sm truncate">{photo.file_name}</p>
        <div className="flex items-center gap-3 mt-0.5">
          {getStatusBadge()}
          {photo.face_count > 0 && (
            <span className="text-xs text-muted-foreground flex items-center gap-1">
              <Users className="h-3 w-3" /> {photo.face_count} ใบหน้า
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

// Folder card component - Grid view (Google Drive style)
function FolderCardGrid({
  name,
  photoCount,
  onClick,
}: {
  name: string
  photoCount: number
  onClick: () => void
}) {
  return (
    <button
      className="group text-left w-full"
      onClick={onClick}
    >
      <div className="aspect-[4/3] bg-gradient-to-br from-muted/80 to-muted rounded-xl border border-border/50 flex flex-col items-center justify-center gap-2 transition-all group-hover:border-primary/30 group-hover:shadow-md group-hover:scale-[1.02]">
        <Folder className="h-12 w-12 md:h-16 md:w-16 text-yellow-500 drop-shadow-sm" />
        <div className="text-center px-2">
          <p className="text-sm font-medium truncate max-w-full" title={name}>
            {name}
          </p>
          <p className="text-xs text-muted-foreground">{photoCount} รูป</p>
        </div>
      </div>
    </button>
  )
}

// Folder card component - List view
function FolderCardList({
  name,
  photoCount,
  onClick,
}: {
  name: string
  photoCount: number
  onClick: () => void
}) {
  return (
    <button
      className="w-full flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors text-left"
      onClick={onClick}
    >
      <Folder className="h-10 w-10 text-yellow-500 flex-shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate font-medium">{name}</p>
        <p className="text-xs text-muted-foreground">{photoCount} รูป</p>
      </div>
      <ChevronRight className="h-4 w-4 text-muted-foreground" />
    </button>
  )
}

export default function GalleryPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { token } = useAuth()

  // State
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    return (localStorage.getItem("gallery-view-mode") as ViewMode) || "grid"
  })
  const [selectedFolderId, setSelectedFolderId] = useState<string | undefined>()
  const [currentPath, setCurrentPath] = useState<string | undefined>()
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([])
  const [page, setPage] = useState(1)
  const [selectedPhotos, setSelectedPhotos] = useState<Set<string>>(new Set())
  const limit = viewMode === "grid" ? 24 : 50

  // Stores
  const downloadProgress = useDownloadProgressStore((state) => state.progress)
  const syncProgressMap = useSyncProgressStore((state) => state.progress)

  // Queries
  const { data: driveStatus, isLoading: statusLoading } = useDriveStatus()
  const { data: sharedFoldersData, isLoading: foldersLoading } = useSharedFolders(driveStatus?.connected)
  const sharedFolders = sharedFoldersData?.folders || []

  const { data: photosData, isLoading: photosLoading } = useFolderPhotos(
    selectedFolderId || "",
    page,
    limit,
    undefined,
    !!selectedFolderId,
    currentPath
  )
  const { data: stats, isLoading: statsLoading } = useFaceStats()

  // Mutations
  const triggerSyncMutation = useTriggerFolderSync()
  const retryFailedMutation = useRetryFailedPhotos()
  const downloadMutation = useDownloadPhotos()

  // Save view mode preference
  useEffect(() => {
    localStorage.setItem("gallery-view-mode", viewMode)
  }, [viewMode])

  // Read folder from URL on mount
  useEffect(() => {
    const folderParam = searchParams.get("folder")
    const pathParam = searchParams.get("path")
    if (folderParam && !selectedFolderId) {
      setSelectedFolderId(folderParam)
      setCurrentPath(pathParam || undefined)
    }
  }, [])

  // Don't auto-select - let user navigate like Google Drive

  // Build breadcrumbs
  useEffect(() => {
    if (!selectedFolderId) {
      setBreadcrumbs([])
      return
    }

    const folder = sharedFolders.find(f => f.id === selectedFolderId)
    if (!folder) return

    const crumbs: BreadcrumbItem[] = [
      { id: folder.id, name: folder.drive_folder_name, path: undefined }
    ]

    if (currentPath) {
      const pathParts = currentPath.split("/").filter(Boolean)
      // Skip the first part (root folder name)
      let buildPath = ""
      for (let i = 1; i < pathParts.length; i++) {
        buildPath = buildPath ? `${buildPath}/${pathParts[i]}` : pathParts[i]
        crumbs.push({
          id: folder.id,
          name: pathParts[i],
          path: `${folder.drive_folder_name}/${buildPath}`
        })
      }
    }

    setBreadcrumbs(crumbs)
  }, [selectedFolderId, currentPath, sharedFolders])

  // Get current folder's subfolders
  const getCurrentSubfolders = useCallback((): SubFolderInfo[] => {
    if (!selectedFolderId) return []

    const folder = sharedFolders.find(f => f.id === selectedFolderId)
    if (!folder?.children) return []

    // Filter subfolders at current level
    const currentPrefix = currentPath ? `${currentPath}/` : `${folder.drive_folder_name}/`

    return folder.children.filter(child => {
      // Must start with current path
      if (!child.path.startsWith(currentPrefix.slice(0, -1))) return false

      // Get the remaining path after current prefix
      const remaining = child.path.slice(currentPrefix.length)

      // Only direct children (no more slashes)
      return remaining && !remaining.includes("/")
    }).map(child => ({
      ...child,
      name: child.path.split("/").pop() || child.name
    }))
  }, [selectedFolderId, currentPath, sharedFolders])

  const subfolders = getCurrentSubfolders()

  // Navigation handlers
  const handleFolderSelect = (folderId: string, folderName: string) => {
    setSelectedFolderId(folderId)
    setCurrentPath(undefined)
    setPage(1)
    setSelectedPhotos(new Set())
    setSearchParams({ folder: folderId }, { replace: true })
  }

  const handleSubfolderClick = (subfolder: SubFolderInfo) => {
    setCurrentPath(subfolder.path)
    setPage(1)
    setSelectedPhotos(new Set())
    setSearchParams({
      folder: selectedFolderId!,
      path: subfolder.path
    }, { replace: true })
  }

  const handleBreadcrumbClick = (crumb: BreadcrumbItem) => {
    setCurrentPath(crumb.path)
    setPage(1)
    setSelectedPhotos(new Set())
    if (crumb.path) {
      setSearchParams({ folder: crumb.id, path: crumb.path }, { replace: true })
    } else {
      setSearchParams({ folder: crumb.id }, { replace: true })
    }
  }

  const handleHomeClick = () => {
    setSelectedFolderId(undefined)
    setCurrentPath(undefined)
    setBreadcrumbs([])
    setPage(1)
    setSelectedPhotos(new Set())
    setSearchParams({}, { replace: true })
  }

  // Selection handlers
  const togglePhotoSelection = useCallback((driveFileId: string) => {
    setSelectedPhotos(prev => {
      const newSet = new Set(prev)
      if (newSet.has(driveFileId)) {
        newSet.delete(driveFileId)
      } else {
        newSet.add(driveFileId)
      }
      return newSet
    })
  }, [])

  const toggleSelectAll = useCallback(() => {
    if (!photosData) return
    const allIds = photosData.photos.map(p => p.drive_file_id)
    const allSelected = allIds.every(id => selectedPhotos.has(id))
    if (allSelected) {
      setSelectedPhotos(new Set())
    } else {
      setSelectedPhotos(new Set(allIds))
    }
  }, [photosData, selectedPhotos])

  const handleDownload = useCallback(() => {
    if (selectedPhotos.size === 0) return
    downloadMutation.mutate(Array.from(selectedPhotos))
  }, [selectedPhotos, downloadMutation])

  // Reset selection on page change
  useEffect(() => {
    setSelectedPhotos(new Set())
  }, [page])

  // Computed values
  const selectedFolder = sharedFolders.find(f => f.id === selectedFolderId)
  const syncProgress = selectedFolderId ? syncProgressMap[selectedFolderId] : undefined
  const isSyncing = selectedFolder?.sync_status === 'syncing' || !!syncProgress
  const totalPages = photosData ? Math.ceil(photosData.total / limit) : 0
  const photos = photosData?.photos || []
  const allSelected = photos.length > 0 && photos.every(p => selectedPhotos.has(p.drive_file_id))
  const someSelected = selectedPhotos.size > 0

  // Not connected or no folders
  if (!statusLoading && !foldersLoading && (!driveStatus?.connected || sharedFolders.length === 0)) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">คลังรูปภาพ</h1>
          <p className="text-sm text-muted-foreground">จัดการรูปภาพกิจกรรมทั้งหมด</p>
        </div>

        <div className="py-12 text-center">
          <Images className="mx-auto h-10 w-10 text-muted-foreground/50" />
          <p className="mt-4 text-sm text-muted-foreground">
            {!driveStatus?.connected
              ? "ยังไม่ได้เชื่อมต่อ Google Drive"
              : "ยังไม่มีโฟลเดอร์ กรุณาเพิ่มโฟลเดอร์ที่ต้องการ Sync"}
          </p>
          <Button className="mt-4" size="sm" onClick={() => navigate("/settings")}>
            ไปที่ตั้งค่า
            <ArrowRight className="h-3 w-3 ml-1" />
          </Button>
        </div>
      </div>
    )
  }

  // Home view - show all shared folders (Google Drive style)
  if (!selectedFolderId) {
    return (
      <div className="space-y-6">
        {/* Header - Google Drive style */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Home className="h-6 w-6 text-muted-foreground" />
            <div>
              <h1 className="text-xl font-semibold">โฟลเดอร์ทั้งหมด</h1>
              <p className="text-xs text-muted-foreground">
                {sharedFolders.length} โฟลเดอร์ • {stats?.total_photos || 0} รูปภาพ • {stats?.total_faces || 0} ใบหน้า
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {/* View toggle */}
            <div className="flex items-center border rounded-lg overflow-hidden">
              <button
                onClick={() => setViewMode("grid")}
                className={cn(
                  "p-2 transition-colors",
                  viewMode === "grid" ? "bg-primary text-primary-foreground" : "hover:bg-muted"
                )}
              >
                <LayoutGrid className="h-4 w-4" />
              </button>
              <button
                onClick={() => setViewMode("list")}
                className={cn(
                  "p-2 transition-colors",
                  viewMode === "list" ? "bg-primary text-primary-foreground" : "hover:bg-muted"
                )}
              >
                <List className="h-4 w-4" />
              </button>
            </div>
            <Button variant="outline" size="sm" onClick={() => navigate("/settings")}>
              <Settings className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <Separator />

        {/* Folders - Google Drive style grid */}
        {foldersLoading ? (
          <div className={cn(
            "gap-4",
            viewMode === "grid"
              ? "grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6"
              : "space-y-2"
          )}>
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className={cn(
                "bg-muted rounded-xl animate-pulse",
                viewMode === "grid" ? "aspect-[4/3]" : "h-14"
              )} />
            ))}
          </div>
        ) : sharedFolders.length === 0 ? (
          <div className="py-16 text-center">
            <Folder className="mx-auto h-16 w-16 text-muted-foreground/30" />
            <p className="mt-4 text-muted-foreground">ยังไม่มีโฟลเดอร์</p>
            <Button className="mt-4" onClick={() => navigate("/settings")}>
              เพิ่มโฟลเดอร์
            </Button>
          </div>
        ) : viewMode === "grid" ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
            {sharedFolders.map(folder => (
              <FolderCardGrid
                key={folder.id}
                name={folder.drive_folder_name}
                photoCount={folder.photo_count}
                onClick={() => handleFolderSelect(folder.id, folder.drive_folder_name)}
              />
            ))}
          </div>
        ) : (
          <div className="space-y-1">
            {sharedFolders.map(folder => (
              <FolderCardList
                key={folder.id}
                name={folder.drive_folder_name}
                photoCount={folder.photo_count}
                onClick={() => handleFolderSelect(folder.id, folder.drive_folder_name)}
              />
            ))}
          </div>
        )}

        {/* Quick Stats - moved to bottom */}
        <Separator />
        <div className="flex items-center justify-center gap-8 text-sm text-muted-foreground py-2">
          <span className="flex items-center gap-1.5">
            <Images className="h-4 w-4" />
            {stats?.total_photos || 0} รูป
          </span>
          <span className="flex items-center gap-1.5">
            <CheckCircle2 className="h-4 w-4 text-green-500" />
            {stats?.processed_photos || 0} ประมวลผลแล้ว
          </span>
          <span className="flex items-center gap-1.5">
            <Clock className="h-4 w-4 text-yellow-500" />
            {stats?.pending_photos || 0} รอดำเนินการ
          </span>
          <span className="flex items-center gap-1.5">
            <Users className="h-4 w-4 text-primary" />
            {stats?.total_faces || 0} ใบหน้า
          </span>
        </div>
      </div>
    )
  }

  // Folder view - show contents
  return (
    <div className="space-y-4">
      {/* Breadcrumb & Actions */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-1 text-sm min-w-0 overflow-hidden">
          <button
            onClick={handleHomeClick}
            className="flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
          >
            <Home className="h-4 w-4" />
          </button>
          {breadcrumbs.map((crumb, idx) => (
            <div key={`${crumb.id}-${crumb.path || 'root'}`} className="flex items-center gap-1 min-w-0">
              <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
              {idx === breadcrumbs.length - 1 ? (
                <span className="font-medium truncate">{crumb.name}</span>
              ) : (
                <button
                  onClick={() => handleBreadcrumbClick(crumb)}
                  className="text-muted-foreground hover:text-foreground transition-colors truncate"
                >
                  {crumb.name}
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {/* Selected actions */}
          {someSelected && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  const selectedPhotoData = photos.filter(p => selectedPhotos.has(p.drive_file_id))
                  sessionStorage.setItem('newsWriterPhotos', JSON.stringify(selectedPhotoData))
                  navigate('/news-writer')
                }}
              >
                <Newspaper className="h-4 w-4 mr-1" />
                เขียนข่าว ({selectedPhotos.size})
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={handleDownload}
                disabled={downloadMutation.isPending}
              >
                {downloadMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    {downloadProgress ? `${downloadProgress.current}/${downloadProgress.total}` : '...'}
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-1" />
                    ดาวน์โหลด ({selectedPhotos.size})
                  </>
                )}
              </Button>
            </>
          )}

          {/* View toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setViewMode(viewMode === "grid" ? "list" : "grid")}
          >
            {viewMode === "grid" ? <List className="h-4 w-4" /> : <LayoutGrid className="h-4 w-4" />}
          </Button>

          {/* Sync button */}
          <Button
            size="sm"
            onClick={() => triggerSyncMutation.mutate(selectedFolderId)}
            disabled={isSyncing || triggerSyncMutation.isPending}
          >
            {isSyncing ? (
              <>
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                {syncProgress ? `${syncProgress.percent}%` : 'Sync...'}
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4 mr-1" />
                Sync
              </>
            )}
          </Button>

          {/* More actions */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <Settings className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => navigate("/face-search")}>
                <Search className="h-4 w-4 mr-2" />
                ค้นหาใบหน้า
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => navigate("/settings")}>
                <FolderOpen className="h-4 w-4 mr-2" />
                จัดการโฟลเดอร์
              </DropdownMenuItem>
              {(stats?.failed_photos || 0) > 0 && (
                <DropdownMenuItem
                  onClick={() => retryFailedMutation.mutate(undefined)}
                  disabled={retryFailedMutation.isPending}
                  className="text-destructive"
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Retry ล้มเหลว ({stats?.failed_photos})
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <Separator />

      {/* Content */}
      {photosLoading ? (
        <div className={cn(
          "gap-3",
          viewMode === "grid"
            ? "grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6"
            : "space-y-1"
        )}>
          {Array.from({ length: 18 }).map((_, i) => (
            <div key={i} className={cn(
              "bg-muted rounded-lg animate-pulse",
              viewMode === "grid" ? "aspect-square" : "h-16"
            )} />
          ))}
        </div>
      ) : (
        <div className="space-y-4">
          {/* Select all / info */}
          {photos.length > 0 && (
            <div className="flex items-center justify-between text-sm">
              <button
                onClick={toggleSelectAll}
                className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <div className={cn(
                  "h-4 w-4 rounded border flex items-center justify-center",
                  allSelected ? "bg-primary border-primary text-white" : "border-gray-300"
                )}>
                  {allSelected && <Check className="h-3 w-3" strokeWidth={3} />}
                </div>
                เลือกทั้งหมด
              </button>
              <span className="text-muted-foreground">
                {photosData?.total || 0} รูป {subfolders.length > 0 && `| ${subfolders.length} โฟลเดอร์`}
                {totalPages > 1 && ` | หน้า ${page}/${totalPages}`}
              </span>
            </div>
          )}

          {/* Subfolders */}
          {subfolders.length > 0 && (
            <div className={cn(
              viewMode === "grid"
                ? "grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3"
                : "space-y-1"
            )}>
              {subfolders.map(subfolder => (
                viewMode === "grid" ? (
                  <FolderCardGrid
                    key={subfolder.path}
                    name={subfolder.name}
                    photoCount={subfolder.photo_count}
                    onClick={() => handleSubfolderClick(subfolder)}
                  />
                ) : (
                  <FolderCardList
                    key={subfolder.path}
                    name={subfolder.name}
                    photoCount={subfolder.photo_count}
                    onClick={() => handleSubfolderClick(subfolder)}
                  />
                )
              ))}
            </div>
          )}

          {/* Photos */}
          {photos.length > 0 ? (
            <PhotoProvider
              maskOpacity={0.9}
              toolbarRender={({ onScale, scale }) => (
                <div className="flex items-center gap-2">
                  <button className="p-2 text-white/80 hover:text-white" onClick={() => onScale(scale + 0.5)}>
                    <ZoomIn className="h-5 w-5" />
                  </button>
                </div>
              )}
            >
              <div className={cn(
                viewMode === "grid"
                  ? "grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3"
                  : "space-y-1"
              )}>
                {photos.map(photo => (
                  viewMode === "grid" ? (
                    <PhotoCardGrid
                      key={photo.id}
                      photo={photo}
                      token={token}
                      isSelected={selectedPhotos.has(photo.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(photo.drive_file_id)}
                    />
                  ) : (
                    <PhotoCardList
                      key={photo.id}
                      photo={photo}
                      token={token}
                      isSelected={selectedPhotos.has(photo.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(photo.drive_file_id)}
                    />
                  )
                ))}
              </div>
            </PhotoProvider>
          ) : subfolders.length === 0 && (
            <div className="py-12 text-center">
              <Images className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                ไม่พบรูปภาพในโฟลเดอร์นี้
              </p>
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-4">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(page - 1)}
                disabled={page === 1}
              >
                ก่อนหน้า
              </Button>
              <span className="text-sm text-muted-foreground px-4">
                หน้า {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(page + 1)}
                disabled={page >= totalPages}
              >
                ถัดไป
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
