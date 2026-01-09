import { useState, useEffect, useCallback, useMemo } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { PhotoProvider, PhotoView } from "react-photo-view"
import "react-photo-view/dist/react-photo-view.css"
import {
  Images,
  Folder,
  FolderTree,
  RefreshCw,
  Loader2,
  AlertCircle,
  AlertTriangle,
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
  MoreHorizontal,
  Link2,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
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
  useReconnectFolder,
} from "@/features/folders"
import { useFaceStats, useRetryFailedPhotos } from "@/features/face-search"
import { useAuth } from "@/hooks/use-auth"
import { useSyncProgressStore } from "@/stores/sync-progress"
import { getThumbnailUrl } from "@/shared/config/constants"
import { cn } from "@/lib/utils"
import type { Photo } from "@/shared/types"

// View mode type
type ViewMode = "grid" | "list"

// Metric Item Component (same as Dashboard)
interface MetricItemProps {
  label: string
  value: string | number
  icon?: React.ReactNode
  isLoading?: boolean
}

function MetricItem({ label, value, icon, isLoading }: MetricItemProps) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground flex items-center gap-1">
        {icon}
        {label}
      </p>
      {isLoading ? (
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      ) : (
        <p className="text-lg font-semibold tabular-nums">{value}</p>
      )}
    </div>
  )
}

// Breadcrumb item
interface BreadcrumbItem {
  id: string
  name: string
  path?: string
}

// Computed subfolder for display
interface ComputedSubfolder {
  name: string
  fullPath: string
  photoCount: number
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
          isSelected && "ring-2 ring-green-500"
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
          "h-5 w-5 rounded border-2 flex items-center justify-center transition-all shadow-sm",
          isSelected
            ? "bg-green-500 border-green-500 text-white"
            : "bg-white border-gray-400 hover:border-green-500"
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

