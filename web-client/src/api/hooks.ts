import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getServerInfo,
  getProviders,
  getVoices,
  getBooks,
  uploadBook,
  getBook,
  getBookStatus,
  getBookSegments,
  getVoiceMap,
  setVoiceMap,
  getPipelineStatus,
  getPersonas,
} from './client';
import type { VoiceMap } from '../types/api';

// ── Server ──────────────────────────────────────────────────────────────

export function useServerInfo() {
  return useQuery({
    queryKey: ['serverInfo'],
    queryFn: getServerInfo,
    staleTime: Infinity,
  });
}

export function useProviders() {
  return useQuery({
    queryKey: ['providers'],
    queryFn: getProviders,
    staleTime: Infinity,
  });
}

// ── Books ───────────────────────────────────────────────────────────────

export function useBooks() {
  return useQuery({
    queryKey: ['books'],
    queryFn: getBooks,
    staleTime: 30_000,
  });
}

export function useBook(bookId: string | undefined) {
  return useQuery({
    queryKey: ['book', bookId],
    queryFn: () => getBook(bookId!),
    enabled: !!bookId,
  });
}

// ── Voices ──────────────────────────────────────────────────────────────

export function useVoices(provider?: string) {
  return useQuery({
    queryKey: ['voices', provider],
    queryFn: () => getVoices(provider),
    staleTime: 5 * 60_000,
  });
}

// ── Processing Status ───────────────────────────────────────────────────

export function useBookStatus(bookId: string | undefined) {
  return useQuery({
    queryKey: ['bookStatus', bookId],
    queryFn: () => getBookStatus(bookId!),
    enabled: !!bookId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (
        status &&
        ['uploaded', 'parsing', 'segmenting', 'synthesizing'].includes(status)
      ) {
        return 2_000;
      }
      return false;
    },
  });
}

export function useBookSegments(bookId: string | undefined) {
  const qc = useQueryClient();

  return useQuery({
    queryKey: ['bookSegments', bookId],
    queryFn: () => getBookSegments(bookId!),
    enabled: !!bookId,
    retry: 3,
    retryDelay: (i) => Math.min(1_000 * 2 ** i, 3_000),
    refetchInterval: () => {
      const bs = qc.getQueryData(['bookStatus', bookId]) as
        | { status: string }
        | undefined;
      if (!bs) return false;
      if (bs.status === 'synthesizing' || bs.status === 'segmenting')
        return 5_000;
      return false;
    },
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });
}

// ── Voice Map ───────────────────────────────────────────────────────────

export function useVoiceMap(bookId: string | undefined) {
  return useQuery({
    queryKey: ['voiceMap', bookId],
    queryFn: () => getVoiceMap(bookId!),
    enabled: !!bookId,
  });
}

// ── Mutations ───────────────────────────────────────────────────────────

export function useUploadBook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      fileUri,
      fileName,
      mimeType,
      metadata,
    }: {
      fileUri: string;
      fileName: string;
      mimeType: string;
      metadata?: { title?: string; author?: string; language?: string };
    }) => uploadBook(fileUri, fileName, mimeType, metadata),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['books'] });
    },
  });
}

export function useSetVoiceMap(bookId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      voiceMap,
      options,
    }: {
      voiceMap: Omit<VoiceMap, 'book_id'>;
      options?: { initial?: boolean; update?: boolean };
    }) => setVoiceMap(bookId, voiceMap, options),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ['personas', bookId] });
      qc.invalidateQueries({ queryKey: ['voiceMap', bookId] });
      qc.invalidateQueries({ queryKey: ['bookStatus', bookId] });
      qc.invalidateQueries({ queryKey: ['pipelineStatus', bookId] });
    },
  });
}

// ── Hybrid Pipeline ─────────────────────────────────────────────────────

export function usePipelineStatus(bookId: string | undefined) {
  return useQuery({
    queryKey: ['pipelineStatus', bookId],
    queryFn: () => getPipelineStatus(bookId!),
    enabled: !!bookId,
    refetchInterval: (query) => {
      const stages = query.state.data?.stages;
      if (!stages) return false;
      const processing = stages.some(
        (s) =>
          s.status === 'in_progress' || s.status === 'waiting_for_mapping',
      );
      return processing ? 2_000 : false;
    },
  });
}

export function usePersonas(bookId: string | undefined) {
  return useQuery({
    queryKey: ['personas', bookId],
    queryFn: () => getPersonas(bookId!),
    enabled: !!bookId,
    retry: 3,
    retryDelay: (i) => Math.min(1_000 * 2 ** i, 3_000),
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data && data.unmapped.length > 0) return 2_000;
      return false;
    },
  });
}
