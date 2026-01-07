import { useState, useEffect, useCallback } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { PhotoProvider, PhotoView } from "react-photo-view"
import "react-photo-view/dist/react-photo-view.css"
import {
  Images,
  FolderTree,
  Folder,
  FolderOpen,
  RefreshCw,
  Loader2,
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Clock,
  ImageOff,
  Users,
  Search,
  ArrowRight,
  X,
  Download,
  CheckSquare,
  Square,
  Check,
  ZoomIn,
  Newspaper,
  RotateCcw,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

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
import type { Photo } from "@/shared/types"

// Metric Item Component
interface MetricItemProps {
  label: string
  value: string | number
  subtext?: string
  icon?: React.ReactNode
  isLoading?: boolean
}

function MetricItem({ label, value, subtext, icon, isLoading }: MetricItemProps) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground flex items-center gap-1">
        {icon}
        {label}
      </p>
      {isLoading ? (
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      ) : (
        <>
          <p className="text-lg font-semibold tabular-nums">{value}</p>
          {subtext && <p className="text-xs text-muted-foreground">{subtext}</p>}
        </>
      )}
    </div>
  )
}

// Tree node interface for folder tree
interface FolderTreeNode {
  name: string
  path: string
  photoCount: number
  children: FolderTreeNode[]
  folderId: string
  syncStatus: string
}