  return (
    <div className={cn(
      "flex items-center gap-3 px-2 py-1.5 rounded-md hover:bg-muted/50 transition-colors",
      isSelected && "bg-muted"
    )}>
      <button
        className="flex-shrink-0"
        onClick={(e) => {
          e.stopPropagation()
          onToggleSelect()
        }}
      >
        <div className={cn(
          "h-4 w-4 rounded border-2 flex items-center justify-center transition-all",
          isSelected
            ? "bg-green-500 border-green-500 text-white"
            : "border-gray-400 hover:border-green-500"
        )}>
          {isSelected && <Check className="h-3 w-3" strokeWidth={3} />}
        </div>
      </button>

      <PhotoView src={fullSizeUrl}>
        <div className="w-8 h-8 rounded overflow-hidden bg-muted flex-shrink-0 cursor-pointer">
          {imageError || !token ? (
            <div className="w-full h-full flex items-center justify-center">
              <ImageOff className="h-3 w-3 text-muted-foreground/50" />
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

      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex-1 text-sm truncate">{photo.file_name}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{photo.file_name}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <span className="text-xs text-muted-foreground flex-shrink-0">
        {photo.face_count > 0 && `${photo.face_count} ใบหน้า`}
      </span>
    </div>
  )
}

// Folder item - Grid view (shadcn style, minimal)
function FolderItemGrid({
  name,
  photoCount,
  onClick,
  onSync,
  onForceSync,
  onReconnect,
  isSyncing,
  hasError,
}: {
  name: string
  photoCount: number
  onClick: () => void
  onSync?: () => void
  onForceSync?: () => void
  onReconnect?: () => void
  isSyncing?: boolean
  hasError?: boolean
}) {
  return (
    <div className="group relative">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "w-full flex flex-col items-center gap-2 p-4 rounded-lg border border-transparent hover:bg-muted/50 hover:border-border transition-colors",
                hasError && "border-destructive/50"
              )}
              onClick={onClick}
            >
              <Folder className={cn("h-12 w-12 text-muted-foreground", hasError && "text-destructive")} />
              <div className="text-center w-full">
                <p className="text-sm truncate">{name}</p>
                <p className="text-xs text-muted-foreground">{photoCount} รูป</p>
              </div>
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p>{name}</p>
            {hasError && <p className="text-destructive">Token หมดอายุ - กรุณา Reconnect</p>}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      {(onSync || onForceSync || onReconnect) && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="absolute top-2 right-2 p-1 rounded-md opacity-0 group-hover:opacity-100 hover:bg-muted transition-all">
              <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {onSync && (
              <DropdownMenuItem onClick={onSync} disabled={isSyncing}>
                <RefreshCw className={cn("h-4 w-4 mr-2", isSyncing && "animate-spin")} />
                Sync
              </DropdownMenuItem>
            )}
            {onForceSync && (
              <DropdownMenuItem onClick={onForceSync} disabled={isSyncing}>
                <RotateCcw className="h-4 w-4 mr-2" />
                Force Full Sync
              </DropdownMenuItem>
            )}
            {onReconnect && (
              <DropdownMenuItem onClick={onReconnect}>
                <Link2 className="h-4 w-4 mr-2" />
                Reconnect Google
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}

// Folder item - List view (shadcn style, minimal)
function FolderItemList({
  name,
  photoCount,
  onClick,
  onSync,
  onForceSync,
  onReconnect,
  isSyncing,
  hasError,
}: {
  name: string
  photoCount: number
  onClick: () => void
  onSync?: () => void
  onForceSync?: () => void
  onReconnect?: () => void
  isSyncing?: boolean
  hasError?: boolean
}) {
  return (
    <div className="group flex items-center gap-1">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "flex-1 flex items-center gap-3 px-2 py-1.5 rounded-md hover:bg-muted/50 transition-colors text-left",
                hasError && "bg-destructive/10"
              )}
              onClick={onClick}
            >
              <Folder className={cn("h-5 w-5 text-muted-foreground flex-shrink-0", hasError && "text-destructive")} />
              <span className="flex-1 text-sm truncate">{name}</span>
              <span className="text-xs text-muted-foreground flex-shrink-0">{photoCount} รูป</span>
            </button>
          </TooltipTrigger>
          <TooltipContent>
            <p>{name}</p>
            {hasError && <p className="text-destructive">Token หมดอายุ - กรุณา Reconnect</p>}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      {(onSync || onForceSync || onReconnect) && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="p-1 rounded-md opacity-0 group-hover:opacity-100 hover:bg-muted transition-all">
              <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {onSync && (
              <DropdownMenuItem onClick={onSync} disabled={isSyncing}>
                <RefreshCw className={cn("h-4 w-4 mr-2", isSyncing && "animate-spin")} />
                Sync
              </DropdownMenuItem>
            )}
            {onForceSync && (
              <DropdownMenuItem onClick={onForceSync} disabled={isSyncing}>
                <RotateCcw className="h-4 w-4 mr-2" />
                Force Full Sync
              </DropdownMenuItem>
            )}
            {onReconnect && (
              <DropdownMenuItem onClick={onReconnect}>
                <Link2 className="h-4 w-4 mr-2" />
                Reconnect Google
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}

export default function GalleryPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { token } = useAuth()

  // State
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    return (localStorage.getItem("gallery-view-mode") as ViewMode) || "list"
  })
  const [selectedFolderId, setSelectedFolderId] = useState<string | undefined>()
  const [currentPath, setCurrentPath] = useState<string | undefined>()
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([])
  const [page, setPage] = useState(1)
  const [selectedPhotos, setSelectedPhotos] = useState<Set<string>>(new Set())
  const limit = viewMode === "grid" ? 24 : 50

  // Stores
  const syncProgressMap = useSyncProgressStore((state) => state.progress)

  // Queries
  const { data: driveStatus, isLoading: statusLoading } = useDriveStatus()
  const { data: sharedFoldersData, isLoading: foldersLoading } = useSharedFolders(driveStatus?.connected)
  const sharedFolders = sharedFoldersData?.folders || []

  // Compute subfolders first to determine if we should fetch photos
  // We need to compute this before useFolderPhotos to decide whether to enable it
  const computedSubfolders = useMemo((): ComputedSubfolder[] => {
    if (!selectedFolderId) return []

    const folder = sharedFolders.find(f => f.id === selectedFolderId)
    if (!folder?.children || folder.children.length === 0) return []

    const rootName = folder.drive_folder_name
    const currentPrefix = currentPath || rootName

    // Group by first segment after current prefix
    const folderMap = new Map<string, ComputedSubfolder>()

    folder.children.forEach(child => {
      if (!child.path.startsWith(currentPrefix)) return

      const remaining = child.path.slice(currentPrefix.length)
      if (!remaining || remaining === "/") return

      const withoutLeadingSlash = remaining.startsWith("/") ? remaining.slice(1) : remaining
      const firstSegment = withoutLeadingSlash.split("/")[0]

      if (!firstSegment) return

      const fullPath = `${currentPrefix}/${firstSegment}`

      if (folderMap.has(firstSegment)) {
        const existing = folderMap.get(firstSegment)!
        existing.photoCount += child.photo_count
      } else {
        folderMap.set(firstSegment, {
          name: firstSegment,
          fullPath,
          photoCount: child.photo_count
        })
      }
    })

    return Array.from(folderMap.values()).sort((a, b) => a.name.localeCompare(b.name))
  }, [selectedFolderId, currentPath, sharedFolders])

  // Compute the folder path to filter photos by
  // At root level, use drive_folder_name (e.g., "KU TEST")
  // In subfolders, use currentPath (e.g., "KU TEST/2566")
  const folderPathFilter = useMemo(() => {
    if (!selectedFolderId) return undefined
    if (currentPath) return currentPath
    // At root level, use the folder's drive_folder_name to only show photos at root
    const folder = sharedFolders.find(f => f.id === selectedFolderId)
    return folder?.drive_folder_name
  }, [selectedFolderId, currentPath, sharedFolders])

  // Only fetch photos when:
  // 1. We have a selected folder AND
  // 2. Either we have a currentPath (inside a subfolder) OR there are no subfolders at this level
  const shouldFetchPhotos = !!selectedFolderId && (!!currentPath || computedSubfolders.length === 0)

  const { data: photosData, isLoading: photosLoading } = useFolderPhotos(
    selectedFolderId || "",
    page,
    limit,
    undefined,
    shouldFetchPhotos,
    folderPathFilter
  )
  const { data: stats, isLoading: statsLoading } = useFaceStats()

  // Mutations
  const triggerSyncMutation = useTriggerFolderSync()
  const retryFailedMutation = useRetryFailedPhotos()
  const downloadMutation = useDownloadPhotos()
  const reconnectMutation = useReconnectFolder()

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
      let buildPath = folder.drive_folder_name
      for (let i = 1; i < pathParts.length; i++) {
        buildPath = `${buildPath}/${pathParts[i]}`
        crumbs.push({
          id: folder.id,
          name: pathParts[i],
          path: buildPath
        })
      }
    }

    setBreadcrumbs(crumbs)
  }, [selectedFolderId, currentPath, sharedFolders])

