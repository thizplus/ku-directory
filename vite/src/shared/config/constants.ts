/**
 * Application Constants
 */

// API Base URL
export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Helper to generate thumbnail URL (proxied through backend)
export function getThumbnailUrl(driveFileId: string, token: string, size = 400): string {
  return `${API_BASE_URL}/drive/thumbnail/${driveFileId}?size=${size}&token=${token}`
}

// Pagination
export const PAGINATION = {
  DEFAULT_LIMIT: 20,
  MAX_LIMIT: 50,
} as const

// Form Limits
export const FORM_LIMITS = {
  FILE_SIZE_MAX: 10 * 1024 * 1024, // 10MB
} as const

// UI Settings
export const UI = {
  TOAST_DURATION: 3000,
  DEBOUNCE_DELAY: 300,
} as const

// Routes
export const ROUTES = {
  // Auth
  LOGIN: '/login',
  AUTH_CALLBACK: '/auth/callback',

  // Main
  DASHBOARD: '/dashboard',
  GALLERY: '/gallery',
  FACE_SEARCH: '/face-search',
  NEWS_WRITER: '/news-writer',
  SETTINGS: '/settings',
} as const

// Face Search Settings
export const FACE_SEARCH = {
  DEFAULT_THRESHOLD: 0.6,
  MIN_THRESHOLD: 0.3,
  MAX_THRESHOLD: 0.95,
  DEFAULT_LIMIT: 20,
  MIN_LIMIT: 5,
  MAX_LIMIT: 50,
} as const

// Allowed Image Types
export const ALLOWED_IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/gif',
] as const

// Cache Times (in milliseconds)
export const CACHE_TIMES = {
  DEFAULT: 60 * 1000, // 1 minute
  STATS: 30 * 1000, // 30 seconds
  PHOTOS: 5 * 60 * 1000, // 5 minutes
  FACES: 5 * 60 * 1000, // 5 minutes
  FOLDERS: 2 * 60 * 1000, // 2 minutes
} as const
