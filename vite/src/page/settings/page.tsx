import { useEffect, useState } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Cloud,
  Key,
  Loader2,
  FolderOpen,
  RefreshCw,
  ChevronRight,
  ArrowLeft,
  Folder,
  AlertCircle,
  Save,
  Sparkles,
  X,
  Link as LinkIcon,
  Plus,
  Radio,
  Clock,
  AlertTriangle,
  WifiOff,
  Settings,
  Activity,
  ExternalLink,
  Trash2,
  CheckCircle2,
  XCircle,
  Unlink,
} from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

// Feature imports
import {
  useDriveStatus,
  useConnectDrive,
  useDisconnectDrive,
  useDriveFolders,
} from "@/features/drive"
import {
  useSharedFolders,
  useAddFolder,
  useRemoveFolder,
  useTriggerFolderSync,
} from "@/features/folders"
import { userService, type GeminiSettings } from "@/services/user"
import type { DriveFolder, SharedFolder } from "@/shared/types"

// Helper function to get webhook status display info
function getWebhookStatusInfo(status: SharedFolder['webhook_status'], expiry: string | null) {
  const expiryDate = expiry ? new Date(expiry) : null
  const expiryText = expiryDate
    ? `หมดอายุ ${expiryDate.toLocaleDateString('th-TH', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}`
    : ''

  switch (status) {
    case 'active':
      return {
        icon: Radio,
        label: 'Webhook ทำงานปกติ',
        description: expiryText,
        className: 'text-foreground',
      }
    case 'expiring':
      return {
        icon: Clock,
        label: 'Webhook ใกล้หมดอายุ',
        description: expiryText,
        className: 'text-muted-foreground',
      }
    case 'expired':
      return {
        icon: AlertTriangle,
        label: 'Webhook หมดอายุ',
        description: 'รอ auto-renewal',
        className: 'text-destructive',
      }
    case 'inactive':
    default:
      return {
        icon: WifiOff,
        label: 'Webhook ไม่ทำงาน',
        description: 'กด Sync เพื่อลงทะเบียน',
        className: 'text-muted-foreground',
      }
  }
}

// Sync status badge component
function SyncStatusBadge({ status }: { status: string }) {
  switch (status) {
    case 'syncing':
      return (
        <Badge variant="secondary" className="gap-1">
          <Loader2 className="h-3 w-3 animate-spin" />
          กำลัง Sync
        </Badge>
      )
    case 'completed':
      return (
        <Badge variant="outline" className="gap-1">
          <CheckCircle2 className="h-3 w-3" />
          Sync แล้ว
        </Badge>
      )
    case 'failed':
      return (
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Sync ล้มเหลว
        </Badge>
      )
    default:
      return (
        <Badge variant="outline" className="gap-1">
          <Clock className="h-3 w-3" />
          รอ Sync
        </Badge>
      )
  }
}

