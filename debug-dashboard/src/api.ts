import type {
  BookMetadata,
  HealthResponse,
  PersonaDiscovery,
  PipelineStatus,
  ProcessingStatus,
  ProvidersResponse,
  Segment,
  SmokeVisibilityResponse,
  SynthJob,
  AudioArtifactValidation,
  PlaybackEvent,
  UserProgress,
  LiveEvent,
} from './types';

const env = import.meta.env as Record<string, string | undefined>;
const configuredOrigin = env.VITE_TWELVEREADER_API_URL;
export const API_ORIGIN = (configuredOrigin && configuredOrigin.trim().length > 0 ? configuredOrigin : window.location.origin).replace(/\/$/, '');
const API_BASE = `${API_ORIGIN}/api/v1`;

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: `${response.status} ${response.statusText}` }));
    throw new Error(body.error || `${response.status} ${response.statusText}`);
  }
  return response.json() as Promise<T>;
}

export async function getHealth(): Promise<HealthResponse> {
  return request<HealthResponse>(`${API_ORIGIN}/health`);
}

export async function getProviders(): Promise<ProvidersResponse> {
  return request<ProvidersResponse>(`${API_BASE}/providers`);
}

export async function getBooks(): Promise<BookMetadata[]> {
  return request<BookMetadata[]>(`${API_BASE}/books`);
}

export async function getBook(bookId: string): Promise<BookMetadata> {
  return request<BookMetadata>(`${API_BASE}/books/${bookId}`);
}

export async function getBookStatus(bookId: string): Promise<ProcessingStatus> {
  return request<ProcessingStatus>(`${API_BASE}/books/${bookId}/status`);
}

export async function getSegments(bookId: string): Promise<Segment[]> {
  return request<Segment[]>(`${API_BASE}/books/${bookId}/segments`);
}

export async function getPipelineStatus(bookId: string): Promise<PipelineStatus> {
  return request<PipelineStatus>(`${API_BASE}/books/${bookId}/pipeline/status`);
}

export async function getPersonas(bookId: string): Promise<PersonaDiscovery> {
  return request<PersonaDiscovery>(`${API_BASE}/books/${bookId}/personas`);
}

export async function fetchBookStream(bookId: string): Promise<Segment[]> {
  const response = await fetch(`${API_BASE}/books/${bookId}/stream`);
  if (!response.ok) throw new Error(`stream request failed: ${response.status}`);
  const text = await response.text();
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line) as Segment);
}

export async function getSynthJobs(bookId: string): Promise<SynthJob[]> {
  const data = await request<{ jobs: SynthJob[] }>(`${API_BASE}/debug/books/${bookId}/synth-jobs`);
  return data.jobs || [];
}

export async function getAudioValidation(bookId: string): Promise<AudioArtifactValidation[]> {
  const data = await request<{ artifacts: AudioArtifactValidation[] }>(`${API_BASE}/debug/books/${bookId}/audio-validation`);
  return data.artifacts || [];
}

export async function getPlaybackEvents(bookId: string): Promise<PlaybackEvent[]> {
  const data = await request<{ events: PlaybackEvent[] }>(`${API_BASE}/debug/books/${bookId}/playback-events`);
  return data.events || [];
}

export async function getUserProgress(bookId: string): Promise<UserProgress> {
  return request<UserProgress>(`${API_BASE}/debug/books/${bookId}/user-progress`);
}

export async function getDebugEvents(bookId?: string): Promise<LiveEvent[]> {
  const url = bookId ? `${API_BASE}/debug/books/${bookId}/events` : `${API_BASE}/debug/events`;
  const data = await request<{ events: LiveEvent[] }>(url);
  return (data.events || []).map((event) => ({
    ...event,
    at: event.at || event.created_at || new Date().toISOString(),
    bookId: event.bookId || event.book_id,
    segmentId: event.segmentId || event.segment_id,
  }));
}

export async function getReadinessSmoke(): Promise<SmokeVisibilityResponse> {
  return request<SmokeVisibilityResponse>(`${API_BASE}/debug/readiness/smoke`);
}
