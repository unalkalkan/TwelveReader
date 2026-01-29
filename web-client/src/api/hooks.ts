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
  return useQuery({
    queryKey: ['bookSegments', bookId],
    queryFn: () => getBookSegments(bookId!),
    enabled: !!bookId,
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
    mutationFn: (voiceMap: Omit<VoiceMap, 'book_id'>) =>
      setVoiceMap(bookId, voiceMap),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['voiceMap', bookId] })
      queryClient.invalidateQueries({ queryKey: ['bookStatus', bookId] })
    },
  })
}
