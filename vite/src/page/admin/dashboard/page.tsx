import { useNavigate } from "react-router-dom"
import {
  LayoutDashboard,
  Images,
  Users,
  ScanFace,
  Cloud,
  CheckCircle2,
  Clock,
  AlertCircle,
  Loader2,
  ArrowRight,
  RefreshCw,
  Newspaper,
  Settings,
  Search,
  FolderOpen,
  AlertTriangle,
  FolderPlus,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

import { useDriveStatus, useSyncStatus, useStartSync } from "@/features/drive"
import { useSharedFolders } from "@/features/folders"
import { useFaceStats } from "@/features/face-search"

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

export default function DashboardPage() {
  const navigate = useNavigate()

  // Hooks
  const { data: driveStatus, isLoading: driveLoading } = useDriveStatus()
  const { data: stats, isLoading: statsLoading } = useFaceStats()
  const { data: syncStatus } = useSyncStatus(driveStatus?.connected)
  const { data: foldersData, isLoading: foldersLoading } = useSharedFolders(driveStatus?.connected)
  const startSyncMutation = useStartSync()

  // Computed
  const isConnected = driveStatus?.connected ?? false
  const hasFolders = (foldersData?.folders?.length ?? 0) > 0
  const isReady = isConnected && hasFolders
  const isSyncing = syncStatus?.status === "running" || syncStatus?.status === "pending"
  const processingProgress = stats
    ? ((stats.processed_photos / Math.max(stats.total_photos, 1)) * 100)
    : 0

  const quickLinks = [
    {
      label: "คลังรูปภาพ",
      icon: Images,
      href: "/gallery",
      description: "ดูและจัดการรูปภาพ",
    },
    {
      label: "ค้นหาใบหน้า",
      icon: Search,
      href: "/face-search",
      description: "ค้นหาด้วย AI",
    },
    {
      label: "เขียนข่าว AI",
      icon: Newspaper,
      href: "/news-writer",
      description: "สร้างข่าวประชาสัมพันธ์",
    },
    {
      label: "ตั้งค่า",
      icon: Settings,
      href: "/settings",
      description: "จัดการระบบ",
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">แดชบอร์ด</h1>
          <p className="text-sm text-muted-foreground">
            ภาพรวมระบบ KU Directory
          </p>
        </div>
        {isReady && (
          <Button
            size="sm"
            onClick={() => startSyncMutation.mutate()}
            disabled={isSyncing || startSyncMutation.isPending || !driveStatus?.rootFolder}
          >
            {isSyncing ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                กำลัง Sync...
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4 mr-2" />
                Sync
              </>
            )}
          </Button>
        )}
      </div>

      {/* Alert: Not connected to Google Drive */}
      {!driveLoading && !isConnected && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>ยังไม่ได้เชื่อมต่อ Google Drive</AlertTitle>
          <AlertDescription className="flex flex-col gap-2">
            <span>กรุณาเชื่อมต่อ Google Drive เพื่อเริ่มใช้งานระบบ</span>
            <Button
              variant="outline"
              size="sm"
              className="w-fit text-foreground"
              onClick={() => navigate("/settings")}
            >
              <Settings className="h-3.5 w-3.5 mr-1.5" />
              ไปที่ตั้งค่า
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Alert: No folders added */}
      {!driveLoading && !foldersLoading && isConnected && !hasFolders && (
        <Alert>
          <FolderPlus className="h-4 w-4" />
          <AlertTitle>ยังไม่มีโฟลเดอร์</AlertTitle>
          <AlertDescription className="flex flex-col gap-2">
            <span>กรุณาเพิ่มโฟลเดอร์ที่ต้องการ Sync จาก Google Drive</span>
            <Button
              variant="outline"
              size="sm"
              className="w-fit"
              onClick={() => navigate("/settings")}
            >
              <FolderOpen className="h-3.5 w-3.5 mr-1.5" />
              เพิ่มโฟลเดอร์
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Key Metrics */}
      <div className={`grid grid-cols-2 lg:grid-cols-5 gap-4 ${!isReady && !driveLoading && !foldersLoading ? 'opacity-50' : ''}`}>
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
          label="Google Drive"
          value={isConnected ? "เชื่อมต่อแล้ว" : "ยังไม่เชื่อมต่อ"}
          icon={<Cloud className="h-3.5 w-3.5" />}
          isLoading={driveLoading}
        />
      </div>

      {/* Processing Progress */}
      {stats && stats.pending_photos > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">กำลังประมวลผลใบหน้า...</span>
            <span>{processingProgress.toFixed(0)}%</span>
          </div>
          <Progress value={processingProgress} className="h-1.5" />
        </div>
      )}

      <Separator />

      {/* Main Content */}
      <div className={`grid grid-cols-12 gap-6 ${!isReady && !driveLoading && !foldersLoading ? 'opacity-50 pointer-events-none' : ''}`}>
        {/* Sidebar - Quick Links */}
        <div className="col-span-12 lg:col-span-2">
          <div className="space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <LayoutDashboard className="h-3.5 w-3.5" />
              เมนูลัด
            </h3>

            <div className="space-y-1">
              {quickLinks.map((link) => (
                <button
                  key={link.href}
                  onClick={() => navigate(link.href)}
                  className="w-full flex items-center gap-2 py-2 px-3 rounded-lg text-sm hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-colors text-left"
                >
                  <link.icon className="h-4 w-4" />
                  {link.label}
                </button>
              ))}
            </div>
          </div>

          <Separator className="my-4" />

          {/* Drive Status */}
          <div className="space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <Cloud className="h-3.5 w-3.5" />
              Google Drive
            </h3>

            {driveLoading ? (
              <div className="h-16 bg-muted rounded-lg animate-pulse" />
            ) : isConnected ? (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-3.5 w-3.5 text-primary" />
                  <span className="text-xs text-primary">เชื่อมต่อแล้ว</span>
                </div>
                {driveStatus?.rootFolder && (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <FolderOpen className="h-3.5 w-3.5" />
                    <span className="truncate">{driveStatus.rootFolder.name}</span>
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <AlertCircle className="h-3.5 w-3.5 text-muted-foreground" />
                  <span className="text-xs text-muted-foreground">ยังไม่ได้เชื่อมต่อ</span>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full text-xs h-7"
                  onClick={() => navigate("/settings")}
                >
                  ไปตั้งค่า
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Main Area */}
        <div className="col-span-12 lg:col-span-10">
          {!isConnected ? (
            // Not connected state
            <div className="py-12 text-center">
              <Cloud className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                ยังไม่ได้เชื่อมต่อ Google Drive
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                เชื่อมต่อเพื่อเริ่มใช้งานระบบ
              </p>
              <Button className="mt-4" size="sm" onClick={() => navigate("/settings")}>
                ไปที่ตั้งค่า
                <ArrowRight className="h-3 w-3 ml-1" />
              </Button>
            </div>
          ) : (
            // Connected - Show feature cards
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {/* Gallery Card */}
              <div
                className="group rounded-lg border p-4 hover:border-primary/50 hover:bg-muted/30 transition-colors cursor-pointer"
                onClick={() => navigate("/gallery")}
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                    <Images className="h-5 w-5 text-foreground" />
                  </div>
                  <div>
                    <h3 className="font-medium">คลังรูปภาพ</h3>
                    <p className="text-xs text-muted-foreground">
                      {stats?.total_photos || 0} รูป
                    </p>
                  </div>
                </div>
                <p className="text-sm text-muted-foreground">
                  ดูและจัดการรูปภาพจาก Google Drive
                </p>
                <div className="mt-3 flex items-center text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                  เปิดคลังรูปภาพ
                  <ArrowRight className="h-3 w-3 ml-1" />
                </div>
              </div>

              {/* Face Search Card */}
              <div
                className="group rounded-lg border p-4 hover:border-primary/50 hover:bg-muted/30 transition-colors cursor-pointer"
                onClick={() => navigate("/face-search")}
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                    <ScanFace className="h-5 w-5 text-foreground" />
                  </div>
                  <div>
                    <h3 className="font-medium">ค้นหาใบหน้า</h3>
                    <p className="text-xs text-muted-foreground">
                      {stats?.total_faces || 0} ใบหน้า
                    </p>
                  </div>
                </div>
                <p className="text-sm text-muted-foreground">
                  ค้นหารูปภาพจากใบหน้าด้วย AI
                </p>
                <div className="mt-3 flex items-center text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                  เริ่มค้นหา
                  <ArrowRight className="h-3 w-3 ml-1" />
                </div>
              </div>

              {/* News Writer Card */}
              <div
                className="group rounded-lg border p-4 hover:border-primary/50 hover:bg-muted/30 transition-colors cursor-pointer"
                onClick={() => navigate("/news-writer")}
              >
                <div className="flex items-center gap-3 mb-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                    <Newspaper className="h-5 w-5 text-foreground" />
                  </div>
                  <div>
                    <h3 className="font-medium">เขียนข่าว AI</h3>
                    <p className="text-xs text-muted-foreground">Gemini AI</p>
                  </div>
                </div>
                <p className="text-sm text-muted-foreground">
                  สร้างข่าวประชาสัมพันธ์จากรูปภาพ
                </p>
                <div className="mt-3 flex items-center text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                  เริ่มเขียนข่าว
                  <ArrowRight className="h-3 w-3 ml-1" />
                </div>
              </div>

              {/* Sync Status Card */}
              {syncStatus && (
                <div className="rounded-lg border p-4 md:col-span-2 lg:col-span-3">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <RefreshCw className={`h-5 w-5 ${isSyncing ? "animate-spin text-primary" : "text-muted-foreground"}`} />
                      <div>
                        <h3 className="font-medium">Sync Status</h3>
                        <p className="text-xs text-muted-foreground">
                          อัพเดทล่าสุด: {syncStatus.processed_files} ไฟล์
                        </p>
                      </div>
                    </div>
                    <Badge
                      variant={
                        syncStatus.status === "completed"
                          ? "secondary"
                          : syncStatus.status === "failed"
                          ? "outline"
                          : "default"
                      }
                    >
                      {syncStatus.status === "running"
                        ? "กำลัง Sync"
                        : syncStatus.status === "pending"
                        ? "รอดำเนินการ"
                        : syncStatus.status === "completed"
                        ? "เสร็จสิ้น"
                        : "ล้มเหลว"}
                    </Badge>
                  </div>
                  {isSyncing && (
                    <Progress
                      value={Math.round((syncStatus.processed_files / Math.max(syncStatus.total_files, 1)) * 100)}
                      className="h-2"
                    />
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
