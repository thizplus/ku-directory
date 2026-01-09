import { useState, useMemo } from "react"
import { useSearchParams } from "react-router-dom"
import {
  Activity,
  Loader2,
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Clock,
  CheckCircle2,
  XCircle,
  Plus,
  Trash2,
  FolderOpen,
  RotateCcw,
  RefreshCw,
  Bell,
  AlertTriangle,
  Eye,
  ChevronDown,
} from "lucide-react"
import { format } from "date-fns"
import { th } from "date-fns/locale"

import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"

import { useSharedFolders } from "@/features/folders"
import { useActivityLogs, useActivityTypes } from "@/features/activity-logs"
import type { ActivityLog } from "@/services/activity-logs"

// Activity type icons and styles - using semantic CSS classes from index.css
const ACTIVITY_CONFIG: Record<
  string,
  { icon: typeof Activity; className: string }
> = {
  // Sync
  sync_started: { icon: RefreshCw, className: "activity-sync" },
  sync_completed: { icon: CheckCircle2, className: "activity-success" },
  sync_failed: { icon: XCircle, className: "activity-error" },
  // Photos
  photos_added: { icon: Plus, className: "activity-add" },
  photos_trashed: { icon: Trash2, className: "activity-trash" },
  photos_restored: { icon: RotateCcw, className: "activity-restore" },
  photos_deleted: { icon: XCircle, className: "activity-error" },
  photo_renamed: { icon: Activity, className: "activity-folder" },
  photo_moved: { icon: Activity, className: "activity-move" },
  photo_updated: { icon: RefreshCw, className: "activity-sync" },
  // Folders
  folder_trashed: { icon: Trash2, className: "activity-trash" },
  folder_restored: { icon: RotateCcw, className: "activity-restore" },
  folder_renamed: { icon: FolderOpen, className: "activity-folder" },
  folder_moved: { icon: FolderOpen, className: "activity-move" },
  folder_deleted: { icon: XCircle, className: "activity-error" },
  // Webhook
  webhook_received: { icon: Bell, className: "activity-webhook" },
  webhook_renewed: { icon: RefreshCw, className: "activity-webhook" },
  webhook_expired: { icon: AlertTriangle, className: "activity-warning" },
  // Errors
  token_expired: { icon: AlertCircle, className: "activity-error" },
  sync_error: { icon: AlertCircle, className: "activity-error" },
}

const DEFAULT_CONFIG = { icon: Activity, className: "text-muted-foreground" }

function ActivityIcon({ type }: { type: string }) {
  const config = ACTIVITY_CONFIG[type] || DEFAULT_CONFIG
  const Icon = config.icon
  return <Icon className={`h-4 w-4 ${config.className}`} />
}

// Raw Data Viewer Component
function RawDataViewer({ data }: { data: Record<string, unknown> | undefined }) {
  const [isOpen, setIsOpen] = useState(false)

  if (!data || Object.keys(data).length === 0) {
    return null
  }

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <Button variant="ghost" size="sm" className="gap-1 h-6 px-2 text-xs">
          <Eye className="h-3 w-3" />
          Raw
          <ChevronDown
            className={`h-3 w-3 transition-transform ${isOpen ? "rotate-180" : ""}`}
          />
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent className="mt-2">
        <pre className="text-xs bg-muted p-3 rounded-lg overflow-auto max-h-48">
          {JSON.stringify(data, null, 2)}
        </pre>
      </CollapsibleContent>
    </Collapsible>
  )
}

