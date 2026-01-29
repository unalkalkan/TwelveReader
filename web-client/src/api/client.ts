import {
  BookMetadataSchema,
  ProcessingStatusSchema,
  SegmentSchema,
  VoiceMapSchema,
  VoicesResponseSchema,
  ServerInfoSchema,
  ProvidersSchema,
  type BookMetadata,
  type ProcessingStatus,
  type Segment,
  type VoiceMap,
  type VoicesResponse,
  type ServerInfo,
  type Providers,
} from '../types/api'

const API_BASE = '/api/v1'

// Helper function to handle API responses
async function apiRequest<T>(
  url: string,
  options?: RequestInit,
  schema?: any
): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Request failed',
      code: 'UNKNOWN_ERROR',
    }))
    throw new Error(error.error || 'Request failed')
  }

  const data = await response.json()
  return schema ? schema.parse(data) : data
}

// Server Info
export async function getServerInfo(): Promise<ServerInfo> {
  return apiRequest<ServerInfo>(`${API_BASE}/info`, {}, ServerInfoSchema)
}

// Providers
export async function getProviders(): Promise<Providers> {
  return apiRequest<Providers>(`${API_BASE}/providers`, {}, ProvidersSchema)
}

// Voices
export async function getVoices(provider?: string): Promise<VoicesResponse> {
  const params = new URLSearchParams()
  if (provider) params.append('provider', provider)
  const queryString = params.toString()
  return apiRequest<VoicesResponse>(
    `${API_BASE}/voices${queryString ? `?${queryString}` : ''}`,
    {},
    VoicesResponseSchema
  )
}

// Books
export async function uploadBook(
  file: File,
  metadata?: { title?: string; author?: string; language?: string }
): Promise<BookMetadata> {
  const formData = new FormData()
  formData.append('file', file)
  if (metadata?.title) formData.append('title', metadata.title)
  if (metadata?.author) formData.append('author', metadata.author)
  if (metadata?.language) formData.append('language', metadata.language)

  const response = await fetch(`${API_BASE}/books`, {
    method: 'POST',
    body: formData,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Upload failed',
      code: 'UPLOAD_ERROR',
    }))
    throw new Error(error.error || 'Upload failed')
  }

  const data = await response.json()
  return BookMetadataSchema.parse(data)
}

export async function getBook(bookId: string): Promise<BookMetadata> {
  return apiRequest<BookMetadata>(
    `${API_BASE}/books/${bookId}`,
    {},
    BookMetadataSchema
  )
}

export async function getBookStatus(bookId: string): Promise<ProcessingStatus> {
  return apiRequest<ProcessingStatus>(
    `${API_BASE}/books/${bookId}/status`,
    {},
    ProcessingStatusSchema
  )
}

export async function getBookSegments(bookId: string): Promise<Segment[]> {
  const data = await apiRequest<any[]>(`${API_BASE}/books/${bookId}/segments`)
  return data.map((item) => SegmentSchema.parse(item))
}

// Voice Map
export async function getVoiceMap(bookId: string): Promise<VoiceMap> {
  return apiRequest<VoiceMap>(
    `${API_BASE}/books/${bookId}/voice-map`,
    {},
    VoiceMapSchema
  )
}

export async function setVoiceMap(
  bookId: string,
  voiceMap: Omit<VoiceMap, 'book_id'>
): Promise<VoiceMap> {
  return apiRequest<VoiceMap>(
    `${API_BASE}/books/${bookId}/voice-map`,
    {
      method: 'POST',
      body: JSON.stringify(voiceMap),
    },
    VoiceMapSchema
  )
}

// Streaming and Download
export function getStreamUrl(bookId: string, after?: string): string {
  const params = new URLSearchParams()
  if (after) params.append('after', after)
  return `${API_BASE}/books/${bookId}/stream${params.toString() ? `?${params}` : ''}`
}

export function getDownloadUrl(bookId: string): string {
  return `${API_BASE}/books/${bookId}/download`
}

export function getAudioUrl(bookId: string, segmentId: string): string {
  return `${API_BASE}/books/${bookId}/audio/${segmentId}`
}
