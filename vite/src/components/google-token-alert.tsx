/**
 * GoogleTokenAlert - Listens for Google token expired events and shows reconnect toast
 */
import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { GOOGLE_TOKEN_EXPIRED_EVENT } from '@/shared/lib/api'
import { driveService } from '@/services'

export function GoogleTokenAlert() {
  const toastShownRef = useRef(false)

  useEffect(() => {
    const handleTokenExpired = () => {
      // ป้องกันแสดง toast ซ้ำหลายครั้ง
      if (toastShownRef.current) return
      toastShownRef.current = true

      toast.error('Google Drive Token หมดอายุ', {
        description: 'กรุณาเชื่อมต่อ Google Drive ใหม่เพื่อใช้งานต่อ',
        duration: 10000,
        action: {
          label: 'เชื่อมต่อใหม่',
          onClick: async () => {
            try {
              const response = await driveService.getConnectUrl()
              if (response.success && response.data) {
                window.location.href = response.data.authUrl
              }
            } catch (error) {
              console.error('Failed to get connect URL:', error)
              toast.error('ไม่สามารถเชื่อมต่อได้ กรุณาลองใหม่อีกครั้ง')
            }
          },
        },
        onDismiss: () => {
          // Reset flag เมื่อ toast ถูกปิด
          toastShownRef.current = false
        },
      })
    }

    window.addEventListener(GOOGLE_TOKEN_EXPIRED_EVENT, handleTokenExpired)

    return () => {
      window.removeEventListener(GOOGLE_TOKEN_EXPIRED_EVENT, handleTokenExpired)
    }
  }, [])

  return null
}
