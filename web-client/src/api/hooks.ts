import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getServerInfo,
  getProviders,
  getVoices,
  uploadBook,
  getBook,
  getBookStatus,
  getBookSegments,
  getVoiceMap,
  setVoiceMap,
  getPipelineStatus,
  getPersonas,
} from './client'
import type { VoiceMap } from '../types/api'

// Server queries
export function useServerInfo() {
  return useQuery({
    queryKey: ['serverInfo'],
    queryFn: getServerInfo,
    staleTime: Infinity,
  })
}

export function useProviders() {
  return useQuery({
    queryKey: ['providers'],
    queryFn: getProviders,
    staleTime: Infinity,
  })
}

export function useVoices(provider?: string) {
  return useQuery({
    queryKey: ['voices', provider],
    queryFn: () => getVoices(provider),
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  })
}

// Book queries
export function useBook(bookId: string | undefined) {
  return useQuery({
    queryKey: ['book', bookId],
    queryFn: () => getBook(bookId!),
    enabled: !!bookId,
  })
}

export function useBookStatus(bookId: string | undefined) {
  return useQuery({
    queryKey: ['bookStatus', bookId],
    queryFn: () => getBookStatus(bookId!),
    enabled: !!bookId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      // Refetch every 2 seconds if processing
      if (
        status &&
        ['uploaded', 'parsing', 'segmenting', 'synthesizing'].includes(status)
      ) {
        return 2000
      }
      return false
    },
  })
}

export function useBookSegments(bookId: string | undefined) {
  const queryClient = useQueryClient()
  
  return useQuery({
    queryKey: ['bookSegments', bookId],
    queryFn: () => getBookSegments(bookId!),
    enabled: !!bookId,
    retry: 3, // Retry failed requests up to 3 times
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 3000), // Exponential backoff: 1s, 2s, 3s
    refetchInterval: () => {
      // Get the latest book status from the query cache
      const bookStatus = queryClient.getQueryData(['bookStatus', bookId]) as { status: string } | undefined
      
      // Only poll if the book is still being processed
      if (!bookStatus) return false
      
      if (bookStatus.status === 'synthesizing' || bookStatus.status === 'segmenting') {
        return 5000 // Poll every 5 seconds
      }
      return false // Stop polling when done or ready
    },
    refetchOnWindowFocus: false, // Don't refetch when window regains focus
    refetchOnMount: false, // Don't refetch on component mount if data exists
  })
}

export function useVoiceMap(bookId: string | undefined) {
  return useQuery({
    queryKey: ['voiceMap', bookId],
    queryFn: () => getVoiceMap(bookId!),
    enabled: !!bookId,
  })
}

// Mutations
export function useUploadBook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      file,
      metadata,
    }: {
      file: File
      metadata?: { title?: string; author?: string; language?: string }
    }) => uploadBook(file, metadata),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['books'] })
    },
  })
}

export function useSetVoiceMap(bookId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      voiceMap,
      options,
    }: {
      voiceMap: Omit<VoiceMap, 'book_id'>
      options?: { initial?: boolean; update?: boolean }
    }) => setVoiceMap(bookId, voiceMap, options),
    onSuccess: async () => {
      // Invalidate and wait for personas query to refetch
      // This ensures the UI updates with the new unmapped personas list
      await queryClient.invalidateQueries({ queryKey: ['personas', bookId] })
      queryClient.invalidateQueries({ queryKey: ['voiceMap', bookId] })
      queryClient.invalidateQueries({ queryKey: ['bookStatus', bookId] })
      queryClient.invalidateQueries({ queryKey: ['pipelineStatus', bookId] })
    },
  })
}

// Hybrid Pipeline hooks
export function usePipelineStatus(bookId: string | undefined) {
  return useQuery({
    queryKey: ['pipelineStatus', bookId],
    queryFn: () => getPipelineStatus(bookId!),
    enabled: !!bookId,
    refetchInterval: (query) => {
      const stages = query.state.data?.stages
      if (!stages) return false
      
      // Check if any stage is in progress
      const isProcessing = stages.some(
        (stage) => stage.status === 'in_progress' || stage.status === 'waiting_for_mapping'
      )
      
      // Refetch every 2 seconds if processing
      return isProcessing ? 2000 : false
    },
  })
}

export function usePersonas(bookId: string | undefined) {
  return useQuery({
    queryKey: ['personas', bookId],
    queryFn: () => getPersonas(bookId!),
    enabled: !!bookId,
    retry: 3, // Retry failed requests up to 3 times
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 3000), // Exponential backoff: 1s, 2s, 3s
    refetchInterval: (query) => {
      const data = query.state.data
      // Refetch every 2 seconds if there are unmapped personas
      if (data && data.unmapped.length > 0) {
        return 2000
      }
      return false
    },
  })
}