// Activity Log Row Component
function ActivityLogRow({ log }: { log: ActivityLog }) {
  const hasRawData = log.rawData && Object.keys(log.rawData).length > 0

  return (
    <div className="py-3 px-4 hover:bg-muted/30 transition-colors">
      <div className="flex items-start gap-3">
        {/* Icon */}
        <div className="mt-0.5">
          <ActivityIcon type={log.activityType} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex items-start justify-between gap-4">
            <p className="text-sm">{log.message}</p>
            <span className="text-xs text-muted-foreground whitespace-nowrap flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {format(new Date(log.createdAt), "d MMM HH:mm", { locale: th })}
            </span>
          </div>

          {/* Details badges */}
          {log.details && (
            <div className="flex flex-wrap items-center gap-1.5">
              {log.details.total_new !== undefined && log.details.total_new > 0 && (
                <Badge variant="secondary" className="text-xs h-5 px-1.5">
                  +{log.details.total_new} ใหม่
                </Badge>
              )}
              {log.details.total_updated !== undefined && log.details.total_updated > 0 && (
                <Badge variant="secondary" className="text-xs h-5 px-1.5">
                  {log.details.total_updated} อัพเดท
                </Badge>
              )}
              {log.details.total_deleted !== undefined && log.details.total_deleted > 0 && (
                <Badge variant="secondary" className="text-xs h-5 px-1.5">
                  -{log.details.total_deleted} ลบ
                </Badge>
              )}
              {log.details.total_trashed !== undefined && log.details.total_trashed > 0 && (
                <Badge variant="secondary" className="text-xs h-5 px-1.5">
                  {log.details.total_trashed} ถังขยะ
                </Badge>
              )}
              {log.details.total_restored !== undefined && log.details.total_restored > 0 && (
                <Badge variant="secondary" className="text-xs h-5 px-1.5">
                  {log.details.total_restored} กู้คืน
                </Badge>
              )}
              {log.details.duration_ms !== undefined && (
                <Badge variant="outline" className="text-xs h-5 px-1.5">
                  {(log.details.duration_ms / 1000).toFixed(1)}s
                </Badge>
              )}
              {log.details.error_message && (
                <Badge variant="destructive" className="text-xs h-5 px-1.5">
                  {log.details.error_message.substring(0, 40)}...
                </Badge>
              )}
              {hasRawData && <RawDataViewer data={log.rawData} />}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function ActivityLogsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedFolderId = searchParams.get("folderId") || ""
  const selectedType = searchParams.get("type") || ""
  const page = parseInt(searchParams.get("page") || "1")
  const limit = 25

  // Fetch folders for dropdown
  const { data: foldersData, isLoading: foldersLoading } = useSharedFolders()
  const folders = useMemo(() => foldersData?.folders ?? [], [foldersData])

  // Fetch activity types
  const { data: activityTypes } = useActivityTypes()

  // Fetch activity logs
  const {
    data: logsData,
    isLoading: logsLoading,
    error: logsError,
    refetch,
  } = useActivityLogs(
    {
      folderId: selectedFolderId,
      page,
      limit,
      activityType: selectedType || undefined,
    },
    !!selectedFolderId
  )

  // Auto-select first folder if none selected
  useMemo(() => {
    if (!selectedFolderId && folders.length > 0) {
      setSearchParams({ folderId: folders[0].id })
    }
  }, [folders, selectedFolderId, setSearchParams])

  // Selected folder info
  const selectedFolder = useMemo(
    () => folders.find((f) => f.id === selectedFolderId),
    [folders, selectedFolderId]
  )

  // Handlers
  const handleFolderChange = (folderId: string) => {
    setSearchParams({ folderId, page: "1" })
  }

  const handleTypeChange = (type: string) => {
    const params: Record<string, string> = { folderId: selectedFolderId, page: "1" }
    if (type && type !== "all") {
      params.type = type
    }
    setSearchParams(params)
  }

  const handlePageChange = (newPage: number) => {
    const params: Record<string, string> = { folderId: selectedFolderId, page: String(newPage) }
    if (selectedType) {
      params.type = selectedType
    }
    setSearchParams(params)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Activity Logs</h1>
          <p className="text-sm text-muted-foreground">
            ติดตามกิจกรรมทั้งหมดที่เกิดขึ้นกับ Google Drive
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={logsLoading}
        >
          {logsLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          <span className="ml-2">รีเฟรช</span>
        </Button>
      </div>

      {/* Filters Row */}
      <div className="flex flex-wrap items-end gap-4">
        <div className="min-w-[200px]">
          <label className="text-xs text-muted-foreground mb-1.5 block">
            โฟลเดอร์
          </label>
          <Select
            value={selectedFolderId}
            onValueChange={handleFolderChange}
            disabled={foldersLoading}
          >
            <SelectTrigger className="h-9">
              <SelectValue placeholder="เลือกโฟลเดอร์" />
            </SelectTrigger>
            <SelectContent>
              {folders.map((folder) => (
                <SelectItem key={folder.id} value={folder.id}>
                  <span className="flex items-center gap-2">
                    <FolderOpen className="h-3.5 w-3.5 text-muted-foreground" />
                    {folder.drive_folder_name}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="min-w-[180px]">
          <label className="text-xs text-muted-foreground mb-1.5 block">
            ประเภทกิจกรรม
          </label>
          <Select
            value={selectedType || "all"}
            onValueChange={handleTypeChange}
          >
            <SelectTrigger className="h-9">
              <SelectValue placeholder="ทั้งหมด" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">ทั้งหมด</SelectItem>
              {activityTypes?.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  {type.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {logsData?.meta && (
          <div className="ml-auto text-xs text-muted-foreground">
            {logsData.meta.total.toLocaleString()} รายการ
          </div>
        )}
      </div>

      <Separator />

      {/* Activity List */}
      <div className="rounded-lg border">
        {/* List Header */}
        <div className="flex items-center justify-between px-4 py-2 bg-muted/30 border-b">
          <span className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
            <Activity className="h-3.5 w-3.5" />
            {selectedFolder
              ? `กิจกรรมของ ${selectedFolder.drive_folder_name}`
              : "เลือกโฟลเดอร์เพื่อดูกิจกรรม"}
          </span>
        </div>

        {/* List Content */}
        {!selectedFolderId ? (
          <div className="text-center py-12 text-muted-foreground">
            <FolderOpen className="h-10 w-10 mx-auto mb-2 opacity-50" />
            <p className="text-sm">กรุณาเลือกโฟลเดอร์เพื่อดูกิจกรรม</p>
          </div>
        ) : logsLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : logsError ? (
          <div className="text-center py-12 text-destructive">
            <AlertCircle className="h-10 w-10 mx-auto mb-2" />
            <p className="text-sm">เกิดข้อผิดพลาดในการโหลดกิจกรรม</p>
          </div>
        ) : !logsData?.data?.length ? (
          <div className="text-center py-12 text-muted-foreground">
            <Activity className="h-10 w-10 mx-auto mb-2 opacity-50" />
            <p className="text-sm">ไม่พบกิจกรรม</p>
          </div>
        ) : (
          <div className="divide-y">
            {logsData.data.map((log) => (
              <ActivityLogRow key={log.id} log={log} />
            ))}
          </div>
        )}

        {/* Pagination */}
        {logsData?.meta && logsData.meta.totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t bg-muted/20">
            <span className="text-xs text-muted-foreground">
              หน้า {logsData.meta.page} จาก {logsData.meta.totalPages}
            </span>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                className="h-7"
                onClick={() => handlePageChange(page - 1)}
                disabled={!logsData.meta.hasPrev}
              >
                <ChevronLeft className="h-4 w-4" />
                ก่อนหน้า
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="h-7"
                onClick={() => handlePageChange(page + 1)}
                disabled={!logsData.meta.hasNext}
              >
                ถัดไป
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
