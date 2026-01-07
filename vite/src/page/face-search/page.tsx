import { useState, useCallback, useRef, useEffect } from "react"
import { Link } from "react-router-dom"
import {
  ScanFace,
  Upload,
  X,
  Search,
  Images,
  Loader2,
  FolderOpen,
  CheckCircle2,
  Check,
  Clock,
  AlertCircle,
  Users,
  ImageOff,
  Settings2,
  UserCircle,
  Download,
  CheckSquare,
  Square,
  ChevronLeft,
  ChevronRight,
  ZoomIn,
  ZoomOut,
} from "lucide-react"
import { PhotoProvider, PhotoView } from "react-photo-view"
import "react-photo-view/dist/react-photo-view.css"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Slider } from "@/components/ui/slider"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

// Feature imports
import { useFaceSearchByImage, useFaceStats, useDetectFaces } from "@/features/face-search"
import { FACE_SEARCH, ALLOWED_IMAGE_TYPES, FORM_LIMITS, getThumbnailUrl } from "@/shared/config/constants"
import { useDownloadPhotos } from "@/features/drive"
import { useAuth } from "@/hooks/use-auth"
import { useDownloadProgressStore } from "@/stores/download-progress"
import type { FaceSearchResult, DetectedFace } from "@/shared/types"

// Pagination constants
const DEFAULT_ITEMS_PER_PAGE = 24
const MIN_ITEMS_PER_PAGE = 12
const MAX_ITEMS_PER_PAGE = 60

// Metric Item Component (same style as Gallery)
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

