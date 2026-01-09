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
  Filter,
} from "lucide-react"
import { format } from "date-fns"
import { th } from "date-fns/locale"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"

import { useSharedFolders } from "@/features/folders"
import { useActivityLogs, useActivityTypes } from "@/features/activity-logs"
import type { ActivityLog } from "@/services/activity-logs"

// Activity type icons and styles
const ACTIVITY_CONFIG: Record<
  string,
  { icon: typeof Activity; className: string; bgClassName: string }
> = {
  // Sync
  sync_started: {
    icon: RefreshCw,
    className: "text-blue-500",
    bgClassName: "bg-blue-50 dark:bg-blue-950",
  },
  sync_completed: {
    icon: CheckCircle2,
    className: "text-green-500",
    bgClassName: "bg-green-50 dark:bg-green-950",
  },
  sync_failed: {
    icon: XCircle,
    className: "text-red-500",
    bgClassName: "bg-red-50 dark:bg-red-950",
  },
  // Photos
  photos_added: {
    icon: Plus,
    className: "text-emerald-500",
    bgClassName: "bg-emerald-50 dark:bg-emerald-950",
  },
  photos_trashed: {
    icon: Trash2,
    className: "text-orange-500",
    bgClassName: "bg-orange-50 dark:bg-orange-950",
  },
  photos_restored: {
    icon: RotateCcw,
    className: "text-cyan-500",
    bgClassName: "bg-cyan-50 dark:bg-cyan-950",
  },
  photos_deleted: {
    icon: XCircle,
    className: "text-red-600",
    bgClassName: "bg-red-50 dark:bg-red-950",
  },
  // Folders
  folder_trashed: {
    icon: Trash2,
    className: "text-orange-600",
    bgClassName: "bg-orange-50 dark:bg-orange-950",
  },
  folder_restored: {
    icon: RotateCcw,
    className: "text-cyan-600",
    bgClassName: "bg-cyan-50 dark:bg-cyan-950",
  },
  folder_renamed: {
    icon: FolderOpen,
    className: "text-purple-500",
    bgClassName: "bg-purple-50 dark:bg-purple-950",
  },
  folder_deleted: {
    icon: XCircle,
    className: "text-red-700",
    bgClassName: "bg-red-50 dark:bg-red-950",
  },
  // Webhook
  webhook_received: {
    icon: Bell,
    className: "text-indigo-500",
    bgClassName: "bg-indigo-50 dark:bg-indigo-950",
  },
  webhook_renewed: {
    icon: RefreshCw,
    className: "text-indigo-600",
    bgClassName: "bg-indigo-50 dark:bg-indigo-950",
  },
  webhook_expired: {
    icon: AlertTriangle,
    className: "text-yellow-600",
    bgClassName: "bg-yellow-50 dark:bg-yellow-950",
  },
  // Errors
  token_expired: {
    icon: AlertCircle,
    className: "text-red-500",
    bgClassName: "bg-red-50 dark:bg-red-950",
  },
  sync_error: {
    icon: AlertCircle,
    className: "text-red-600",
    bgClassName: "bg-red-50 dark:bg-red-950",
  },
}

// Default config for unknown types
const DEFAULT_CONFIG = {
  icon: Activity,
  className: "text-gray-500",
  bgClassName: "bg-gray-50 dark:bg-gray-950",
}

function ActivityIcon({ type }: { type: string }) {
  const config = ACTIVITY_CONFIG[type] || DEFAULT_CONFIG
  const Icon = config.icon
  return (
    <div className={`p-2 rounded-lg ${config.bgClassName}`}>
      <Icon className={`h-4 w-4 ${config.className}`} />
    </div>
  )
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
        <Button variant="ghost" size="sm" className="gap-1 h-7 px-2">
          <Eye className="h-3 w-3" />
          Raw Data
          <ChevronDown
            className={`h-3 w-3 transition-transform ${isOpen ? "rotate-180" : ""}`}
          />
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent className="mt-2">
        <pre className="text-xs bg-muted p-3 rounded-lg overflow-auto max-h-60">
          {JSON.stringify(data, null, 2)}
        </pre>
      </CollapsibleContent>
    </Collapsible>
  )
}

