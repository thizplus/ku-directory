import { useMutation } from '@tanstack/react-query'
import { newsService, type GenerateNewsRequest, type NewsArticle } from '@/services/news'
import type { ApiResponse } from '@/shared/lib/api'

/**
 * Hook for generating news using Gemini AI
 */
export function useGenerateNews() {
  return useMutation<ApiResponse<NewsArticle>, Error, GenerateNewsRequest>({
    mutationFn: (request) => newsService.generateNews(request),
  })
}