  // Use computed subfolders (calculated earlier for shouldFetchPhotos)
  const subfolders = computedSubfolders

  // Navigation handlers
  const handleFolderSelect = (folderId: string) => {
    setSelectedFolderId(folderId)
    setCurrentPath(undefined)
    setPage(1)
    setSelectedPhotos(new Set())
    setSearchParams({ folder: folderId }, { replace: true })
  }

  const handleSubfolderClick = (subfolder: ComputedSubfolder) => {
    setCurrentPath(subfolder.fullPath)
    setPage(1)
    setSelectedPhotos(new Set())
    setSearchParams({
      folder: selectedFolderId!,
      path: subfolder.fullPath
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
          <h1 className="text-xl font-medium">คลังรูปภาพ</h1>
        </div>

        <div className="py-16 text-center">
          <Images className="mx-auto h-12 w-12 text-muted-foreground/30" />
          <p className="mt-4 text-sm text-muted-foreground">
            {!driveStatus?.connected
              ? "ยังไม่ได้เชื่อมต่อ Google Drive"
              : "ยังไม่มีโฟลเดอร์"}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => navigate("/settings")}>
            ไปที่ตั้งค่า
            <ArrowRight className="h-4 w-4 ml-2" />
          </Button>
        </div>
      </div>
    )
  }

