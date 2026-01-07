/**
 * News Service Layer
 */
import { apiClient, type ApiResponse } from '@/shared/lib/api'
import { NEWS_API } from '@/shared/lib/api/constants/api'

// Types
export interface NewsArticle {
  title: string
  paragraphs: {
    heading: string
    content: string
  }[]
  tags: string[]
}

export interface GenerateNewsRequest {
  photo_ids: string[]
  headings: string[]
  tone: string
  length: string
}

/**
 * Generate news from photos using Gemini AI
 */
async function generateNews(request: GenerateNewsRequest): Promise<ApiResponse<NewsArticle>> {
  const response = await apiClient.post<ApiResponse<NewsArticle>>(NEWS_API.GENERATE, request, {
    timeout: 60000, // 60 seconds for AI generation
  })
  return response.data
}

export const newsService = {
  generateNews,
}
