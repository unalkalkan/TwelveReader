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

function authHeaders(token?: string | null): Record<string, string> {
  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

async function request<T>(url: string, init?: RequestInit, token?: string | null): Promise<T> {
  const response = await fetch(url, {
    ...init,
    headers: {
      ...authHeaders(token),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: `${response.status} ${response.statusText}` }));
    throw Object.assign(new Error(body.error || `${response.status} ${response.statusText}`), { status: response.status } as { status: number });
  }
  return response.json() as Promise<T>;
}

// --- Auth API functions ---

export async function apiRequestMagicLink(email: string): Promise<{ message: string }> {
  return request<{ message: string }>(`${API_BASE}/auth/request`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  });
}

export interface AuthVerifyResponse {
  user: Record<string, unknown>;
  session_token: string;
  refresh_token: string;
}

export async function apiVerifyMagicLink(token: string): Promise<AuthVerifyResponse> {
  return request<AuthVerifyResponse>(`${API_BASE}/auth/verify?token=${encodeURIComponent(token)}`);
}

export async function apiMe(sessionToken: string): Promise<Record<string, unknown>> {
  return request<Record<string, unknown>>(`${API_BASE}/auth/me`, undefined, sessionToken);
}

export interface RefreshSessionResponse {
  session_token: string;
  refresh_token: string;
}

export async function apiRefreshSession(refreshToken: string): Promise<RefreshSessionResponse> {
  return request<RefreshSessionResponse>(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

// --- Health ---

export async function getHealth(): Promise<HealthResponse> {
  return request<HealthResponse>(`${API_ORIGIN}/health`);
}

// --- Providers ---

export async function getProviders(sessionToken?: string | null): Promise<ProvidersResponse> {
  return request<ProvidersResponse>(`${API_BASE}/providers`, undefined, sessionToken);
}

// --- Books ---

export async function getBooks(sessionToken?: string | null): Promise<BookMetadata[]> {
  return request<BookMetadata[]>(`${API_BASE}/books`, undefined, sessionToken);
}

export async function getBook(bookId: string, sessionToken?: string | null): Promise<BookMetadata> {
  return request<BookMetadata>(`${API_BASE}/books/${bookId}`, undefined, sessionToken);
}

export async function getBookStatus(bookId: string, sessionToken?: string | null): Promise<ProcessingStatus> {
  return request<ProcessingStatus>(`${API_BASE}/books/${bookId}/status`, undefined, sessionToken);
}

export async function getSegments(bookId: string, sessionToken?: string | null): Promise<Segment[]> {
  return request<Segment[]>(`${API_BASE}/books/${bookId}/segments`, undefined, sessionToken);
}

export async function getPipelineStatus(bookId: string, sessionToken?: string | null): Promise<PipelineStatus> {
  return request<PipelineStatus>(`${API_BASE}/books/${bookId}/pipeline/status`, undefined, sessionToken);
}

export async function getPersonas(bookId: string, sessionToken?: string | null): Promise<PersonaDiscovery> {
  return request<PersonaDiscovery>(`${API_BASE}/books/${bookId}/personas`, undefined, sessionToken);
}

export async function fetchBookStream(bookId: string, sessionToken?: string | null): Promise<Segment[]> {
  const response = await fetch(`${API_BASE}/books/${bookId}/stream`, {
    headers: authHeaders(sessionToken),
  });
  if (!response.ok) throw new Error(`stream request failed: ${response.status}`);
  const text = await response.text();
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line) as Segment);
}

// --- Debug endpoints (require admin role) ---

export async function getSynthJobs(bookId: string, sessionToken?: string | null): Promise<SynthJob[]> {
  const data = await request<{ jobs: SynthJob[] }>(`${API_BASE}/debug/books/${bookId}/synth-jobs`, undefined, sessionToken);
  return data.jobs || [];
}

export async function getAudioValidation(bookId: string, sessionToken?: string | null): Promise<AudioArtifactValidation[]> {
  const data = await request<{ artifacts: AudioArtifactValidation[] }>(`${API_BASE}/debug/books/${bookId}/audio-validation`, undefined, sessionToken);
  return data.artifacts || [];
}

export async function getPlaybackEvents(bookId: string, sessionToken?: string | null): Promise<PlaybackEvent[]> {
  const data = await request<{ events: PlaybackEvent[] }>(`${API_BASE}/debug/books/${bookId}/playback-events`, undefined, sessionToken);
  return data.events || [];
}

export async function getUserProgress(bookId: string, sessionToken?: string | null): Promise<UserProgress> {
  return request<UserProgress>(`${API_BASE}/debug/books/${bookId}/user-progress`, undefined, sessionToken);
}

export async function getDebugEvents(bookId?: string, sessionToken?: string | null): Promise<LiveEvent[]> {
  const url = bookId ? `${API_BASE}/debug/books/${bookId}/events` : `${API_BASE}/debug/events`;
  const data = await request<{ events: LiveEvent[] }>(url, undefined, sessionToken);
  return (data.events || []).map((event) => ({
    ...event,
    at: event.at || event.created_at || new Date().toISOString(),
    bookId: event.bookId || event.book_id,
    segmentId: event.segmentId || event.segment_id,
  }));
}

export async function getReadinessSmoke(sessionToken?: string | null): Promise<SmokeVisibilityResponse> {
  return request<SmokeVisibilityResponse>(`${API_BASE}/debug/readiness/smoke`, undefined, sessionToken);
}