  // Home view - show all shared folders
  if (!selectedFolderId) {
    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex-1">
            <h1 className="text-2xl font-semibold">คลังรูปภาพ</h1>
            <p className="text-sm text-muted-foreground">
              จัดการรูปภาพกิจกรรมทั้งหมด
            </p>
          </div>
          <div className="flex items-center gap-2">
            {/* Retry Failed Button */}
            {(stats?.failed_photos || 0) > 0 && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => retryFailedMutation.mutate(undefined)}
                disabled={retryFailedMutation.isPending}
                className="text-destructive border-destructive/50 hover:bg-destructive/10"
              >
                {retryFailedMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    กำลัง Retry...
                  </>
                ) : (
                  <>
                    <RotateCcw className="h-4 w-4 mr-2" />
                    Retry ({stats?.failed_photos})
                  </>
                )}
              </Button>
            )}
          </div>
        </div>

        {/* Key Metrics */}
        <div className={cn(
          "grid gap-4",
          (stats?.failed_photos || 0) > 0
            ? "grid-cols-2 lg:grid-cols-6"
            : "grid-cols-2 lg:grid-cols-5"
        )}>
          <MetricItem
            label="โฟลเดอร์"
            value={sharedFolders.length}
            icon={<FolderTree className="h-3.5 w-3.5" />}
            isLoading={foldersLoading}
          />
          <MetricItem
            label="รูปภาพทั้งหมด"
            value={stats?.total_photos || 0}
            icon={<Images className="h-3.5 w-3.5" />}
            isLoading={statsLoading}
          />
          <MetricItem
            label="ประมวลผลแล้ว"
            value={stats?.processed_photos || 0}
            icon={<CheckCircle2 className="h-3.5 w-3.5" />}
            isLoading={statsLoading}
          />
          <MetricItem
            label="รอประมวลผล"
            value={stats?.pending_photos || 0}
            icon={<Clock className="h-3.5 w-3.5" />}
            isLoading={statsLoading}
          />
          {(stats?.failed_photos || 0) > 0 && (
            <MetricItem
              label="ล้มเหลว"
              value={stats?.failed_photos || 0}
              icon={<AlertTriangle className="h-3.5 w-3.5 text-destructive" />}
              isLoading={statsLoading}
            />
          )}
          <MetricItem
            label="ใบหน้าที่พบ"
            value={stats?.total_faces || 0}
            icon={<Users className="h-3.5 w-3.5" />}
            isLoading={statsLoading}
          />
        </div>

        <Separator />

        {/* Folders Header */}
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium">โฟลเดอร์ทั้งหมด</h2>
          <div className="flex items-center gap-1">
            <div className="flex items-center border rounded-md">
              <button
                onClick={() => setViewMode("list")}
                className={cn(
                  "p-1.5 rounded-l-md transition-colors",
                  viewMode === "list" ? "bg-muted" : "hover:bg-muted/50"
                )}
              >
                <List className="h-4 w-4" />
              </button>
              <button
                onClick={() => setViewMode("grid")}
                className={cn(
                  "p-1.5 rounded-r-md transition-colors",
                  viewMode === "grid" ? "bg-muted" : "hover:bg-muted/50"
                )}
              >
                <LayoutGrid className="h-4 w-4" />
              </button>
            </div>
            <Button variant="ghost" size="icon" onClick={() => navigate("/settings")}>
              <Settings className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Folders */}
        {foldersLoading ? (
          <div className="space-y-1">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-10 bg-muted rounded-md animate-pulse" />
            ))}
          </div>
        ) : sharedFolders.length === 0 ? (
          <div className="py-16 text-center">
            <Folder className="mx-auto h-12 w-12 text-muted-foreground/30" />
            <p className="mt-4 text-sm text-muted-foreground">ยังไม่มีโฟลเดอร์</p>
            <Button variant="outline" className="mt-4" onClick={() => navigate("/settings")}>
              เพิ่มโฟลเดอร์
            </Button>
          </div>
        ) : viewMode === "grid" ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
            {sharedFolders.map(folder => (
              <FolderItemGrid
                key={folder.id}
                name={folder.drive_folder_name}
                photoCount={folder.photo_count}
                onClick={() => handleFolderSelect(folder.id)}
                onSync={() => triggerSyncMutation.mutate({ folderId: folder.id })}
                onForceSync={() => triggerSyncMutation.mutate({ folderId: folder.id, force: true })}
                onReconnect={() => reconnectMutation.mutate(folder.id)}
                isSyncing={syncProgressMap[folder.id] !== undefined || triggerSyncMutation.isPending}
                hasError={folder.sync_status === 'failed'}
              />
            ))}
          </div>
        ) : (
          <div className="space-y-0.5">
            {sharedFolders.map(folder => (
              <FolderItemList
                key={folder.id}
                name={folder.drive_folder_name}
                photoCount={folder.photo_count}
                onClick={() => handleFolderSelect(folder.id)}
                onSync={() => triggerSyncMutation.mutate({ folderId: folder.id })}
                onForceSync={() => triggerSyncMutation.mutate({ folderId: folder.id, force: true })}
                onReconnect={() => reconnectMutation.mutate(folder.id)}
                isSyncing={syncProgressMap[folder.id] !== undefined || triggerSyncMutation.isPending}
                hasError={folder.sync_status === 'failed'}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // Folder view
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex-1">
          <h1 className="text-2xl font-semibold">คลังรูปภาพ</h1>
          <p className="text-sm text-muted-foreground">
            จัดการรูปภาพกิจกรรมทั้งหมด
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Retry Failed Button */}
          {(stats?.failed_photos || 0) > 0 && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => retryFailedMutation.mutate(undefined)}
              disabled={retryFailedMutation.isPending}
              className="text-destructive border-destructive/50 hover:bg-destructive/10"
            >
              {retryFailedMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  กำลัง Retry...
                </>
              ) : (
                <>
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Retry ({stats?.failed_photos})
                </>
              )}
            </Button>
          )}
        </div>
      </div>

      {/* Key Metrics */}
      <div className={cn(
        "grid gap-4",
        (stats?.failed_photos || 0) > 0
          ? "grid-cols-2 lg:grid-cols-6"
          : "grid-cols-2 lg:grid-cols-5"
      )}>
        <MetricItem
          label="โฟลเดอร์"
          value={sharedFolders.length}
          icon={<FolderTree className="h-3.5 w-3.5" />}
          isLoading={foldersLoading}
        />
        <MetricItem
          label="รูปภาพทั้งหมด"
          value={stats?.total_photos || 0}
          icon={<Images className="h-3.5 w-3.5" />}
          isLoading={statsLoading}
        />
        <MetricItem
          label="ประมวลผลแล้ว"
          value={stats?.processed_photos || 0}
          icon={<CheckCircle2 className="h-3.5 w-3.5" />}
          isLoading={statsLoading}
        />
        <MetricItem
          label="รอประมวลผล"
          value={stats?.pending_photos || 0}
          icon={<Clock className="h-3.5 w-3.5" />}
          isLoading={statsLoading}
        />
        {(stats?.failed_photos || 0) > 0 && (
          <MetricItem
            label="ล้มเหลว"
            value={stats?.failed_photos || 0}
            icon={<AlertTriangle className="h-3.5 w-3.5 text-destructive" />}
            isLoading={statsLoading}
          />
        )}
        <MetricItem
          label="ใบหน้าที่พบ"
          value={stats?.total_faces || 0}
          icon={<Users className="h-3.5 w-3.5" />}
          isLoading={statsLoading}
        />
      </div>

      <Separator />

      {/* Breadcrumb & Actions */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-1 text-sm min-w-0">
          <button
            onClick={handleHomeClick}
            className="text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
          >
            <Home className="h-4 w-4" />
          </button>
          {breadcrumbs.map((crumb, idx) => (
            <div key={`${crumb.id}-${crumb.path || 'root'}`} className="flex items-center gap-1 min-w-0">
              <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
              {idx === breadcrumbs.length - 1 ? (
                <span className="truncate">{crumb.name}</span>
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

        <div className="flex items-center gap-1 flex-shrink-0">
          {someSelected && (
            <>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  const selectedPhotoData = photos.filter(p => selectedPhotos.has(p.drive_file_id))
                  sessionStorage.setItem('newsWriterPhotos', JSON.stringify(selectedPhotoData))
                  navigate('/news-writer')
                }}
              >
                <Newspaper className="h-4 w-4" />
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={handleDownload}
                disabled={downloadMutation.isPending}
              >
                {downloadMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Download className="h-4 w-4" />
                )}
              </Button>
              <Separator orientation="vertical" className="h-4" />
            </>
          )}

          <div className="flex items-center border rounded-md">
            <button
              onClick={() => setViewMode("list")}
              className={cn(
                "p-1.5 rounded-l-md transition-colors",
                viewMode === "list" ? "bg-muted" : "hover:bg-muted/50"
              )}
            >
              <List className="h-4 w-4" />
            </button>
            <button
              onClick={() => setViewMode("grid")}
              className={cn(
                "p-1.5 rounded-r-md transition-colors",
                viewMode === "grid" ? "bg-muted" : "hover:bg-muted/50"
              )}
            >
              <LayoutGrid className="h-4 w-4" />
            </button>
          </div>

          <Button
            size="sm"
            variant="ghost"
            onClick={() => triggerSyncMutation.mutate({ folderId: selectedFolderId })}
            disabled={isSyncing || triggerSyncMutation.isPending}
            title="Sync"
          >
            <RefreshCw className={cn("h-4 w-4", isSyncing && "animate-spin")} />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => navigate("/face-search")}>
                <Search className="h-4 w-4 mr-2" />
                ค้นหาใบหน้า
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => navigate("/settings")}>
                <Settings className="h-4 w-4 mr-2" />
                จัดการโฟลเดอร์
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => triggerSyncMutation.mutate({ folderId: selectedFolderId, force: true })}
                disabled={isSyncing || triggerSyncMutation.isPending}
              >
                <RotateCcw className="h-4 w-4 mr-2" />
                Force Full Sync
              </DropdownMenuItem>
              {(stats?.failed_photos || 0) > 0 && (
                <DropdownMenuItem
                  onClick={() => retryFailedMutation.mutate(undefined)}
                  disabled={retryFailedMutation.isPending}
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Retry ({stats?.failed_photos})
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <Separator />

      {/* Content */}
      {photosLoading ? (
        <div className="space-y-1">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-10 bg-muted rounded-md animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          {/* Info bar */}
          {(subfolders.length > 0 || photos.length > 0) && (
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-2">
                {photos.length > 0 && (
                  <button
                    onClick={toggleSelectAll}
                    className="flex items-center gap-1.5 hover:text-foreground transition-colors"
                  >
                    <div className={cn(
                      "h-3.5 w-3.5 rounded border-2 flex items-center justify-center",
                      allSelected ? "bg-green-500 border-green-500 text-white" : "border-gray-400"
                    )}>
                      {allSelected && <Check className="h-2.5 w-2.5" strokeWidth={3} />}
                    </div>
                    เลือกทั้งหมด
                  </button>
                )}
                {someSelected && (
                  <span className="text-green-600 dark:text-green-400">{selectedPhotos.size} รายการ</span>
                )}
              </div>
              <span>
                {subfolders.length > 0 && `${subfolders.length} โฟลเดอร์`}
                {subfolders.length > 0 && photos.length > 0 && " • "}
                {photos.length > 0 && `${photosData?.total || 0} รูป`}
                {totalPages > 1 && ` • หน้า ${page}/${totalPages}`}
              </span>
            </div>
          )}

          {/* Subfolders */}
          {subfolders.length > 0 && (
            viewMode === "grid" ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
                {subfolders.map(subfolder => (
                  <FolderItemGrid
                    key={subfolder.fullPath}
                    name={subfolder.name}
                    photoCount={subfolder.photoCount}
                    onClick={() => handleSubfolderClick(subfolder)}
                  />
                ))}
              </div>
            ) : (
              <div className="space-y-0.5">
                {subfolders.map(subfolder => (
                  <FolderItemList
                    key={subfolder.fullPath}
                    name={subfolder.name}
                    photoCount={subfolder.photoCount}
                    onClick={() => handleSubfolderClick(subfolder)}
                  />
                ))}
              </div>
            )
          )}

          {/* Separator between folders and photos */}
          {subfolders.length > 0 && photos.length > 0 && (
            <Separator className="my-2" />
          )}

          {/* Photos */}
          {photos.length > 0 ? (
            <PhotoProvider maskOpacity={0.9}>
              {viewMode === "grid" ? (
                <div className="grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
                  {photos.map(photo => (
                    <PhotoCardGrid
                      key={photo.id}
                      photo={photo}
                      token={token}
                      isSelected={selectedPhotos.has(photo.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(photo.drive_file_id)}
                    />
                  ))}
                </div>
              ) : (
                <div className="space-y-0.5">
                  {photos.map(photo => (
                    <PhotoCardList
                      key={photo.id}
                      photo={photo}
                      token={token}
                      isSelected={selectedPhotos.has(photo.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(photo.drive_file_id)}
                    />
                  ))}
                </div>
              )}
            </PhotoProvider>
          ) : subfolders.length === 0 && (
            <div className="py-16 text-center">
              <Images className="mx-auto h-12 w-12 text-muted-foreground/30" />
              <p className="mt-4 text-sm text-muted-foreground">ไม่พบรูปภาพ</p>
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
              <span className="text-sm text-muted-foreground">
                {page} / {totalPages}
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
