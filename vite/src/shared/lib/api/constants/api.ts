/**
 * API Endpoint Constants
 */

// Auth API
export const AUTH_API = {
  GOOGLE_LOGIN: '/auth/google',
  CALLBACK: '/auth/callback',
  ME: '/auth/me',
  LOGOUT: '/auth/logout',
} as const

// User API
export const USER_API = {
  PROFILE: '/users/profile',
  UPDATE_PROFILE: '/users/profile',
  GEMINI_SETTINGS: '/users/gemini-settings',
} as const

// Drive API
export const DRIVE_API = {
  CONNECT: '/drive/connect',
  CALLBACK: '/drive/callback',
  STATUS: '/drive/status',
  DISCONNECT: '/drive/disconnect',
  FOLDERS: '/drive/folders',
  SET_ROOT_FOLDER: '/drive/root-folder',
  SYNC: '/drive/sync',
  SYNC_STATUS: '/drive/sync/status',
  PHOTOS: '/drive/photos',
  DOWNLOAD: '/drive/download',
} as const

// Face API
export const FACE_API = {
  DETECT: '/faces/detect',
  SEARCH_BY_IMAGE: '/faces/search/image',
  SEARCH_BY_FACE: '/faces/search/face',
  LIST: '/faces',
  BY_PHOTO: (photoId: string) => `/faces/photo/${photoId}`,
  STATS: '/faces/stats',
  RETRY: '/faces/retry',
} as const

// Photo API
export const PHOTO_API = {
  LIST: '/photos',
  DETAIL: (id: string) => `/photos/${id}`,
} as const

// News API
export const NEWS_API = {
  GENERATE: '/news/generate',
} as const

// Shared Folders API
export const FOLDERS_API = {
  LIST: '/folders',
  GET: (id: string) => `/folders/${id}`,
  ADD: '/folders',
  REMOVE: (id: string) => `/folders/${id}`,
  SYNC: (id: string) => `/folders/${id}/sync`,
  PHOTOS: (id: string) => `/folders/${id}/photos`,
  SUBFOLDERS: (id: string) => `/folders/${id}/subfolders`,
} as const
