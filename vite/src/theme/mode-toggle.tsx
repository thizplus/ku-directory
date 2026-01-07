import { Moon, Sun } from "lucide-react"
import { useTheme } from "./theme-provider"
import { useState, useEffect } from "react"

export function ModeToggle() {
  const { theme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  // ตรวจสอบว่า component ถูก mount แล้วเพื่อป้องกัน hydration error
  useEffect(() => {
    setMounted(true)
  }, [])

  // สลับระหว่าง light และ dark mode
  const toggleTheme = () => {
    if (theme === "dark") {
      setTheme("light")
    } else {
      setTheme("dark")
    }
  }

  // ถ้ายังไม่ mount ให้แสดง placeholder เพื่อป้องกัน layout shift
  if (!mounted) {
    return (
      <button 
        className="ml-auto w-8 h-8 rounded-full bg-transparent border border-border flex items-center justify-center opacity-0"
        aria-label="เปลี่ยนธีม"
      />
    )
  }

  return (
    <button
      onClick={toggleTheme}
      className="ml-auto w-8 h-8 flex items-center justify-center border border-border rounded-full hover:bg-muted/50 text-muted-foreground"
      aria-label="เปลี่ยนธีม"
    >
      {theme === "dark" ? (
        <Moon size={20} className="h-5 w-5" />
      ) : (
        <Sun size={20} className="h-5 w-5" />
      )}
    </button>
  )
}