/**
 * User Service Layer
 */
import { apiClient, type ApiResponse } from '@/shared/lib/api'
import { USER_API } from '@/shared/lib/api/constants/api'

export interface GeminiSettings {
  apiKey: string
  model: string
}

export interface UserProfile {
  id: string
  email: string
  username: string
  firstName: string
  lastName: string
  avatar: string
  role: string
  provider: string
  geminiApiKey?: string
  geminiModel?: string
}

/**
 * Get user profile
 */
async function getProfile(): Promise<ApiResponse<UserProfile>> {
  const response = await apiClient.get<ApiResponse<UserProfile>>(USER_API.PROFILE)
  return response.data
}

/**
 * Update Gemini settings
 */
async function updateGeminiSettings(settings: GeminiSettings): Promise<ApiResponse<UserProfile>> {
  const response = await apiClient.put<ApiResponse<UserProfile>>(USER_API.GEMINI_SETTINGS, settings)
  return response.data
}

export const userService = {
  getProfile,
  updateGeminiSettings,
}
