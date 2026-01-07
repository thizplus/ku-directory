import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import {
  Sparkles,
  Images,
  FileText,
  Loader2,
  Copy,
  Download,
  RefreshCw,
  Check,
  ImageOff,
  X,
  ArrowRight,
  AlertTriangle,
  Settings,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { Input } from "@/components/ui/input"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { useDriveStatus } from "@/features/drive"
import { useGenerateNews } from "@/features/news"
import { useAuth } from "@/hooks/use-auth"
import { getThumbnailUrl } from "@/shared/config/constants"
import { userService } from "@/services/user"
import type { Photo } from "@/shared/types"
import type { NewsArticle } from "@/services/news"

// Default headings for gen4kr format
const DEFAULT_HEADINGS = [
  "ความเป็นมา",
  "กิจกรรมที่จัด",
  "ผู้เข้าร่วม",
  "สรุป",
]

export default function NewsWriterPage() {
  const navigate = useNavigate()
  const { token } = useAuth()

  // Settings
  const [tone, setTone] = useState("formal")
  const [length, setLength] = useState("medium")
  const [headings, setHeadings] = useState<string[]>(DEFAULT_HEADINGS)

  // Photos
  const [selectedPhotos, setSelectedPhotos] = useState<Photo[]>([])

  // Generated content
  const [generatedArticle, setGeneratedArticle] = useState<NewsArticle | null>(null)
  const [editedContent, setEditedContent] = useState("")
  const [copied, setCopied] = useState(false)

  // Hooks
  const { data: driveStatus } = useDriveStatus()
  const generateMutation = useGenerateNews()

  // User profile query to check Gemini API key
  const { data: userProfile, isLoading: profileLoading } = useQuery({
    queryKey: ['user-profile'],
    queryFn: () => userService.getProfile(),
  })

  const isConnected = driveStatus?.connected ?? false
  const hasGeminiKey = userProfile?.data?.geminiApiKey && userProfile.data.geminiApiKey.length > 0

  // Load photos from sessionStorage on mount
  useEffect(() => {
    const storedPhotos = sessionStorage.getItem('newsWriterPhotos')
    if (storedPhotos) {
      try {
        const photos = JSON.parse(storedPhotos) as Photo[]
        setSelectedPhotos(photos)
      } catch {
        console.error('Failed to parse stored photos')
      }
    }
  }, [])

  // Update editable content when article is generated
  useEffect(() => {
    if (generatedArticle) {
      const content = formatArticleAsText(generatedArticle)
      setEditedContent(content)
    }
  }, [generatedArticle])

  // Format article as editable text
  const formatArticleAsText = (article: NewsArticle): string => {
    let text = `${article.title}\n\n`
    article.paragraphs.forEach((p, i) => {
      text += `${p.heading}\n${p.content}\n`
      if (i < article.paragraphs.length - 1) text += '\n'
    })
    if (article.tags.length > 0) {
      text += `\n\nแท็ก: ${article.tags.join(', ')}`
    }
    return text
  }

  const handleGenerate = () => {
    generateMutation.mutate({
      photo_ids: selectedPhotos.map(p => p.id),
      headings,
      tone,
      length,
    }, {
      onSuccess: (response) => {
        if (response.success && response.data) {
          setGeneratedArticle(response.data)
        }
      },
    })
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(editedContent)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDownload = () => {
    const blob = new Blob([editedContent], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `news-${new Date().toISOString().split('T')[0]}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const handleRemovePhoto = (photoId: string) => {
    const newPhotos = selectedPhotos.filter(p => p.id !== photoId)
    setSelectedPhotos(newPhotos)
    sessionStorage.setItem('newsWriterPhotos', JSON.stringify(newPhotos))
  }

  const handleHeadingChange = (index: number, value: string) => {
    const newHeadings = [...headings]
    newHeadings[index] = value
    setHeadings(newHeadings)
  }

  // Get folder name from first photo's path
  const getFolderName = (): string => {
    if (selectedPhotos.length === 0) return ""
    const path = selectedPhotos[0].drive_folder_path
    if (!path) return ""
    const parts = path.split('/')
    return parts[parts.length - 1] || parts[parts.length - 2] || ""
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">เขียนข่าว AI</h1>
          <p className="text-sm text-muted-foreground">
            สร้างข่าวประชาสัมพันธ์ตามหัวข้อที่กำหนด
          </p>
        </div>
        <Button
          onClick={handleGenerate}
          disabled={generateMutation.isPending || !hasGeminiKey || profileLoading}
        >
          {generateMutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              กำลังสร้าง...
            </>
          ) : (
            <>
              <Sparkles className="h-4 w-4 mr-2" />
              สร้างข่าว
            </>
          )}
        </Button>
      </div>

      {/* Alert when no Gemini API key */}
      {!profileLoading && !hasGeminiKey && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>ยังไม่ได้ตั้งค่า Gemini API Key</AlertTitle>
          <AlertDescription className="flex flex-col gap-2">
            <span>กรุณาไปที่หน้าตั้งค่าเพื่อใส่ Gemini API Key ก่อนใช้งานฟีเจอร์เขียนข่าว AI</span>
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

      {/* Settings Row */}
      <div className={`grid grid-cols-2 lg:grid-cols-6 gap-4 ${!hasGeminiKey && !profileLoading ? 'opacity-50 pointer-events-none' : ''}`}>
        <div>
          <p className="text-xs text-muted-foreground mb-1">โทน</p>
          <Select value={tone} onValueChange={setTone}>
            <SelectTrigger className="h-9">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="formal">เป็นทางการ</SelectItem>
              <SelectItem value="friendly">เป็นกันเอง</SelectItem>
              <SelectItem value="news">สไตล์ข่าว</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div>
          <p className="text-xs text-muted-foreground mb-1">ความยาว</p>
          <Select value={length} onValueChange={setLength}>
            <SelectTrigger className="h-9">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="short">สั้น</SelectItem>
              <SelectItem value="medium">กลาง</SelectItem>
              <SelectItem value="long">ยาว</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {headings.map((heading, index) => (
          <div key={index}>
            <p className="text-xs text-muted-foreground mb-1">ย่อหน้า {index + 1}</p>
            <Input
              value={heading}
              onChange={(e) => handleHeadingChange(index, e.target.value)}
              className="h-9"
            />
          </div>
        ))}
      </div>

      <Separator />

      {/* Main Content */}
      <div className="grid grid-cols-12 gap-6">
        {/* Sidebar - Photos */}
        <div className="col-span-12 lg:col-span-2">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Images className="h-3.5 w-3.5" />
                รูปภาพ {selectedPhotos.length > 0 ? `(${selectedPhotos.length})` : '(ไม่จำเป็น)'}
              </h3>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 text-xs px-2"
                onClick={() => navigate("/gallery")}
              >
                + เพิ่ม
              </Button>
            </div>

            {selectedPhotos.length === 0 ? (
              <div className="py-6 text-center">
                <Images className="mx-auto h-8 w-8 text-muted-foreground/30" />
                <p className="mt-2 text-xs text-muted-foreground">ไม่จำเป็นต้องมีรูป</p>
                <Button
                  variant="link"
                  size="sm"
                  className="mt-1 h-auto p-0 text-xs"
                  onClick={() => navigate("/gallery")}
                >
                  เพิ่มรูปภาพ (ถ้าต้องการ)
                </Button>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-1.5">
                {selectedPhotos.slice(0, 9).map((photo) => (
                  <div key={photo.id} className="relative group aspect-square">
                    <div className="w-full h-full bg-muted rounded overflow-hidden">
                      {token ? (
                        <img
                          src={getThumbnailUrl(photo.drive_file_id, token)}
                          alt={photo.file_name}
                          className="w-full h-full object-cover"
                          onError={(e) => {
                            const target = e.target as HTMLImageElement
                            target.style.display = 'none'
                          }}
                        />
                      ) : (
                        <div className="flex items-center justify-center h-full">
                          <ImageOff className="h-3 w-3 text-muted-foreground/50" />
                        </div>
                      )}
                    </div>
                    <button
                      className="absolute -top-1 -right-1 h-4 w-4 bg-destructive text-destructive-foreground rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={() => handleRemovePhoto(photo.id)}
                    >
                      <X className="h-2.5 w-2.5" />
                    </button>
                  </div>
                ))}
                {selectedPhotos.length > 9 && (
                  <div className="aspect-square bg-muted/50 rounded flex items-center justify-center">
                    <span className="text-xs text-muted-foreground">+{selectedPhotos.length - 9}</span>
                  </div>
                )}
              </div>
            )}

            {getFolderName() && (
              <p className="text-xs text-muted-foreground truncate" title={getFolderName()}>
                📁 {getFolderName()}
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
                onClick={() => navigate("/gallery")}
              >
                <Images className="h-3.5 w-3.5 mr-1.5" />
                คลังรูปภาพ
              </Button>
            </div>
          </div>
        </div>

        {/* Main Area - Output */}
        <div className={`col-span-12 lg:col-span-10 ${!hasGeminiKey && !profileLoading ? 'opacity-50 pointer-events-none' : ''}`}>
          {!isConnected ? (
            <div className="py-12 text-center">
              <Sparkles className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-4 text-sm text-muted-foreground">
                ยังไม่ได้เชื่อมต่อ Google Drive
              </p>
              <Button className="mt-4" size="sm" onClick={() => navigate("/settings")}>
                ไปที่ตั้งค่า
                <ArrowRight className="h-3 w-3 ml-1" />
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Output Header */}
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium flex items-center gap-2">
                  <FileText className="h-4 w-4" />
                  ข่าวที่สร้าง
                </h3>
                {editedContent && (
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" className="h-7 px-2" onClick={handleCopy}>
                      {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 px-2"
                      onClick={handleGenerate}
                      disabled={generateMutation.isPending || !hasGeminiKey}
                    >
                      <RefreshCw className={`h-3.5 w-3.5 ${generateMutation.isPending ? 'animate-spin' : ''}`} />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-7 px-2" onClick={handleDownload}>
                      <Download className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                )}
              </div>

              {/* Error Message */}
              {generateMutation.isError && (
                <p className="text-xs text-destructive">
                  เกิดข้อผิดพลาด: {generateMutation.error?.message || 'ไม่สามารถสร้างข่าวได้'}
                </p>
              )}

              {/* Textarea */}
              <Textarea
                placeholder="กดปุ่ม 'สร้างข่าว' เพื่อสร้างข่าวตามหัวข้อที่กำหนด (รูปภาพไม่จำเป็น)"
                className="min-h-[500px] resize-none text-sm leading-relaxed"
                value={editedContent}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setEditedContent(e.target.value)}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