// Activity Log Item Component
function ActivityLogItem({ log }: { log: ActivityLog }) {
  const hasRawData = log.rawData && Object.keys(log.rawData).length > 0

  return (
    <div className="flex gap-3 py-3 border-b last:border-b-0">
      <ActivityIcon type={log.activityType} />
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <p className="text-sm font-medium">{log.message}</p>
          <p className="text-xs text-muted-foreground whitespace-nowrap flex items-center gap-1">
            <Clock className="h-3 w-3" />
            {format(new Date(log.createdAt), "d MMM yyyy HH:mm", { locale: th })}
          </p>
        </div>

        {/* Details summary */}
        {log.details && (
          <div className="flex flex-wrap gap-2 mt-1.5">
            {log.details.total_new !== undefined && log.details.total_new > 0 && (
              <Badge variant="secondary" className="text-xs">
                +{log.details.total_new} ใหม่
              </Badge>
            )}
            {log.details.total_updated !== undefined &&
              log.details.total_updated > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {log.details.total_updated} อัพเดท
                </Badge>
              )}
            {log.details.total_deleted !== undefined &&
              log.details.total_deleted > 0 && (
                <Badge variant="secondary" className="text-xs">
                  -{log.details.total_deleted} ลบ
                </Badge>
              )}
            {log.details.total_trashed !== undefined &&
              log.details.total_trashed > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {log.details.total_trashed} ถังขยะ
                </Badge>
              )}
            {log.details.total_restored !== undefined &&
              log.details.total_restored > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {log.details.total_restored} กู้คืน
                </Badge>
              )}
            {log.details.duration_ms !== undefined && (
              <Badge variant="outline" className="text-xs">
                {(log.details.duration_ms / 1000).toFixed(1)}s
              </Badge>
            )}
            {log.details.error_message && (
              <Badge variant="destructive" className="text-xs">
                {log.details.error_message.substring(0, 50)}
              </Badge>
            )}
          </div>
        )}

        {/* Expandable details/raw data */}
        {hasRawData && (
          <div className="mt-2">
            <RawDataViewer data={log.rawData} />
          </div>
        )}
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
  const folders = useMemo(
    () => foldersData?.folders ?? [],
    [foldersData]
  )

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
    <div className="container mx-auto py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Activity className="h-6 w-6" />
            Activity Logs
          </h1>
          <p className="text-muted-foreground mt-1">
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

      {/* Filters */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Filter className="h-4 w-4" />
            ตัวกรอง
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4">
            {/* Folder selector */}
            <div className="flex-1 min-w-[200px]">
              <label className="text-xs text-muted-foreground mb-1.5 block">
                โฟลเดอร์
              </label>
              <Select
                value={selectedFolderId}
                onValueChange={handleFolderChange}
                disabled={foldersLoading}
              >
                <SelectTrigger>
                  <SelectValue placeholder="เลือกโฟลเดอร์" />
                </SelectTrigger>
                <SelectContent>
                  {folders.map((folder) => (
                    <SelectItem key={folder.id} value={folder.id}>
                      <span className="flex items-center gap-2">
                        <FolderOpen className="h-4 w-4 text-muted-foreground" />
                        {folder.drive_folder_name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Activity type selector */}
            <div className="flex-1 min-w-[200px]">
              <label className="text-xs text-muted-foreground mb-1.5 block">
                ประเภทกิจกรรม
              </label>
              <Select
                value={selectedType || "all"}
                onValueChange={handleTypeChange}
              >
                <SelectTrigger>
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
          </div>
        </CardContent>
      </Card>

      {/* Activity List */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium">
              {selectedFolder
                ? `กิจกรรมของ ${selectedFolder.drive_folder_name}`
                : "เลือกโฟลเดอร์เพื่อดูกิจกรรม"}
            </CardTitle>
            {logsData?.meta && (
              <span className="text-xs text-muted-foreground">
                {logsData.meta.total.toLocaleString()} รายการ
              </span>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {!selectedFolderId ? (
            <div className="text-center py-12 text-muted-foreground">
              <FolderOpen className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>กรุณาเลือกโฟลเดอร์เพื่อดูกิจกรรม</p>
            </div>
          ) : logsLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : logsError ? (
            <div className="text-center py-12 text-destructive">
              <AlertCircle className="h-12 w-12 mx-auto mb-3" />
              <p>เกิดข้อผิดพลาดในการโหลดกิจกรรม</p>
            </div>
          ) : !logsData?.data?.length ? (
            <div className="text-center py-12 text-muted-foreground">
              <Activity className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>ไม่พบกิจกรรม</p>
            </div>
          ) : (
            <div className="divide-y">
              {logsData.data.map((log) => (
                <ActivityLogItem key={log.id} log={log} />
              ))}
            </div>
          )}
        </CardContent>

        {/* Pagination */}
        {logsData?.meta && logsData.meta.totalPages > 1 && (
          <div className="flex items-center justify-between px-6 py-4 border-t">
            <p className="text-sm text-muted-foreground">
              หน้า {logsData.meta.page} จาก {logsData.meta.totalPages}
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page - 1)}
                disabled={!logsData.meta.hasPrev}
              >
                <ChevronLeft className="h-4 w-4" />
                ก่อนหน้า
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handlePageChange(page + 1)}
                disabled={!logsData.meta.hasNext}
              >
                ถัดไป
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}