export default function FaceSearchPage() {
  // State
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [threshold, setThreshold] = useState<number>(FACE_SEARCH.DEFAULT_THRESHOLD)
  const [itemsPerPage, setItemsPerPage] = useState<number>(DEFAULT_ITEMS_PER_PAGE)
  const [showSettings, setShowSettings] = useState(false)
  const [detectedFaces, setDetectedFaces] = useState<DetectedFace[]>([])
  const [selectedFaceIndex, setSelectedFaceIndex] = useState<number>(0)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Multi-select state
  const [selectedPhotos, setSelectedPhotos] = useState<Set<string>>(new Set())

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1)

  // Hooks
  const { token } = useAuth()
  const { data: stats, isLoading: statsLoading } = useFaceStats()
  const searchMutation = useFaceSearchByImage()
  const detectMutation = useDetectFaces()
  const downloadMutation = useDownloadPhotos()
  const downloadProgress = useDownloadProgressStore((state) => state.progress)

  // WebSocket connection is handled at layout level (PageLayout)

  // Reset selection when search results change
  useEffect(() => {
    setSelectedPhotos(new Set())
    setCurrentPage(1)
  }, [searchMutation.data])

  // Auto-detect faces when file changes
  useEffect(() => {
    if (selectedFile) {
      detectMutation.mutate(selectedFile, {
        onSuccess: (data) => {
          setDetectedFaces(data.faces)
          setSelectedFaceIndex(0)
        },
        onError: () => {
          setDetectedFaces([])
          setSelectedFaceIndex(0)
        },
      })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedFile])

  // Handlers
  const handleFileChange = useCallback((file: File | null) => {
    if (file) {
      if (!ALLOWED_IMAGE_TYPES.includes(file.type as typeof ALLOWED_IMAGE_TYPES[number])) {
        alert("กรุณาเลือกไฟล์รูปภาพ (JPG, PNG, WebP, GIF)")
        return
      }
      if (file.size > FORM_LIMITS.FILE_SIZE_MAX) {
        alert("ขนาดไฟล์ต้องไม่เกิน 10MB")
        return
      }
      setSelectedFile(file)
      const url = URL.createObjectURL(file)
      setPreviewUrl(url)
      // Reset face detection state
      setDetectedFaces([])
      setSelectedFaceIndex(0)
    } else {
      setSelectedFile(null)
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl)
      }
      setPreviewUrl(null)
      setDetectedFaces([])
      setSelectedFaceIndex(0)
    }
  }, [previewUrl])

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault()
      const file = e.dataTransfer.files[0]
      handleFileChange(file)
    },
    [handleFileChange]
  )

  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
  }, [])

  const handleSearch = useCallback(() => {
    if (!selectedFile) return
    searchMutation.mutate({
      imageFile: selectedFile,
      limit: FACE_SEARCH.MAX_LIMIT, // Always fetch max results, paginate on frontend
      threshold,
      faceIndex: selectedFaceIndex,
    })
  }, [selectedFile, threshold, selectedFaceIndex, searchMutation])

  const clearSelection = useCallback(() => {
    handleFileChange(null)
    searchMutation.reset()
  }, [handleFileChange, searchMutation])

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
    if (!searchMutation.data) return

    const allIds = searchMutation.data.results.map(r => r.drive_file_id)
    const allSelected = allIds.every(id => selectedPhotos.has(id))

    if (allSelected) {
      setSelectedPhotos(new Set())
    } else {
      setSelectedPhotos(new Set(allIds))
    }
  }, [searchMutation.data, selectedPhotos])

  const handleDownload = useCallback(() => {
    if (selectedPhotos.size === 0) return
    downloadMutation.mutate(Array.from(selectedPhotos))
  }, [selectedPhotos, downloadMutation])

  // Computed
  const processingProgress = stats
    ? ((stats.processed_photos / stats.total_photos) * 100) || 0
    : 0

  const results = searchMutation.data?.results || []
  const totalResults = results.length
  const totalPages = Math.ceil(totalResults / itemsPerPage)
  const paginatedResults = results.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  )

  const allSelected = results.length > 0 && results.every(r => selectedPhotos.has(r.drive_file_id))
  const someSelected = selectedPhotos.size > 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">ค้นหาใบหน้า</h1>
          <p className="text-sm text-muted-foreground">
            {searchMutation.data
              ? `พบ ${searchMutation.data.count} รูปที่ตรงกัน`
              : "อัปโหลดรูปใบหน้าเพื่อค้นหา"
            }
          </p>
        </div>
        <div className="flex items-center gap-2">
          {someSelected && (
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
          )}
          <Button
            size="sm"
            onClick={handleSearch}
            disabled={!selectedFile || searchMutation.isPending || detectMutation.isPending || detectedFaces.length === 0}
          >
            {searchMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                กำลังค้นหา...
              </>
            ) : (
              <>
                <Search className="h-4 w-4 mr-2" />
                ค้นหา
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
        <MetricItem
          label="รูปภาพทั้งหมด"
          value={stats?.total_photos || 0}
          icon={<Images className="h-3.5 w-3.5" />}
          isLoading={statsLoading}
        />
        <MetricItem
          label="ใบหน้าที่พบ"
          value={stats?.total_faces || 0}
          icon={<Users className="h-3.5 w-3.5" />}
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
        <MetricItem
          label="ความเหมือนขั้นต่ำ"
          value={`${(threshold * 100).toFixed(0)}%`}
          icon={<ScanFace className="h-3.5 w-3.5" />}
        />
      </div>

      {/* Processing Progress */}
      {stats && stats.pending_photos > 0 && (
        <Progress value={processingProgress} className="h-1.5" />
      )}

      <Separator />

      {/* Main Content */}
      <div className="grid grid-cols-12 gap-6">
        {/* Sidebar - Upload & Settings */}
        <div className="col-span-12 lg:col-span-2">
          <div className="space-y-4">
            {/* Upload Zone */}
            <div className="space-y-2">
              <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Upload className="h-3.5 w-3.5" />
                รูปค้นหา
              </h3>

              <div
                className={cn(
                  "relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-4 transition-colors cursor-pointer",
                  selectedFile
                    ? "border-primary bg-primary/5"
                    : "border-muted-foreground/25 hover:border-primary/50 hover:bg-muted/50"
                )}
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onClick={() => fileInputRef.current?.click()}
                role="button"
                tabIndex={0}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) => handleFileChange(e.target.files?.[0] || null)}
                />

                {previewUrl ? (
                  <div className="relative w-full">
                    <img
                      src={previewUrl}
                      alt="Preview"
                      className="w-full aspect-square object-cover rounded-lg"
                    />
                    {/* Face detection bounding boxes */}
                    {detectMutation.isPending && (
                      <div className="absolute inset-0 flex items-center justify-center bg-black/30 rounded-lg">
                        <Loader2 className="h-6 w-6 animate-spin text-white" />
                      </div>
                    )}
                    {detectedFaces.map((face, index) => (
                      <button
                        key={index}
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelectedFaceIndex(index)
                        }}
                        className={cn(
                          "absolute border-2 rounded transition-all",
                          index === selectedFaceIndex
                            ? "border-green-500 bg-green-500/20 ring-2 ring-green-500 ring-offset-1"
                            : "border-white/70 hover:border-green-400 hover:bg-green-500/10"
                        )}
                        style={{
                          left: `${face.bbox_x * 100}%`,
                          top: `${face.bbox_y * 100}%`,
                          width: `${face.bbox_width * 100}%`,
                          height: `${face.bbox_height * 100}%`,
                        }}
                        title={`ใบหน้าที่ ${index + 1}`}
                      >
                        {detectedFaces.length > 1 && (
                          <span className={cn(
                            "absolute -top-2 -left-2 flex h-4 w-4 items-center justify-center rounded-full text-[10px] font-medium",
                            index === selectedFaceIndex
                              ? "bg-green-500 text-white"
                              : "bg-white text-gray-700"
                          )}>
                            {index + 1}
                          </span>
                        )}
                      </button>
                    ))}
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        clearSelection()
                      }}
                      className="absolute -right-2 -top-2 rounded-full bg-destructive p-1 text-white hover:bg-destructive/80"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                      <Upload className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <p className="mt-2 text-xs text-center text-muted-foreground">
                      ลากไฟล์มาวาง<br />หรือคลิกเลือก
                    </p>
                  </>
                )}
              </div>
            </div>

            {/* Face Detection Info */}
            {selectedFile && (
              <>
                <Separator />
                <div className="space-y-2">
                  <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                    <UserCircle className="h-3.5 w-3.5" />
                    ใบหน้าที่ตรวจพบ
                  </h3>

                  {detectMutation.isPending ? (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      กำลังตรวจจับ...
                    </div>
                  ) : detectMutation.isError ? (
                    <div className="flex items-start gap-2 text-xs text-destructive">
                      <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                      <span>{detectMutation.error?.message || "ตรวจจับใบหน้าล้มเหลว"}</span>
                    </div>
                  ) : detectedFaces.length === 0 ? (
                    <div className="flex items-start gap-2 text-xs text-amber-600">
                      <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                      <span>ไม่พบใบหน้า กรุณาใช้รูปที่เห็นใบหน้าชัดเจน</span>
                    </div>
                  ) : detectedFaces.length === 1 ? (
                    <p className="text-xs text-muted-foreground">
                      พบ 1 ใบหน้า พร้อมค้นหา
                    </p>
                  ) : (
                    <div className="space-y-1.5">
                      <p className="text-xs text-muted-foreground">
                        พบ {detectedFaces.length} ใบหน้า
                      </p>
                      <p className="text-xs text-green-600">
                        คลิกที่ใบหน้าที่ต้องการค้นหา
                      </p>
                      <Badge className="text-xs bg-green-500 hover:bg-green-600">
                        เลือก: ใบหน้าที่ {selectedFaceIndex + 1}
                      </Badge>
                    </div>
                  )}
                </div>
              </>
            )}

            <Separator />

            {/* Settings */}
            <div className="space-y-2">
              <button
                className="w-full flex items-center justify-between text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowSettings(!showSettings)}
              >
                <span className="flex items-center gap-1.5">
                  <Settings2 className="h-3.5 w-3.5" />
                  ตั้งค่าการค้นหา
                </span>
                <span>{showSettings ? "−" : "+"}</span>
              </button>

              {showSettings && (
                <div className="space-y-4 pt-2">
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-xs">
                      <Label className="text-xs">ความเหมือน</Label>
                      <span className="text-muted-foreground">
                        {(threshold * 100).toFixed(0)}%
                      </span>
                    </div>
                    <Slider
                      value={[threshold]}
                      onValueChange={([value]) => setThreshold(value)}
                      min={FACE_SEARCH.MIN_THRESHOLD}
                      max={FACE_SEARCH.MAX_THRESHOLD}
                      step={0.05}
                    />
                  </div>

                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-xs">
                      <Label className="text-xs">แสดงหน้าละ</Label>
                      <span className="text-muted-foreground">{itemsPerPage} รูป</span>
                    </div>
                    <Slider
                      value={[itemsPerPage]}
                      onValueChange={([value]) => {
                        setItemsPerPage(value)
                        setCurrentPage(1) // Reset to first page when changing items per page
                      }}
                      min={MIN_ITEMS_PER_PAGE}
                      max={MAX_ITEMS_PER_PAGE}
                      step={6}
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Error Message */}
            {searchMutation.isError && (
              <>
                <Separator />
                <div className="flex items-start gap-2 text-xs text-destructive">
                  <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                  <span>{searchMutation.error?.message || "เกิดข้อผิดพลาด"}</span>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Results Grid */}
        <div className="col-span-12 lg:col-span-10">
          {searchMutation.isPending ? (
            <div className="grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {Array.from({ length: 18 }).map((_, i) => (
                <div key={i}>
                  <div className="aspect-square bg-muted rounded-lg animate-pulse" />
                  <div className="h-3 bg-muted rounded mt-1.5 w-3/4 animate-pulse" />
                </div>
              ))}
            </div>
          ) : searchMutation.data && searchMutation.data.results.length > 0 ? (
            <div className="space-y-4">
              {/* Select All & Pagination Header */}
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
                    เลือกทั้งหมด ({results.length})
                  </button>
                  {someSelected && (
                    <span className="text-xs text-muted-foreground">
                      เลือก {selectedPhotos.size} รูป
                    </span>
                  )}
                </div>

                {/* Pagination Info */}
                {totalPages > 1 && (
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">
                      หน้า {currentPage} / {totalPages}
                    </span>
                    <div className="flex items-center gap-1">
                      <Button
                        size="icon"
                        variant="outline"
                        className="h-7 w-7"
                        onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                        disabled={currentPage === 1}
                      >
                        <ChevronLeft className="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="outline"
                        className="h-7 w-7"
                        onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                        disabled={currentPage === totalPages}
                      >
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </div>

              {/* Results Grid with PhotoProvider for gallery */}
              <PhotoProvider
                maskOpacity={0.9}
                toolbarRender={({ onScale, scale }) => (
                  <div className="flex items-center gap-2">
                    <button
                      className="p-2 text-white/80 hover:text-white"
                      onClick={() => onScale(scale + 0.5)}
                      title="ซูมเข้า"
                    >
                      <ZoomIn className="h-5 w-5" />
                    </button>
                    <button
                      className="p-2 text-white/80 hover:text-white"
                      onClick={() => onScale(scale - 0.5)}
                      title="ซูมออก"
                    >
                      <ZoomOut className="h-5 w-5" />
                    </button>
                  </div>
                )}
              >
                <div className="grid grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
                  {paginatedResults.map((result) => (
                    <FaceResultCard
                      key={result.face_id}
                      result={result}
                      token={token}
                      isSelected={selectedPhotos.has(result.drive_file_id)}
                      onToggleSelect={() => togglePhotoSelection(result.drive_file_id)}
                    />
                  ))}
                </div>
              </PhotoProvider>

              {/* Bottom Pagination */}
              {totalPages > 1 && (
                <div className="flex justify-center gap-1">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                    disabled={currentPage === 1}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    ก่อนหน้า
                  </Button>
                  <div className="flex items-center gap-1 px-2">
                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                      let pageNum: number
                      if (totalPages <= 5) {
                        pageNum = i + 1
                      } else if (currentPage <= 3) {
                        pageNum = i + 1
                      } else if (currentPage >= totalPages - 2) {
                        pageNum = totalPages - 4 + i
                      } else {
                        pageNum = currentPage - 2 + i
                      }
                      return (
                        <Button
                          key={pageNum}
                          size="sm"
                          variant={currentPage === pageNum ? "default" : "outline"}
                          className="h-8 w-8 p-0"
                          onClick={() => setCurrentPage(pageNum)}
                        >
                          {pageNum}
                        </Button>
                      )
                    })}
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                    disabled={currentPage === totalPages}
                  >
                    ถัดไป
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              )}
            </div>
          ) : searchMutation.data ? (
            <div className="py-12 text-center">
              <ScanFace className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                ไม่พบใบหน้าที่ตรงกัน
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                ลองปรับค่าความเหมือนให้ต่ำลง หรือใช้รูปอื่น
              </p>
            </div>
          ) : (
            <div className="py-12 text-center">
              <Search className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                พร้อมค้นหา
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                อัปโหลดรูปใบหน้าทางซ้ายแล้วกดค้นหา
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// Face Result Card - With multi-select support and PhotoView
interface FaceResultCardProps {
  result: FaceSearchResult
  token: string | null
  isSelected: boolean
  onToggleSelect: () => void
}

function FaceResultCard({ result, token, isSelected, onToggleSelect }: FaceResultCardProps) {
  const [imageError, setImageError] = useState(false)

  // Use proxied thumbnail URL with authentication
  const thumbnailUrl = token && result.drive_file_id
    ? getThumbnailUrl(result.drive_file_id, token)
    : result.thumbnail_url

  // Full size image URL for lightbox
  const fullSizeUrl = token && result.drive_file_id
    ? getThumbnailUrl(result.drive_file_id, token, 1600)
    : result.thumbnail_url

  // Extract folder name from path
  const folderName = result.folder_path
    ? result.folder_path.split('/').pop() || result.folder_path
    : null

  return (
    <div className="group relative rounded-lg transition-all">
      {/* PhotoView wraps the clickable area */}
      <PhotoView src={fullSizeUrl}>
        <div
          className={cn(
            "relative aspect-square bg-muted rounded-lg overflow-hidden cursor-pointer transition-all",
            isSelected && "ring-1 ring-green-500"
          )}
        >
          {imageError ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <ImageOff className="h-6 w-6 text-muted-foreground/50" />
            </div>
          ) : (
            <>
              <img
                src={thumbnailUrl}
                alt={result.file_name}
                className="w-full h-full object-cover transition-transform group-hover:scale-105"
                onError={() => setImageError(true)}
                loading="lazy"
              />
              {/* Face bounding box */}
              <div
                className="absolute border-2 border-green-500/80 rounded transition-all group-hover:border-green-500"
                style={{
                  left: `${result.bbox_x * 100}%`,
                  top: `${result.bbox_y * 100}%`,
                  width: `${result.bbox_width * 100}%`,
                  height: `${result.bbox_height * 100}%`,
                }}
              />
            </>
          )}

          {/* Zoom indicator on hover */}
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center pointer-events-none">
            <ZoomIn className="h-8 w-8 text-white opacity-0 group-hover:opacity-70 transition-opacity" />
          </div>

          {/* Similarity badge */}
          <Badge
            className="absolute bottom-1.5 left-1.5 text-xs pointer-events-none"
            variant={
              result.similarity >= 0.8
                ? "success"
                : result.similarity >= 0.6
                ? "warning"
                : "secondary"
            }
          >
            {(result.similarity * 100).toFixed(0)}%
          </Badge>
        </div>
      </PhotoView>

      {/* Selection checkbox - top right, always visible */}
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

      {/* File name */}
      <p className="mt-1.5 text-xs truncate text-muted-foreground" title={result.file_name}>
        {result.file_name}
      </p>

      {/* Folder name - links to gallery */}
      {folderName && result.shared_folder_id && (
        <Link
          to={`/gallery?folder=${result.shared_folder_id}&name=${encodeURIComponent(folderName)}`}
          className="flex items-center gap-1 text-xs text-muted-foreground/70 hover:text-primary transition-colors truncate"
          onClick={(e) => e.stopPropagation()}
          title={`ดูรูปใน ${result.folder_path}`}
        >
          <FolderOpen className="h-3 w-3 shrink-0" />
          <span className="truncate">{folderName}</span>
        </Link>
      )}
      {folderName && !result.shared_folder_id && (
        <p
          className="flex items-center gap-1 text-xs text-muted-foreground/70 truncate"
          title={result.folder_path}
        >
          <FolderOpen className="h-3 w-3 shrink-0" />
          <span className="truncate">{folderName}</span>
        </p>
      )}
    </div>
  )
}
