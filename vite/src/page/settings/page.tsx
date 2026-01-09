import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Cloud,
  Key,
  Bell,
  Loader2,
  FolderOpen,
  RefreshCw,
  ChevronRight,
  ArrowLeft,
  Folder,
  AlertCircle,
  ChevronDown,
  Save,
  Sparkles,
  X,
  Link,
  Plus,
  Radio,
  Clock,
  AlertTriangle,
  WifiOff,
} from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

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
    ? `หมดอายุ ${expiryDate.toLocaleDateString('th-TH', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })}`
    : ''

  switch (status) {
    case 'active':
      return {
        icon: Radio,
        label: 'Webhook ทำงานปกติ',
        description: expiryText,
        className: 'text-green-600',
        badgeVariant: 'outline' as const,
        badgeClassName: 'text-green-600 border-green-600/30',
      }
    case 'expiring':
      return {
        icon: Clock,
        label: 'Webhook ใกล้หมดอายุ',
        description: expiryText,
        className: 'text-yellow-600',
        badgeVariant: 'outline' as const,
        badgeClassName: 'text-yellow-600 border-yellow-600/30',
      }
    case 'expired':
      return {
        icon: AlertTriangle,
        label: 'Webhook หมดอายุ',
        description: 'รอ auto-renewal หรือกด Sync',
        className: 'text-red-600',
        badgeVariant: 'outline' as const,
        badgeClassName: 'text-red-600 border-red-600/30',
      }
    case 'inactive':
    default:
      return {
        icon: WifiOff,
        label: 'Webhook ไม่ทำงาน',
        description: 'กด Sync เพื่อลงทะเบียน',
        className: 'text-muted-foreground',
        badgeVariant: 'outline' as const,
        badgeClassName: '',
      }
  }
}

