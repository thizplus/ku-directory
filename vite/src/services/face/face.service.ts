/**
 * Face Service - API calls for face detection and search
 */
import axios from 'axios'
import { apiClient, FACE_API, type ApiResponse } from '@/shared/lib/api'
import type {
  FaceSearchData,
  FaceProcessingStats,
  FaceListData,
  Face,
  DetectedFacesData,
} from '@/shared/types'

export const faceService = {
  /**
   * Detect all faces in an uploaded image
   * Returns bounding boxes for face selection
   */
  detectFaces: async (
    imageFile: File
  ): Promise<ApiResponse<DetectedFacesData>> => {
    const formData = new FormData()
    formData.append('image', imageFile)

    try {
      const response = await apiClient.post<ApiResponse<DetectedFacesData>>(
        FACE_API.DETECT,
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      )
      return response.data
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.data) {
        return error.response.data as ApiResponse<DetectedFacesData>
      }
      throw error
    }
  },

  /**
   * Search faces by uploading an image
   * @param faceIndex - Index of the face to search (default: 0)
   */
  searchByImage: async (
    imageFile: File,
    limit = 20,
    threshold = 0.6,
    faceIndex = 0
  ): Promise<ApiResponse<FaceSearchData>> => {
    const formData = new FormData()
    formData.append('image', imageFile)

    const params = new URLSearchParams({
      limit: limit.toString(),
      threshold: threshold.toString(),
      face_index: faceIndex.toString(),
    })

    try {
      const response = await apiClient.post<ApiResponse<FaceSearchData>>(
        `${FACE_API.SEARCH_BY_IMAGE}?${params}`,
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        }
      )
      return response.data
    } catch (error) {
      // Handle axios errors - extract message from response
      if (axios.isAxiosError(error) && error.response?.data) {
        return error.response.data as ApiResponse<FaceSearchData>
      }
      throw error
    }
  },

  /**
   * Search faces by existing face ID
   */
  searchByFaceId: async (
    faceId: string,
    limit = 20,
    threshold = 0.6
  ): Promise<ApiResponse<FaceSearchData>> => {
    const response = await apiClient.post<ApiResponse<FaceSearchData>>(
      FACE_API.SEARCH_BY_FACE,
      {
        face_id: faceId,
        limit,
        threshold,
      }
    )

    return response.data
  },

  /**
   * Get face processing statistics
   */
  getStats: async (): Promise<ApiResponse<FaceProcessingStats>> => {
    const response = await apiClient.get<ApiResponse<FaceProcessingStats>>(
      FACE_API.STATS
    )

    return response.data
  },

  /**
   * Get paginated list of faces
   */
  getFaces: async (
    page = 1,
    limit = 50
  ): Promise<ApiResponse<FaceListData>> => {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    })

    const response = await apiClient.get<ApiResponse<FaceListData>>(
      `${FACE_API.LIST}?${params}`
    )

    return response.data
  },

  /**
   * Get faces for a specific photo
   */
  getFacesByPhoto: async (
    photoId: string
  ): Promise<ApiResponse<{ faces: Face[]; count: number }>> => {
    const response = await apiClient.get<ApiResponse<{ faces: Face[]; count: number }>>(
      FACE_API.BY_PHOTO(photoId)
    )

    return response.data
  },

  /**
   * Retry failed face processing
   * Resets failed photos to pending for reprocessing
   */
  retryFailed: async (
    folderId?: string
  ): Promise<ApiResponse<{ reset_count: number }>> => {
    const params = folderId ? `?folder_id=${folderId}` : ''
    const response = await apiClient.post<ApiResponse<{ reset_count: number }>>(
      `${FACE_API.RETRY}${params}`
    )

    return response.data
  },
}
