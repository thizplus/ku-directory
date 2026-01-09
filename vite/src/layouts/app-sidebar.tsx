"use client"

import * as React from "react"
import {
  FolderSearch,
  LayoutDashboard,
  Images,
  ScanFace,
  Newspaper,
  Settings,
  Activity,
} from "lucide-react"
import { Link, useLocation } from "react-router-dom"

import { NavUser } from "@/layouts/nav-user"
import { useAuth } from "@/hooks/use-auth"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  SidebarGroup,
  SidebarGroupLabel,
  useSidebar,
} from "@/components/ui/sidebar"

// Navigation items
const navItems = [
  {
    title: "แดชบอร์ด",
    url: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    title: "คลังรูปภาพ",
    url: "/gallery",
    icon: Images,
  },
  {
    title: "ค้นหาใบหน้า",
    url: "/face-search",
    icon: ScanFace,
  },
  {
    title: "เขียนข่าว AI",
    url: "/news-writer",
    icon: Newspaper,
  },
]

const settingsItems = [
  {
    title: "Activity Logs",
    url: "/activity-logs",
    icon: Activity,
  },
  {
    title: "ตั้งค่า",
    url: "/settings",
    icon: Settings,
  },
]

// NavLink component that closes sidebar on mobile when clicked
function NavLink({ item, isActive }: { item: typeof navItems[0], isActive: boolean }) {
  const { setOpenMobile } = useSidebar()

  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild isActive={isActive} tooltip={item.title}>
        <Link to={item.url} onClick={() => setOpenMobile(false)}>
          <item.icon />
          <span>{item.title}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const authUser = useAuth((state) => state.user)
  const location = useLocation()

  // Convert auth user to NavUser format
  const getAvatarUrl = (email: string) => {
    // Use UI Avatars as fallback
    const name = encodeURIComponent(email.split('@')[0])
    return `https://ui-avatars.com/api/?name=${name}&background=random&color=fff`
  }

  const user = authUser
    ? {
        name: `${authUser.firstName} ${authUser.lastName}`.trim() || authUser.username,
        email: authUser.email,
        avatar: authUser.avatar || getAvatarUrl(authUser.email),
      }
    : {
        name: "Guest",
        email: "",
        avatar: "",
      }

  const isActive = (url: string) => location.pathname === url

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link to="/dashboard">
                <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                  <FolderSearch className="size-4" />
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="font-medium">KU DIRECTORY</span>
                  <span className="text-xs">V{import.meta.env.VITE_APP_VERSION || '1.0.0'}</span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>เมนูหลัก</SidebarGroupLabel>
          <SidebarMenu>
            {navItems.map((item) => (
              <NavLink key={item.title} item={item} isActive={isActive(item.url)} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
        <SidebarGroup>
          <SidebarGroupLabel>ระบบ</SidebarGroupLabel>
          <SidebarMenu>
            {settingsItems.map((item) => (
              <NavLink key={item.title} item={item} isActive={isActive(item.url)} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
