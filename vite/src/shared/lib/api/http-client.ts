/**
 * HTTP Client - Centralized API client with interceptors
 */
import axios, { type AxiosError, type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'

// Error codes from backend
export const ERROR_CODES = {
  GOOGLE_TOKEN_EXPIRED: 'GOOGLE_TOKEN_EXPIRED',
} as const

// Types
export interface ApiResponse<T> {
  success: boolean
  message: string
  data?: T
  error?: string
  error_code?: string
}

export interface PaginatedResponse<T> {
  success: boolean
  message: string
  data: T
  meta: {
    total: number
    offset: number
    limit: number
  }
}

// API Base URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000, // 30 seconds
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor - add auth token
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Get token from localStorage (zustand persisted store)
    const authStorage = localStorage.getItem('auth-storage')
    if (authStorage) {
      try {
        const parsed = JSON.parse(authStorage)
        const token = parsed?.state?.token
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
      } catch {
        // Ignore parse errors
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Custom event for Google token expired
export const GOOGLE_TOKEN_EXPIRED_EVENT = 'google-token-expired'

// Response interceptor - handle errors
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse<unknown>>) => {
    const errorCode = error.response?.data?.error_code

    // Handle Google token expired (different from JWT 401)
    if (errorCode === ERROR_CODES.GOOGLE_TOKEN_EXPIRED) {
      // Dispatch custom event for UI to handle
      window.dispatchEvent(new CustomEvent(GOOGLE_TOKEN_EXPIRED_EVENT))
      return Promise.reject(error)
    }

    // Handle 401 Unauthorized (JWT expired)
    if (error.response?.status === 401) {
      // Clear auth and redirect to login
      localStorage.removeItem('auth-storage')
      window.location.href = '/login'
    }

    return Promise.reject(error)
  }
)

/**
 * Extract error message from API response
 */
export function getErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiResponse<unknown>>
    return axiosError.response?.data?.message || axiosError.message || 'An error occurred'
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'An error occurred'
}

/**
 * Set auth token manually
 */
export function setAuthToken(token: string | null): void {
  if (token) {
    apiClient.defaults.headers.common.Authorization = `Bearer ${token}`
  } else {
    delete apiClient.defaults.headers.common.Authorization
  }
}

export { apiClient }
export default apiClient
