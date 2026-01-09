import { Moon, Sun } from "lucide-react"
import { useTheme } from "./theme-provider"
import { useState, useEffect } from "react"

export function ModeToggle() {
  const { theme, setTheme } = useTheme()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const isDark = theme === "dark"

  const toggleTheme = () => {
    setTheme(isDark ? "light" : "dark")
  }

  if (!mounted) {
    return <div className="ml-auto h-7 w-14 rounded-full bg-muted opacity-0" />
  }

  return (
    <button
      onClick={toggleTheme}
      className="theme-toggle ml-auto relative h-7 w-14 rounded-full p-1 transition-colors duration-300 ease-in-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      aria-label="เปลี่ยนธีม"
    >
      {/* Track icons */}
      <div className="absolute inset-1 flex items-center justify-between px-1">
        <Sun className={`h-3.5 w-3.5 text-foreground transition-opacity duration-300 ${isDark ? 'opacity-30' : 'opacity-0'}`} />
        <Moon className={`h-3.5 w-3.5 text-foreground transition-opacity duration-300 ${isDark ? 'opacity-0' : 'opacity-30'}`} />
      </div>

      {/* Sliding thumb */}
      <div
        className={`
          theme-toggle-thumb relative z-10 flex h-5 w-5 items-center justify-center rounded-full shadow-sm
          transition-transform duration-300 ease-in-out
          ${isDark ? 'translate-x-7' : 'translate-x-0'}
        `}
      >
        {isDark ? (
          <Moon className="theme-toggle-icon h-3 w-3" />
        ) : (
          <Sun className="theme-toggle-icon h-3 w-3" />
        )}
      </div>
    </button>
  )
}