export default function SettingsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [driveOpen, setDriveOpen] = useState(true)
  const [apiOpen, setApiOpen] = useState(false)
  const [notifyOpen, setNotifyOpen] = useState(false)
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
      // Only set if the input is empty (first load)
      if (!geminiApiKey && userProfile.data.geminiApiKey) {
        // Don't show masked key in input, leave empty for new input
      }
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
    // Format: https://drive.google.com/drive/folders/FOLDER_ID or https://drive.google.com/drive/folders/FOLDER_ID?usp=sharing
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
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold">ตั้งค่า</h1>
        <p className="text-sm text-muted-foreground">จัดการการเชื่อมต่อและตั้งค่าระบบ</p>
      </div>

      <Separator />

      {/* Settings List */}
      <div className="space-y-3">
        {/* Google Drive */}
        <Collapsible open={driveOpen} onOpenChange={setDriveOpen}>
          <div className="rounded-lg border">
            <CollapsibleTrigger className="w-full">
              <div className="flex items-center justify-between p-4 hover:bg-muted/50 transition-colors">
                <div className="flex items-center gap-3">
                  <Cloud className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium">Google Drive</p>
                    <p className="text-xs text-muted-foreground">Sync รูปภาพจาก Drive</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {statusLoading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : isConnected ? (
                    <Badge variant="outline" className="text-green-600 border-green-600/30">
                      เชื่อมต่อแล้ว
                    </Badge>
                  ) : (
                    <Badge variant="outline">ยังไม่เชื่อมต่อ</Badge>
                  )}
                  <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform ${driveOpen ? "rotate-180" : ""}`} />
                </div>
              </div>
            </CollapsibleTrigger>

            <CollapsibleContent>
              <div className="border-t px-4 py-4 space-y-4">
                {isConnected ? (
                  <>
                    {/* Shared Folders List */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium">โฟลเดอร์ที่เพิ่ม ({sharedFolders.length})</span>
                        <Button variant="outline" size="sm" onClick={() => setFolderDialogOpen(true)}>
                          <FolderOpen className="h-3.5 w-3.5 mr-1.5" />
                          เพิ่มโฟลเดอร์
                        </Button>
                      </div>

                      {sharedFoldersLoading ? (
                        <div className="space-y-2">
                          {Array.from({ length: 2 }).map((_, i) => (
                            <div key={i} className="h-14 bg-muted rounded animate-pulse" />
                          ))}
                        </div>
                      ) : sharedFolders.length > 0 ? (
                        <div className="space-y-2">
                          {sharedFolders.map((folder: SharedFolder) => {
                            const webhookInfo = getWebhookStatusInfo(folder.webhook_status, folder.webhook_expiry)
                            const WebhookIcon = webhookInfo.icon
                            return (
                              <div key={folder.id} className="rounded-lg border p-3 space-y-2">
                                <div className="flex items-center justify-between">
                                  <div className="flex items-center gap-2">
                                    <Folder className="h-4 w-4 text-yellow-500" />
                                    <span className="text-sm font-medium">{folder.drive_folder_name}</span>
                                  </div>
                                  <Badge variant="secondary" className="text-xs">
                                    {folder.sync_status === 'syncing' ? 'กำลัง Sync' :
                                     folder.sync_status === 'completed' ? 'Sync แล้ว' :
                                     folder.sync_status === 'failed' ? 'Sync ล้มเหลว' : 'รอ Sync'}
                                  </Badge>
                                </div>
                                {/* Webhook Status */}
                                <div className="flex items-center gap-1.5 text-xs">
                                  <WebhookIcon className={`h-3 w-3 ${webhookInfo.className}`} />
                                  <span className={webhookInfo.className}>{webhookInfo.label}</span>
                                  {webhookInfo.description && (
                                    <span className="text-muted-foreground">• {webhookInfo.description}</span>
                                  )}
                                </div>
                                <div className="flex items-center justify-between text-xs text-muted-foreground">
                                  <span>{folder.photo_count} รูป</span>
                                  <div className="flex items-center gap-1">
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-6 text-xs"
                                      onClick={() => handleSyncFolder(folder.id)}
                                      disabled={folder.sync_status === 'syncing' || triggerSyncMutation.isPending}
                                    >
                                      <RefreshCw className={`h-3 w-3 mr-1 ${folder.sync_status === 'syncing' ? 'animate-spin' : ''}`} />
                                      Sync
                                    </Button>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-6 text-xs text-destructive hover:text-destructive"
                                      onClick={() => handleRemoveFolder(folder.id)}
                                      disabled={removeFolderMutation.isPending}
                                    >
                                      <X className="h-3 w-3" />
                                    </Button>
                                  </div>
                                </div>
                              </div>
                            )
                          })}
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground text-center py-4 flex items-center justify-center gap-1">
                          <AlertCircle className="h-3 w-3" />
                          ยังไม่มีโฟลเดอร์ กดเพิ่มโฟลเดอร์เพื่อเริ่มต้น
                        </p>
                      )}
                    </div>

                    <Separator />

                    {/* Disconnect */}
                    <div className="flex justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDisconnectDialogOpen(true)}
                        disabled={disconnectMutation.isPending || hasSyncingFolder}
                      >
                        ยกเลิกการเชื่อมต่อ
                      </Button>
                    </div>
                  </>
                ) : (
                  <div className="text-center py-4">
                    <p className="text-sm text-muted-foreground mb-3">
                      เชื่อมต่อ Google Drive เพื่อ Sync รูปภาพ
                    </p>
                    <Button size="sm" onClick={handleConnect} disabled={connectMutation.isPending}>
                      {connectMutation.isPending && <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />}
                      เชื่อมต่อ
                    </Button>
                  </div>
                )}
              </div>
            </CollapsibleContent>
          </div>
        </Collapsible>

        {/* Gemini AI Settings */}
        <Collapsible open={apiOpen} onOpenChange={setApiOpen}>
          <div className="rounded-lg border">
            <CollapsibleTrigger className="w-full">
              <div className="flex items-center justify-between p-4 hover:bg-muted/50 transition-colors">
                <div className="flex items-center gap-3">
                  <Sparkles className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium">Gemini AI</p>
                    <p className="text-xs text-muted-foreground">ตั้งค่าสำหรับเขียนข่าว AI</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {profileLoading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : hasGeminiKey ? (
                    <Badge variant="outline" className="text-primary border-primary/30">
                      ตั้งค่าแล้ว
                    </Badge>
                  ) : (
                    <Badge variant="outline">ยังไม่ได้ตั้งค่า</Badge>
                  )}
                  <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform ${apiOpen ? "rotate-180" : ""}`} />
                </div>
              </div>
            </CollapsibleTrigger>

            <CollapsibleContent>
              <div className="border-t px-4 py-4 space-y-4">
                {/* Current Status */}
                {hasGeminiKey && (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Key className="h-4 w-4" />
                    <span>API Key: {userProfile?.data?.geminiApiKey}</span>
                  </div>
                )}

                {/* API Key Input */}
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">API Key</label>
                  <Input
                    type="password"
                    placeholder={hasGeminiKey ? "ใส่ API Key ใหม่เพื่อเปลี่ยน" : "ใส่ Gemini API Key"}
                    value={geminiApiKey}
                    onChange={(e) => setGeminiApiKey(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
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
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Model</label>
                  <Select value={geminiModel} onValueChange={setGeminiModel}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="gemini-2.0-flash">Gemini 2.0 Flash (แนะนำ)</SelectItem>
                      <SelectItem value="gemini-1.5-pro">Gemini 1.5 Pro</SelectItem>
                      <SelectItem value="gemini-1.5-flash">Gemini 1.5 Flash</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* Save Button */}
                <div className="pt-2">
                  <Button
                    size="sm"
                    onClick={handleSaveGemini}
                    disabled={updateGeminiMutation.isPending || (!geminiApiKey && !hasGeminiKey)}
                  >
                    {updateGeminiMutation.isPending ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                        บันทึก...
                      </>
                    ) : (
                      <>
                        <Save className="h-3.5 w-3.5 mr-1.5" />
                        บันทึก
                      </>
                    )}
                  </Button>
                </div>
              </div>
            </CollapsibleContent>
          </div>
        </Collapsible>

        {/* Notifications */}
        <Collapsible open={notifyOpen} onOpenChange={setNotifyOpen}>
          <div className="rounded-lg border">
            <CollapsibleTrigger className="w-full">
              <div className="flex items-center justify-between p-4 hover:bg-muted/50 transition-colors">
                <div className="flex items-center gap-3">
                  <Bell className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium">การแจ้งเตือน</p>
                    <p className="text-xs text-muted-foreground">ตั้งค่าการแจ้งเตือน</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-green-600 border-green-600/30">เปิด</Badge>
                  <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform ${notifyOpen ? "rotate-180" : ""}`} />
                </div>
              </div>
            </CollapsibleTrigger>

            <CollapsibleContent>
              <div className="border-t px-4 py-6 text-center">
                <p className="text-sm text-muted-foreground">การแจ้งเตือนเปิดใช้งานอยู่</p>
              </div>
            </CollapsibleContent>
          </div>
        </Collapsible>
      </div>

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
            <DialogTitle>เพิ่มโฟลเดอร์</DialogTitle>
            <DialogDescription>วางลิงค์หรือเลือกโฟลเดอร์จาก Google Drive</DialogDescription>
          </DialogHeader>

          <Tabs value={folderPickerTab} onValueChange={(v) => setFolderPickerTab(v as "url" | "select")}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="url" className="gap-1.5">
                <Link className="h-3.5 w-3.5" />
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
                className="w-full"
              >
                {addFolderMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    กำลังเพิ่ม...
                  </>
                ) : (
                  "เพิ่มโฟลเดอร์"
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
                    // Pagination: show max 20 folders at a time
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
            <DialogTitle>ยกเลิกการเชื่อมต่อ</DialogTitle>
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
              size="sm"
              onClick={() => setDisconnectDialogOpen(false)}
            >
              ยกเลิก
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={handleDisconnect}
              disabled={disconnectMutation.isPending}
            >
              {disconnectMutation.isPending ? (
                <>
                  <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  กำลังยกเลิก...
                </>
              ) : (
                "ยืนยัน"
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
