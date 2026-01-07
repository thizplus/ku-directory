import { useState } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { getGoogleLoginUrl } from "@/hooks/use-auth"
import { useSearchParams } from "react-router-dom"
import { AlertCircle } from "lucide-react"

export function LoginForm({
  className,
  ...props
}: React.ComponentProps<"div">) {
  const [searchParams] = useSearchParams()
  const [isLoading, setIsLoading] = useState(false)
  const error = searchParams.get("error")

  const handleGoogleLogin = () => {
    setIsLoading(true)
    window.location.href = getGoogleLoginUrl("/dashboard")
  }

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-bold">เข้าสู่ระบบ</h1>
        <p className="text-muted-foreground text-sm text-balance">
          เข้าสู่ระบบด้วยบัญชี Google ของมหาวิทยาลัย
        </p>
      </div>

      {error && (
        <div className="flex items-center gap-2 bg-destructive/10 text-destructive text-sm p-3 rounded-md">
          <AlertCircle className="h-4 w-4 shrink-0" />
          <span>
            {error === "invalid_state" && "เกิดข้อผิดพลาดในการยืนยันตัวตน"}
            {error === "no_token" && "ไม่สามารถเข้าสู่ระบบได้"}
            {error === "auth_failed" && "การเข้าสู่ระบบล้มเหลว"}
            {!["invalid_state", "no_token", "auth_failed"].includes(error) && error}
          </span>
        </div>
      )}

      <Button
        type="button"
        variant="outline"
        size="lg"
        className="w-full gap-3"
        onClick={handleGoogleLogin}
        disabled={isLoading}
      >
        {isLoading ? (
          <span className="animate-pulse">กำลังเข้าสู่ระบบ...</span>
        ) : (
          <>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" className="h-5 w-5">
              <path
                d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"
                fill="currentColor"
              />
            </svg>
            เข้าสู่ระบบด้วย Google
          </>
        )}
      </Button>

      <div className="text-center text-sm text-muted-foreground">
        ระบบค้นหารูปภาพกิจกรรมของมหาวิทยาลัย
      </div>
    </div>
  )
}