export default function SettingsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [folderDialogOpen, setFolderDialogOpen] = useState(false)
  const [currentFolderId, setCurrentFolderId] = useState<string | undefined>()
  const [folderPath, setFolderPath] = useState<DriveFolder[]>([])

  // Gemini settings state
  const [geminiApiKey, setGeminiApiKey] = useState("")
  const [geminiModel, setGeminiModel] = useState("gemini-2.0-flash")
  const [disconnectDialogOpen, setDisconnectDialogOpen] = useState(false)
  const [folderFilter, setFolderFilter] = useState("")
  const [folderPickerTab, setFolderPickerTab] = useState<"url" | "select">("select")
  const [folderUrlInput, setFolderUrlInput] = useState("")
  const queryClient = useQueryClient()

  // User profile query
  const { data: userProfile, isLoading: profileLoading } = useQuery({
    queryKey: ['user-profile'],
    queryFn: () => userService.getProfile(),
  })

  // Update Gemini settings mutation
  const updateGeminiMutation = useMutation({
    mutationFn: (settings: GeminiSettings) => userService.updateGeminiSettings(settings),
    onSuccess: () => {
      toast.success("บันทึกการตั้งค่า Gemini สำเร็จ")
      queryClient.invalidateQueries({ queryKey: ['user-profile'] })
    },
    onError: () => {
      toast.error("เกิดข้อผิดพลาดในการบันทึก")
    },
  })

  // Sync Gemini state from profile
  useEffect(() => {
    if (userProfile?.data) {
      if (userProfile.data.geminiModel) {
        setGeminiModel(userProfile.data.geminiModel)
      }
    }
  }, [userProfile])

  const handleSaveGemini = () => {
    updateGeminiMutation.mutate({
      apiKey: geminiApiKey,
      model: geminiModel,
    })
  }

  const hasGeminiKey = userProfile?.data?.geminiApiKey && userProfile.data.geminiApiKey.length > 0

  // Hooks
  const { data: driveStatus, isLoading: statusLoading, refetch: refetchStatus } = useDriveStatus()
  const connectMutation = useConnectDrive()
  const disconnectMutation = useDisconnectDrive()

  // Shared folders
  const { data: sharedFoldersData, isLoading: sharedFoldersLoading } = useSharedFolders(driveStatus?.connected)
  const addFolderMutation = useAddFolder()
  const removeFolderMutation = useRemoveFolder()
  const triggerSyncMutation = useTriggerFolderSync()

  // Folders for the folder picker dialog
  const { data: folders, isLoading: foldersLoading } = useDriveFolders(
    currentFolderId,
    folderDialogOpen && driveStatus?.connected
  )

  useEffect(() => {
    const driveParam = searchParams.get("drive")
    const errorParam = searchParams.get("error")

    if (driveParam === "connected") {
      toast.success("เชื่อมต่อ Google Drive สำเร็จ")
      refetchStatus()
      setSearchParams({})
    } else if (errorParam) {
      toast.error(`เกิดข้อผิดพลาด: ${errorParam}`)
      setSearchParams({})
    }
  }, [searchParams, setSearchParams, refetchStatus])

  const handleConnect = () => connectMutation.mutate()

  const handleDisconnect = async () => {
    try {
      await disconnectMutation.mutateAsync()
      toast.success("ยกเลิกการเชื่อมต่อ Google Drive แล้ว")
      setDisconnectDialogOpen(false)
    } catch {
      toast.error("เกิดข้อผิดพลาด")
    }
  }

  const handleSelectFolder = async (folder: DriveFolder) => {
    try {
      await addFolderMutation.mutateAsync(folder.id)
      setFolderDialogOpen(false)
      resetFolderPicker()
    } catch {
      // Error handled in mutation
    }
  }

  const handleRemoveFolder = async (folderId: string) => {
    try {
      await removeFolderMutation.mutateAsync(folderId)
    } catch {
      // Error handled in mutation
    }
  }

  const handleSyncFolder = async (folderId: string) => {
    try {
      await triggerSyncMutation.mutateAsync({ folderId })
    } catch {
      // Error handled in mutation
    }
  }

  const handleNavigateFolder = (folder: DriveFolder) => {
    setCurrentFolderId(folder.id)
    setFolderPath([...folderPath, folder])
    setFolderFilter("")
  }

  const handleNavigateBack = () => {
    const newPath = [...folderPath]
    newPath.pop()
    setFolderPath(newPath)
    setCurrentFolderId(newPath.length > 0 ? newPath[newPath.length - 1].id : undefined)
    setFolderFilter("")
  }

  const resetFolderPicker = () => {
    setCurrentFolderId(undefined)
    setFolderPath([])
    setFolderFilter("")
    setFolderPickerTab("select")
    setFolderUrlInput("")
  }

  // Extract folder ID from Google Drive URL
  const extractFolderIdFromUrl = (url: string): string | null => {
    const match = url.match(/\/folders\/([a-zA-Z0-9_-]+)/)
    return match ? match[1] : null
  }

  const handleAddFolderFromUrl = () => {
    const folderId = extractFolderIdFromUrl(folderUrlInput)
    if (!folderId) {
      toast.error("URL ไม่ถูกต้อง กรุณาใส่ลิงค์โฟลเดอร์ Google Drive")
      return
    }
    addFolderMutation.mutate(folderId, {
      onSuccess: () => {
        setFolderDialogOpen(false)
        resetFolderPicker()
      },
    })
  }

  const isConnected = driveStatus?.connected ?? false
  const sharedFolders = sharedFoldersData?.folders || []
  const hasSyncingFolder = sharedFolders.some(f => f.sync_status === 'syncing')

  return (
    <TooltipProvider>
      <div className="container mx-auto py-6 space-y-6 max-w-4xl">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Settings className="h-6 w-6" />
              ตั้งค่า
            </h1>
            <p className="text-muted-foreground mt-1">
              จัดการการเชื่อมต่อและตั้งค่าระบบ
            </p>
          </div>
          <Button variant="outline" size="sm" asChild>
            <Link to="/activity-logs" className="gap-2">
              <Activity className="h-4 w-4" />
              ดู Activity Logs
            </Link>
          </Button>
        </div>

        <Separator />

        {/* Google Drive Section */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Cloud className="h-5 w-5 text-muted-foreground" />
                <div>
                  <CardTitle className="text-lg">Google Drive</CardTitle>
                  <CardDescription>เชื่อมต่อและ Sync รูปภาพจาก Google Drive</CardDescription>
                </div>
              </div>
              {statusLoading ? (
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              ) : isConnected ? (
                <Badge variant="outline">
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  เชื่อมต่อแล้ว
                </Badge>
              ) : (
                <Badge variant="secondary">ยังไม่เชื่อมต่อ</Badge>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {isConnected ? (
              <>
                {/* Folders Header */}
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium">
                    โฟลเดอร์ที่ Sync ({sharedFolders.length})
                  </p>
                  <Button size="sm" onClick={() => setFolderDialogOpen(true)} className="gap-2">
                    <Plus className="h-4 w-4" />
                    เพิ่มโฟลเดอร์
                  </Button>
                </div>

                {/* Folders List */}
                {sharedFoldersLoading ? (
                  <div className="space-y-2">
                    {Array.from({ length: 2 }).map((_, i) => (
                      <div key={i} className="h-20 bg-muted rounded-lg animate-pulse" />
                    ))}
                  </div>
                ) : sharedFolders.length > 0 ? (
                  <div className="space-y-3">
                    {sharedFolders.map((folder: SharedFolder) => {
                      const webhookInfo = getWebhookStatusInfo(folder.webhook_status, folder.webhook_expiry)
                      const WebhookIcon = webhookInfo.icon
                      return (
                        <div key={folder.id} className="rounded-lg border p-4 space-y-3 hover:bg-muted/30 transition-colors">
                          {/* Folder Header */}
                          <div className="flex items-start justify-between gap-4">
                            <div className="flex items-center gap-3 min-w-0">
                              <Folder className="h-4 w-4 text-muted-foreground" />
                              <div className="min-w-0">
                                <p className="font-medium truncate">{folder.drive_folder_name}</p>
                                <p className="text-xs text-muted-foreground">{folder.photo_count} รูปภาพ</p>
                              </div>
                            </div>
                            <SyncStatusBadge status={folder.sync_status} />
                          </div>

                          {/* Webhook Status */}
                          <div className="flex items-center gap-2 text-xs px-1">
                            <WebhookIcon className={`h-3.5 w-3.5 ${webhookInfo.className}`} />
                            <span className={webhookInfo.className}>{webhookInfo.label}</span>
                            {webhookInfo.description && (
                              <>
                                <span className="text-muted-foreground">•</span>
                                <span className="text-muted-foreground">{webhookInfo.description}</span>
                              </>
                            )}
                          </div>

                          {/* Actions */}
                          <div className="flex items-center gap-2 pt-1">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleSyncFolder(folder.id)}
                                  disabled={folder.sync_status === 'syncing' || triggerSyncMutation.isPending}
                                  className="gap-2"
                                >
                                  <RefreshCw className={`h-3.5 w-3.5 ${folder.sync_status === 'syncing' ? 'animate-spin' : ''}`} />
                                  Sync ข้อมูล
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>ดึงข้อมูลรูปภาพใหม่จาก Google Drive</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleRemoveFolder(folder.id)}
                                  disabled={removeFolderMutation.isPending}
                                  className="gap-2 text-destructive hover:text-destructive hover:bg-destructive/10"
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                  ลบโฟลเดอร์
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>ลบโฟลเดอร์ออกจากระบบ</TooltipContent>
                            </Tooltip>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <div className="text-center py-8 border rounded-lg bg-muted/30">
                    <FolderOpen className="h-10 w-10 mx-auto text-muted-foreground mb-2" />
                    <p className="text-sm text-muted-foreground">ยังไม่มีโฟลเดอร์</p>
                    <p className="text-xs text-muted-foreground mt-1">กด "เพิ่มโฟลเดอร์" เพื่อเริ่มต้น Sync รูปภาพ</p>
                  </div>
                )}

                <Separator />

                {/* Disconnect Button */}
                <div className="flex justify-end">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setDisconnectDialogOpen(true)}
                    disabled={disconnectMutation.isPending || hasSyncingFolder}
                    className="gap-2 text-muted-foreground hover:text-destructive"
                  >
                    <Unlink className="h-4 w-4" />
                    ยกเลิกการเชื่อมต่อ Google Drive
                  </Button>
                </div>
              </>
            ) : (
              <div className="text-center py-8">
                <Cloud className="h-12 w-12 mx-auto text-muted-foreground mb-3" />
                <p className="text-sm text-muted-foreground mb-4">
                  เชื่อมต่อ Google Drive เพื่อ Sync รูปภาพเข้าสู่ระบบ
                </p>
                <Button onClick={handleConnect} disabled={connectMutation.isPending} className="gap-2">
                  {connectMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Cloud className="h-4 w-4" />
                  )}
                  เชื่อมต่อ Google Drive
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Gemini AI Section */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Sparkles className="h-5 w-5 text-muted-foreground" />
                <div>
                  <CardTitle className="text-lg">Gemini AI</CardTitle>
                  <CardDescription>ตั้งค่า API สำหรับฟีเจอร์เขียนข่าว AI</CardDescription>
                </div>
              </div>
              {profileLoading ? (
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              ) : hasGeminiKey ? (
                <Badge variant="outline">
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  ตั้งค่าแล้ว
                </Badge>
              ) : (
                <Badge variant="secondary">ยังไม่ได้ตั้งค่า</Badge>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Current API Key Status */}
            {hasGeminiKey && (
              <div className="flex items-center gap-2 text-sm p-3 bg-muted/50 rounded-lg">
                <Key className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">API Key ปัจจุบัน:</span>
                <code className="text-xs bg-background px-2 py-0.5 rounded">{userProfile?.data?.geminiApiKey}</code>
              </div>
            )}

            {/* API Key Input */}
            <div className="space-y-2">
              <label className="text-sm font-medium">API Key</label>
              <Input
                type="password"
                placeholder={hasGeminiKey ? "ใส่ API Key ใหม่เพื่อเปลี่ยน..." : "ใส่ Gemini API Key ของคุณ"}
                value={geminiApiKey}
                onChange={(e) => setGeminiApiKey(e.target.value)}
              />
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <ExternalLink className="h-3 w-3" />
                รับ API Key ได้ที่{" "}
                <a
                  href="https://aistudio.google.com/app/apikey"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary underline"
                >
                  Google AI Studio
                </a>
              </p>
            </div>

            {/* Model Select */}
            <div className="space-y-2">
              <label className="text-sm font-medium">Model</label>
              <Select value={geminiModel} onValueChange={setGeminiModel}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="gemini-2.0-flash">
                    <span className="flex items-center gap-2">
                      Gemini 2.0 Flash
                      <Badge variant="secondary" className="text-xs">แนะนำ</Badge>
                    </span>
                  </SelectItem>
                  <SelectItem value="gemini-1.5-pro">Gemini 1.5 Pro</SelectItem>
                  <SelectItem value="gemini-1.5-flash">Gemini 1.5 Flash</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Save Button */}
            <div className="pt-2">
              <Button
                onClick={handleSaveGemini}
                disabled={updateGeminiMutation.isPending || (!geminiApiKey && !hasGeminiKey)}
                className="gap-2"
              >
                {updateGeminiMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    กำลังบันทึก...
                  </>
                ) : (
                  <>
                    <Save className="h-4 w-4" />
                    บันทึกการตั้งค่า
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Folder Picker Dialog */}
        <Dialog
          open={folderDialogOpen}
          onOpenChange={(open) => {
            setFolderDialogOpen(open)
            if (!open) resetFolderPicker()
          }}
        >
          <DialogContent className="max-w-md overflow-hidden">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <FolderOpen className="h-5 w-5" />
                เพิ่มโฟลเดอร์
              </DialogTitle>
              <DialogDescription>วางลิงค์หรือเลือกโฟลเดอร์จาก Google Drive ของคุณ</DialogDescription>
            </DialogHeader>

            <Tabs value={folderPickerTab} onValueChange={(v) => setFolderPickerTab(v as "url" | "select")}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="url" className="gap-1.5">
                  <LinkIcon className="h-3.5 w-3.5" />
                  วาง URL
                </TabsTrigger>
                <TabsTrigger value="select" className="gap-1.5">
                  <Folder className="h-3.5 w-3.5" />
                  เลือกโฟลเดอร์
                </TabsTrigger>
              </TabsList>

              {/* URL Tab */}
              <TabsContent value="url" className="space-y-4 mt-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">ลิงค์โฟลเดอร์ Google Drive</label>
                  <Input
                    type="text"
                    placeholder="https://drive.google.com/drive/folders/..."
                    value={folderUrlInput}
                    onChange={(e) => setFolderUrlInput(e.target.value)}
                    className="text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    วาง URL ที่ได้จากการแชร์โฟลเดอร์ใน Google Drive
                  </p>
                </div>
                <Button
                  onClick={handleAddFolderFromUrl}
                  disabled={!folderUrlInput.trim() || addFolderMutation.isPending}
                  className="w-full gap-2"
                >
                  {addFolderMutation.isPending ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" />
                      กำลังเพิ่ม...
                    </>
                  ) : (
                    <>
                      <Plus className="h-4 w-4" />
                      เพิ่มโฟลเดอร์
                    </>
                  )}
                </Button>
              </TabsContent>

              {/* Select Tab */}
              <TabsContent value="select" className="space-y-3 mt-4 overflow-hidden">
                {/* Folder Filter */}
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

                {folderPath.length > 0 && (
                  <div className="flex items-center gap-1 text-sm min-w-0 overflow-hidden">
                    <Button variant="ghost" size="sm" onClick={handleNavigateBack} className="h-7 px-2 shrink-0">
                      <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <span className="text-muted-foreground truncate">
                      {folderPath.map((f) => f.name).join(" / ")}
                    </span>
                  </div>
                )}

                <div className="max-h-[250px] overflow-y-auto overflow-x-hidden space-y-1">
                  {foldersLoading ? (
                    Array.from({ length: 5 }).map((_, i) => (
                      <div key={i} className="h-10 bg-muted rounded animate-pulse" />
                    ))
                  ) : folders && folders.length > 0 ? (
                    (() => {
                      const filteredFolders = folders.filter((folder) =>
                        folder.name.toLowerCase().includes(folderFilter.toLowerCase())
                      )
                      const maxDisplay = 20
                      const displayFolders = filteredFolders.slice(0, maxDisplay)
                      const hasMore = filteredFolders.length > maxDisplay

                      return filteredFolders.length > 0 ? (
                        <>
                          {displayFolders.map((folder) => (
                            <div
                              key={folder.id}
                              className="flex items-center gap-2 rounded-lg border p-2.5 hover:bg-muted/50"
                            >
                              <button
                                className="flex items-center gap-2 flex-1 min-w-0"
                                onClick={() => handleNavigateFolder(folder)}
                              >
                                <Folder className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                <span className="text-sm truncate block max-w-[200px]">{folder.name}</span>
                              </button>
                              <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                              <Button
                                variant="default"
                                size="sm"
                                onClick={() => handleSelectFolder(folder)}
                                disabled={addFolderMutation.isPending}
                                className="h-7 text-xs flex-shrink-0 gap-1"
                              >
                                <Plus className="h-3 w-3" />
                                เพิ่ม
                              </Button>
                            </div>
                          ))}
                          {hasMore && (
                            <p className="text-xs text-muted-foreground text-center py-2">
                              แสดง {maxDisplay} จาก {filteredFolders.length} โฟลเดอร์ - ใช้ช่องค้นหาเพื่อกรอง
                            </p>
                          )}
                        </>
                      ) : (
                        <p className="text-sm text-muted-foreground text-center py-8">
                          ไม่พบโฟลเดอร์ที่ตรงกับ "{folderFilter}"
                        </p>
                      )
                    })()
                  ) : (
                    <p className="text-sm text-muted-foreground text-center py-8">ไม่พบโฟลเดอร์</p>
                  )}
                </div>
              </TabsContent>
            </Tabs>
          </DialogContent>
        </Dialog>

        {/* Disconnect Confirmation Dialog */}
        <Dialog open={disconnectDialogOpen} onOpenChange={setDisconnectDialogOpen}>
          <DialogContent className="max-w-sm">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <AlertCircle className="h-5 w-5 text-destructive" />
                ยกเลิกการเชื่อมต่อ
              </DialogTitle>
              <DialogDescription>
                ต้องการยกเลิกการเชื่อมต่อ Google Drive หรือไม่?
              </DialogDescription>
            </DialogHeader>
            <p className="text-sm text-muted-foreground">
              รูปภาพที่ Sync มาแล้วจะยังคงอยู่ในระบบ สามารถเชื่อมต่อใหม่ได้ภายหลัง
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <Button
                variant="outline"
                onClick={() => setDisconnectDialogOpen(false)}
              >
                ยกเลิก
              </Button>
              <Button
                variant="destructive"
                onClick={handleDisconnect}
                disabled={disconnectMutation.isPending}
                className="gap-2"
              >
                {disconnectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    กำลังยกเลิก...
                  </>
                ) : (
                  <>
                    <Unlink className="h-4 w-4" />
                    ยืนยันยกเลิก
                  </>
                )}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </TooltipProvider>
  )
}