// Recursive Tree Node component
function TreeNodeItem({
  node,
  level = 0,
  selectedFolderId,
  selectedPath,
  onSelect,
  onSync,
  isSyncing,
  syncProgress,
}: {
  node: FolderTreeNode
  level?: number
  selectedFolderId?: string
  selectedPath?: string
  onSelect: (folderId: string, path: string | undefined, name: string) => void
  onSync: (folderId: string) => void
  isSyncing: boolean
  syncProgress?: { percent: number } | null
}) {
  const [isExpanded, setIsExpanded] = useState(true)
  const hasChildren = node.children.length > 0
  const isRoot = level === 0
  const isSelected = isRoot
    ? (selectedFolderId === node.folderId && !selectedPath)
    : (selectedFolderId === node.folderId && selectedPath === node.path)

  return (
    <div>
      <div
        className={cn(
          "flex items-center justify-between py-1 px-1 rounded cursor-pointer transition-colors group",
          isSelected ? "bg-primary/10 text-primary" : "hover:bg-muted/50"
        )}
        style={{ paddingLeft: `${level * 12 + 4}px` }}
      >
        <button
          className="flex items-center gap-1 flex-1 text-left min-w-0"
          onClick={() => onSelect(node.folderId, isRoot ? undefined : node.path, node.name)}
        >
          <button
            className="p-0.5 hover:bg-muted rounded"
            onClick={(e) => {
              e.stopPropagation()
              if (hasChildren) setIsExpanded(!isExpanded)
            }}
          >
            {isExpanded && hasChildren ? (
              <FolderOpen className="h-3 w-3 text-yellow-500" />
            ) : (
              <Folder className="h-3 w-3 text-yellow-500" />
            )}
          </button>
          <span className="text-xs truncate">{node.name}</span>
          <span className="text-[10px] text-muted-foreground flex-shrink-0">
            ({node.photoCount})
          </span>
        </button>
        {isRoot && (
          <div className="flex items-center gap-1 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
            {syncProgress && (
              <span className="text-[10px] text-primary font-medium">{syncProgress.percent}%</span>
            )}
            <button
              className="p-1 hover:bg-muted rounded"
              onClick={(e) => {
                e.stopPropagation()
                onSync(node.folderId)
              }}
              disabled={isSyncing}
              title="Sync โฟลเดอร์นี้"
            >
              <RefreshCw className={cn("h-3 w-3 text-muted-foreground", isSyncing && "animate-spin")} />
            </button>
          </div>
        )}
      </div>
      {hasChildren && isExpanded && (
        <div>
          {node.children.map((child, idx) => (
            <TreeNodeItem
              key={`${child.path}-${idx}`}
              node={child}
              level={level + 1}
              selectedFolderId={selectedFolderId}
              selectedPath={selectedPath}
              onSelect={onSelect}
              onSync={onSync}
              isSyncing={isSyncing}
              syncProgress={syncProgress}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// Photo card component with selection support
function PhotoCard({
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
          isSelected && "ring-1 ring-green-500"
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
          {/* Zoom indicator on hover */}
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center pointer-events-none">
            <ZoomIn className="h-8 w-8 text-white opacity-0 group-hover:opacity-70 transition-opacity" />
          </div>
          {/* Status indicator */}
          <div className="absolute top-1.5 left-1.5">
            {getStatusIcon()}
          </div>
          {/* Face count */}
          {photo.face_count > 0 && (
            <div className="absolute bottom-1.5 right-1.5 bg-black/60 text-white text-xs px-1.5 py-0.5 rounded flex items-center gap-1">
              <Users className="h-3 w-3" />
              {photo.face_count}
            </div>
          )}
        </div>
      </PhotoView>

      {/* Selection checkbox */}
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
            ? "bg-green-500 border-green-500 text-white"
            : "bg-white/90 border-gray-300 hover:border-green-500 hover:bg-white"
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

export default function GalleryPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { token } = useAuth()
  const [selectedFolderId, setSelectedFolderId] = useState<string | undefined>()
  const [selectedFolderName, setSelectedFolderName] = useState<string | undefined>()
  const [selectedSubFolder, setSelectedSubFolder] = useState<string | undefined>()
  const [selectedSubFolderName, setSelectedSubFolderName] = useState<string | undefined>()
  const [page, setPage] = useState(1)
  const [folderFilter, setFolderFilter] = useState("")
  const limit = 24

  // Multi-select state
  const [selectedPhotos, setSelectedPhotos] = useState<Set<string>>(new Set())
  const downloadMutation = useDownloadPhotos()
  const downloadProgress = useDownloadProgressStore((state) => state.progress)
  const syncProgressMap = useSyncProgressStore((state) => state.progress)

  // WebSocket connection is handled at layout level (PageLayout)

  // Read folder from URL params on mount (only once)
  useEffect(() => {
    const folderParam = searchParams.get("folder")
    const nameParam = searchParams.get("name")
    if (folderParam && !selectedFolderId) {
      setSelectedFolderId(folderParam)
      setSelectedFolderName(nameParam || undefined)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const { data: driveStatus, isLoading: statusLoading } = useDriveStatus()
  const { data: sharedFoldersData, isLoading: foldersLoading } = useSharedFolders(driveStatus?.connected)
  const sharedFolders = sharedFoldersData?.folders || []

  // Get selected folder for sync status
  const selectedFolder = sharedFolders.find(f => f.id === selectedFolderId)

  // Get photos from selected folder (optionally filtered by subfolder)
  const { data: photosData, isLoading: photosLoading } = useFolderPhotos(
    selectedFolderId || "",
    page,
    limit,
    undefined,
    !!selectedFolderId,
    selectedSubFolder
  )
  const { data: stats, isLoading: statsLoading } = useFaceStats()

  // Sync folder
  const triggerSyncMutation = useTriggerFolderSync()

  // Retry failed photos
  const retryFailedMutation = useRetryFailedPhotos()

  // Build tree structure from folder paths (supports unlimited nesting levels)
  interface TreeNode {
    name: string
    path: string
    photoCount: number
    children: TreeNode[]
    folderId: string
    syncStatus: string
  }

  const buildFolderTree = useCallback(() => {
    const trees: TreeNode[] = []

    sharedFolders.forEach(folder => {
      // Create root node for each shared folder
      const rootNode: TreeNode = {
        name: folder.drive_folder_name,
        path: '',
        photoCount: folder.photo_count,
        children: [],
        folderId: folder.id,
        syncStatus: folder.sync_status,
      }

      // Build tree from children paths
      const pathMap = new Map<string, TreeNode>()
      pathMap.set('', rootNode)

      // Sort children by path to ensure parents are created before children
      const sortedChildren = [...(folder.children || [])].sort((a, b) =>
        a.path.localeCompare(b.path)
      )

      sortedChildren.forEach(child => {
        const pathParts = child.path.split('/')
        let currentPath = ''
        let parentNode = rootNode

        // Navigate/create path to this node
        for (let i = 1; i < pathParts.length; i++) { // Skip first part (root folder name)
          const part = pathParts[i]
          const newPath = currentPath ? `${currentPath}/${part}` : part

          let node = pathMap.get(newPath)
          if (!node) {
            node = {
              name: part,
              path: child.path, // Use full path for leaf nodes
              photoCount: 0,
              children: [],
              folderId: folder.id,
              syncStatus: folder.sync_status,
            }
            pathMap.set(newPath, node)
            parentNode.children.push(node)
          }

          // Update photo count for leaf node
          if (i === pathParts.length - 1) {
            node.photoCount = child.photo_count
            node.path = child.path
          }

          parentNode = node
          currentPath = newPath
        }
      })

      trees.push(rootNode)
    })

    return trees
  }, [sharedFolders])

  const folderTree = buildFolderTree()

  // Filter tree by search term (flatten and filter, then show matching items)
  const filterTree = useCallback((nodes: TreeNode[], filter: string): TreeNode[] => {
    if (!filter) return nodes
    const lowerFilter = filter.toLowerCase()

    const filterNode = (node: TreeNode): TreeNode | null => {
      const nameMatches = node.name.toLowerCase().includes(lowerFilter)
      const filteredChildren = node.children.map(filterNode).filter(Boolean) as TreeNode[]

      if (nameMatches || filteredChildren.length > 0) {
        return { ...node, children: filteredChildren }
      }
      return null
    }

    return nodes.map(filterNode).filter(Boolean) as TreeNode[]
  }, [])

  const filteredTree = filterTree(folderTree, folderFilter)

  // Auto-select first folder if none selected
  useEffect(() => {
    if (!selectedFolderId && sharedFolders.length > 0 && !searchParams.get("folder")) {
      const firstFolder = sharedFolders[0]
      setSelectedFolderId(firstFolder.id)
      setSelectedFolderName(firstFolder.drive_folder_name)
    }
  }, [sharedFolders, selectedFolderId, searchParams])

  // Unified handler for tree node selection (works for both root folders and subfolders)
  const handleTreeSelect = (folderId: string, path: string | undefined, name: string) => {
    const isCurrentlySelected = selectedFolderId === folderId && selectedSubFolder === path

    if (isCurrentlySelected) {
      // Deselect - clear all
      setSelectedFolderId(undefined)
      setSelectedFolderName(undefined)
      setSelectedSubFolder(undefined)
      setSelectedSubFolderName(undefined)
      setSearchParams({}, { replace: true })
    } else {
      // Select
      setSelectedFolderId(folderId)
      setSelectedFolderName(name)
      setSelectedSubFolder(path)
      setSelectedSubFolderName(path ? name : undefined)
      setSearchParams({ folder: folderId, name }, { replace: true })
    }
    setPage(1)
    setSelectedPhotos(new Set())
  }

  // Multi-select handlers
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

  // Reset selection when page changes
  useEffect(() => {
    setSelectedPhotos(new Set())
  }, [page])

  // Get selected folder's sync status and progress
  const syncProgress = selectedFolderId ? syncProgressMap[selectedFolderId] : undefined
  const isSyncing = selectedFolder?.sync_status === 'syncing' || !!syncProgress
  const totalPages = photosData ? Math.ceil(photosData.total / limit) : 0

  // Selection computed values
  const photos = photosData?.photos || []
  const allSelected = photos.length > 0 && photos.every(p => selectedPhotos.has(p.drive_file_id))
  const someSelected = selectedPhotos.size > 0

  // Not connected or no folders state
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex-1">
          <h1 className="text-2xl font-semibold">คลังรูปภาพ</h1>
          <p className="text-sm text-muted-foreground">
            {photosData ? `${photosData.total} รูปภาพ` : "กำลังโหลด..."}
            {selectedSubFolderName ? ` ใน "${selectedSubFolderName}"` : selectedFolderName && ` ใน "${selectedFolderName}"`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {someSelected && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  // Store selected photos data in sessionStorage for news-writer
                  const selectedPhotoData = photos.filter(p => selectedPhotos.has(p.drive_file_id))
                  sessionStorage.setItem('newsWriterPhotos', JSON.stringify(selectedPhotoData))
                  navigate('/news-writer')
                }}
              >
                <Newspaper className="h-4 w-4 mr-2" />
                เขียนข่าว ({selectedPhotos.size})
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={handleDownload}
                disabled={downloadMutation.isPending}
                className="min-w-[180px]"
              >
                {downloadMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    {downloadProgress
                      ? `${downloadProgress.current}/${downloadProgress.total} ไฟล์`
                      : 'กำลังเตรียม...'}
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-2" />
                    ดาวน์โหลด ({selectedPhotos.size})
                  </>
                )}
              </Button>
            </>
          )}
          <Button
            size="sm"
            onClick={() => selectedFolderId && triggerSyncMutation.mutate(selectedFolderId)}
            disabled={isSyncing || triggerSyncMutation.isPending || !selectedFolderId}
            className="min-w-[120px]"
          >
            {isSyncing ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {syncProgress ? `กำลัง Sync ${syncProgress.percent}%` : 'กำลัง Sync...'}
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4 mr-2" />
                Sync
              </>
            )}
          </Button>
          {/* Retry Failed Button - only show if there are failed photos */}
          {(stats?.failed_photos || 0) > 0 && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => retryFailedMutation.mutate()}
              disabled={retryFailedMutation.isPending}
              className="text-orange-600 border-orange-300 hover:bg-orange-50"
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
            icon={<AlertTriangle className="h-3.5 w-3.5 text-orange-500" />}
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

      {/* Main Content */}
      <div className="grid grid-cols-12 gap-6">
        {/* Sidebar - Shared Folders */}
        <div className="col-span-12 lg:col-span-2">
          <div className="space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <FolderTree className="h-3.5 w-3.5" />
              โฟลเดอร์ ({sharedFolders.length})
            </h3>

            {/* Folder Filter Input - minimal style */}
            <div className="relative">
              <input
                type="text"
                placeholder="ค้นหาโฟลเดอร์..."
                value={folderFilter}
                onChange={(e) => setFolderFilter(e.target.value)}
                className="w-full text-xs bg-transparent border-0 border-b border-muted-foreground/30 focus:border-primary focus:outline-none py-1.5 px-1 placeholder:text-muted-foreground/50 transition-colors"
              />
              {folderFilter && (
                <button
                  onClick={() => setFolderFilter("")}
                  className="absolute right-1 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>

            {foldersLoading ? (
              <div className="space-y-1">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-6 bg-muted rounded animate-pulse" />
                ))}
              </div>
            ) : filteredTree.length > 0 ? (
              <div className="space-y-0.5 max-h-[400px] overflow-y-auto">
                {filteredTree.map((node) => (
                  <TreeNodeItem
                    key={node.folderId}
                    node={node as FolderTreeNode}
                    selectedFolderId={selectedFolderId}
                    selectedPath={selectedSubFolder}
                    onSelect={handleTreeSelect}
                    onSync={(folderId) => triggerSyncMutation.mutate(folderId)}
                    isSyncing={triggerSyncMutation.isPending || !!syncProgressMap[node.folderId]}
                    syncProgress={syncProgressMap[node.folderId]}
                  />
                ))}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground text-center py-4">
                {folderFilter ? `ไม่พบ "${folderFilter}"` : "ไม่มีโฟลเดอร์"}
              </p>
            )}
          </div>

          <Separator className="my-4" />

          {/* Quick Links */}
          <div className="space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground">เมนูลัด</h3>
            <div className="space-y-1">
              <Button
                variant="ghost"
                size="sm"
                className="w-full justify-start h-7 text-xs"
                onClick={() => navigate("/face-search")}
              >
                <Search className="h-3.5 w-3.5 mr-1.5" />
                ค้นหาใบหน้า
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="w-full justify-start h-7 text-xs"
                onClick={() => navigate("/settings")}
              >
                <FolderOpen className="h-3.5 w-3.5 mr-1.5" />
                ตั้งค่าโฟลเดอร์
              </Button>
            </div>
          </div>
        </div>

        {/* Photo Grid */}
        <div className="col-span-12 lg:col-span-10">
          {photosLoading ? (
            <div className="grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {Array.from({ length: 18 }).map((_, i) => (
                <div key={i}>
                  <div className="aspect-square bg-muted rounded-lg animate-pulse" />
                  <div className="h-3 bg-muted rounded mt-1.5 w-3/4 animate-pulse" />
                </div>
              ))}
            </div>
          ) : photosData && photosData.photos.length > 0 ? (
            <div className="space-y-4">
              {/* Select All Header */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <button
                    onClick={toggleSelectAll}
                    className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
                  >
                    {allSelected ? (
                      <CheckSquare className="h-4 w-4 text-primary" />
                    ) : (
                      <Square className="h-4 w-4" />
                    )}
                    เลือกทั้งหมด ({photos.length})
                  </button>
                  {someSelected && (
                    <span className="text-xs text-muted-foreground">
                      เลือก {selectedPhotos.size} รูป
                    </span>
                  )}
                </div>
                {/* Page indicator */}
                {totalPages > 1 && (
                  <span className="text-xs text-muted-foreground">
                    หน้า {page} / {totalPages}
                  </span>
                )}
              </div>

              {/* Photo Grid with PhotoProvider */}
              <PhotoProvider
                maskOpacity={0.9}
                toolbarRender={({ onScale, scale }) => (
                  <div className="flex items-center gap-2">
                    <button
                      className="p-2 text-white/80 hover:text-white"
                      onClick={() => onScale(scale + 0.5)}
                    >
                      <ZoomIn className="h-5 w-5" />
                    </button>
                    <button
                      className="p-2 text-white/80 hover:text-white"
                      onClick={() => onScale(scale - 0.5)}
                    >
                      <X className="h-5 w-5" />
                    </button>
                  </div>
                )}
              >
                <div className="grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
                  {photos.map((photo) => (
                    <PhotoCard
                      key={photo.id}
                      photo={photo}
                      token={token}
                      isSelected={selectedPhotos.has(photo.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(photo.drive_file_id)}
                    />
                  ))}
                </div>
              </PhotoProvider>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-center gap-2 mt-6">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                  >
                    ก่อนหน้า
                  </Button>
                  <span className="text-xs text-muted-foreground">
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
          ) : (
            <div className="py-12 text-center">
              <Images className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                {selectedFolderName
                  ? `ไม่พบรูปภาพในโฟลเดอร์ "${selectedFolderName}"`
                  : "ยังไม่มีรูปภาพ กด Sync เพื่อดึงรูปจาก Google Drive"}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
