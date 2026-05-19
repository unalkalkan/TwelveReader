import {
  BookMetadataSchema,
  ProcessingStatusSchema,
  SegmentSchema,
  VoiceMapSchema,
  VoicesResponseSchema,
  VoicePreviewResponseSchema,
  DefaultVoiceSchema,
  ServerInfoSchema,
  V1ServerInfoSchema,
  ProvidersSchema,
  PersonaDiscoverySchema,
  PipelineStatusSchema,
  UserProfileResponseSchema,
  type BookMetadata,
  type ProcessingStatus,
  type Segment,
  type VoiceMap,
  type VoicesResponse,
  type VoicePreviewResponse,
  type DefaultVoice,
  type ServerInfo,
  type V1ServerInfo,
  type Providers,
  type PersonaDiscovery,
  type PipelineStatus,
  type UserProfileResponse,
} from '../types/api';

/**
 * Mutable API base reference. Updated by ServerConfigProvider on mount / server change.
 * Falls back to env var or inferred origin if not set externally.
 */
let _apiBaseOverride: string | null = null;

function resolveApiBase(): string {
  if (_apiBaseOverride) return _apiBaseOverride;

  const configuredApiUrl =
    (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
      ?.env?.EXPO_PUBLIC_API_URL;
  const inferredApiUrl =
    typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
  return (configuredApiUrl && configuredApiUrl.trim().length > 0
    ? configuredApiUrl
    : inferredApiUrl
  ).replace(/\/+$/, '') + '/api/v1';
}

/** Override the API base URL. Called by ServerConfigProvider when user selects a server. */
export function setApiBase(base: string): void {
  _apiBaseOverride = base.replace(/\/+$/, '');
}

/** Synchronously read the current API base. Used by auth module to stay in sync. */
export function resolveApiBaseSync(): string {
  return resolveApiBase();
}

/** Validate a candidate server URL by calling /api/v1/server-info directly. */
export async function validateServerUrl(
  baseUrl: string,
): Promise<V1ServerInfo> {
  const url = `${baseUrl.replace(/\/+$/, '')}/api/v1/server-info`;
  const response = await fetch(url, { method: 'GET' });

  if (!response.ok) {
    throw new Error(
      `Server validation failed: HTTP ${response.status} from ${url}`,
    );
  }

  const data = await response.json();
  return V1ServerInfoSchema.parse(data);
}

// ── helpers ─────────────────────────────────────────────────────────────

/**
 * Flag to prevent infinite retry loops when a 401 triggers refresh which itself gets a 401.
 */
let _refreshing = false;

/**
 * Fetch with Bearer token attachment and auto-refresh on 401.
 * Used by both apiRequest and direct fetch calls (upload, delete).
 */
async function authenticatedFetch(
  url: string,
  options?: RequestInit,
): Promise<Response> {
  // Attach Bearer token if available
  const auth = await import('./auth');
  let token: string | null = null;
  try {
    token = await auth.getSessionToken();
  } catch {
    // ignore — proceed without auth
  }

  const headers: Record<string, string> = {
    ...options?.headers as Record<string, string>,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  let response = await fetch(url, {
    ...options,
    headers,
  });

  // Auto-refresh on 401 (but only once per request chain)
  if (response.status === 401 && !_refreshing) {
    _refreshing = true;
    try {
      const newToken = await auth.attemptRefresh();
      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`;
        response = await fetch(url, {
          ...options,
          headers,
        });
      }
    } finally {
      _refreshing = false;
    }
  }

  return response;
}

async function apiRequest<T>(
  url: string,
  options?: RequestInit,
  schema?: { parse: (d: unknown) => T },
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options?.headers as Record<string, string>,
  };

  let response = await authenticatedFetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Request failed',
      code: 'UNKNOWN_ERROR',
    }));
    throw new Error(error.error || 'Request failed');
  }

  const data = await response.json();
  return schema ? schema.parse(data) : data;
}

// ── Server Info ─────────────────────────────────────────────────────────

export async function getServerInfo(): Promise<ServerInfo> {
  return apiRequest<ServerInfo>(`${resolveApiBase()}/info`, {}, ServerInfoSchema);
}

// ── Providers ───────────────────────────────────────────────────────────

export async function getProviders(): Promise<Providers> {
  return apiRequest<Providers>(`${resolveApiBase()}/providers`, {}, ProvidersSchema);
}

// ── Voices ──────────────────────────────────────────────────────────────

export async function getVoices(provider?: string): Promise<VoicesResponse> {
  const params = new URLSearchParams();
  if (provider) params.append('provider', provider);
  const qs = params.toString();
  return apiRequest<VoicesResponse>(
    `${resolveApiBase()}/voices${qs ? `?${qs}` : ''}`,
    {},
    VoicesResponseSchema,
  );
}

export async function previewVoice(params: {
  provider: string;
  voice_id: string;
  text: string;
  language?: string;
  voice_description?: string;
}): Promise<VoicePreviewResponse> {
  return apiRequest<VoicePreviewResponse>(
    `${resolveApiBase()}/voices/preview`,
    {
      method: 'POST',
      body: JSON.stringify(params),
    },
    VoicePreviewResponseSchema,
  );
}

// ── Default Voice ───────────────────────────────────────────────────────

export async function getDefaultVoice(): Promise<DefaultVoice> {
  return apiRequest<DefaultVoice>(
    `${resolveApiBase()}/voices/default`,
    {},
    DefaultVoiceSchema,
  );
}

export async function setDefaultVoice(payload: {
  provider: string;
  voice_id: string;
  language?: string;
  voice_description?: string;
}): Promise<DefaultVoice> {
  return apiRequest<DefaultVoice>(
    `${resolveApiBase()}/voices/default`,
    {
      method: 'PUT',
      body: JSON.stringify(payload),
    },
    DefaultVoiceSchema,
  );
}

// ── Books ───────────────────────────────────────────────────────────────

export type FileSource =
  | { uri: string; name: string; type: string }
  | { blob: Blob; name: string; type: string };

async function appendFileToFormData(formData: FormData, source: FileSource): Promise<void> {
  if ('blob' in source) {
    formData.append('file', source.blob, source.name);
    return;
  }

  if (typeof window !== 'undefined') {
    const response = await fetch(source.uri);
    if (!response.ok) {
      throw new Error(`Failed to read selected file: ${response.status}`);
    }
    const blob = await response.blob();
    const typedBlob = blob.type === source.type ? blob : blob.slice(0, blob.size, source.type);
    formData.append('file', typedBlob, source.name);
    return;
  }

  formData.append('file', {
    uri: source.uri,
    name: source.name,
    type: source.type,
  } as any);
}

function appendMetadata(
  formData: FormData,
  metadata?: { title?: string; author?: string; language?: string },
): void {
  if (metadata?.title) formData.append('title', metadata.title);
  if (metadata?.author) formData.append('author', metadata.author);
  if (metadata?.language) formData.append('language', metadata.language);
}

export async function uploadBook(
  fileSource: FileSource,
  metadata?: { title?: string; author?: string; language?: string },
): Promise<BookMetadata> {
  const formData = new FormData();
  await appendFileToFormData(formData, fileSource);
  appendMetadata(formData, metadata);

  // Use authenticatedFetch but DON'T set Content-Type (multipart boundary is auto-set)
  const authHeaders: Record<string, string> = {};
  try {
    const auth = await import('./auth');
    const token = await auth.getSessionToken();
    if (token) {
      authHeaders['Authorization'] = `Bearer ${token}`;
    }
  } catch {
    // ignore
  }

  const response = await fetch(`${resolveApiBase()}/books`, {
    method: 'POST',
    body: formData,
    headers: authHeaders,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Upload failed',
      code: 'UPLOAD_ERROR',
    }));
    throw new Error(error.error || 'Upload failed');
  }

  const data = await response.json();
  return BookMetadataSchema.parse(data);
}

export async function getBooks(): Promise<BookMetadata[]> {
  return apiRequest<BookMetadata[]>(
    `${resolveApiBase()}/books`,
    {},
    BookMetadataSchema.array() as any,
  );
}

export async function getBook(bookId: string): Promise<BookMetadata> {
  return apiRequest<BookMetadata>(
    `${resolveApiBase()}/books/${bookId}`,
    {},
    BookMetadataSchema,
  );
}

export async function getBookStatus(
  bookId: string,
): Promise<ProcessingStatus> {
  return apiRequest<ProcessingStatus>(
    `${resolveApiBase()}/books/${bookId}/status`,
    {},
    ProcessingStatusSchema,
  );
}

export async function getBookSegments(bookId: string): Promise<Segment[]> {
  const data = await apiRequest<any[]>(`${resolveApiBase()}/books/${bookId}/segments`);
  return data.map((item) => SegmentSchema.parse(item));
}

export async function deleteBook(bookId: string): Promise<void> {
  const response = await authenticatedFetch(
    `${resolveApiBase()}/books/${bookId}`,
    { method: 'DELETE' },
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Delete failed',
    }));
    throw new Error(error.error || 'Delete failed');
  }
}

// ── Voice Map ───────────────────────────────────────────────────────────

export async function getVoiceMap(bookId: string): Promise<VoiceMap> {
  return apiRequest<VoiceMap>(
    `${resolveApiBase()}/books/${bookId}/voice-map`,
    {},
    VoiceMapSchema,
  );
}

export async function setVoiceMap(
  bookId: string,
  voiceMap: Omit<VoiceMap, 'book_id'>,
  options?: { initial?: boolean; update?: boolean },
): Promise<VoiceMap> {
  const params = new URLSearchParams();
  if (options?.initial) params.append('initial', 'true');
  if (options?.update) params.append('update', 'true');
  const qs = params.toString();

  return apiRequest<VoiceMap>(
    `${resolveApiBase()}/books/${bookId}/voice-map${qs ? `?${qs}` : ''}`,
    {
      method: 'POST',
      body: JSON.stringify(voiceMap),
    },
    VoiceMapSchema,
  );
}

// ── Hybrid Pipeline ─────────────────────────────────────────────────────

export async function getPipelineStatus(
  bookId: string,
): Promise<PipelineStatus> {
  return apiRequest<PipelineStatus>(
    `${resolveApiBase()}/books/${bookId}/pipeline/status`,
    {},
    PipelineStatusSchema,
  );
}

export async function getPersonas(bookId: string): Promise<PersonaDiscovery> {
  return apiRequest<PersonaDiscovery>(
    `${resolveApiBase()}/books/${bookId}/personas`,
    {},
    PersonaDiscoverySchema,
  );
}

// ── URLs ────────────────────────────────────────────────────────────────

export function getStreamUrl(bookId: string, after?: string): string {
  const params = new URLSearchParams();
  if (after) params.append('after', after);
  return `${resolveApiBase()}/books/${bookId}/stream${params.toString() ? `?${params}` : ''}`;
}

export function getDownloadUrl(bookId: string): string {
  return `${resolveApiBase()}/books/${bookId}/download`;
}

export function getAudioUrl(bookId: string, segmentId: string): string {
  return `${resolveApiBase()}/books/${bookId}/audio/${segmentId}`;
}

// ── Stream (NDJSON) ─────────────────────────────────────────────────────

export interface StreamSegment {
  id: string;
  book_id: string;
  chapter: string;
  toc_path: string[];
  text: string;
  language: string;
  person: string;
  voice_description: string;
  voice_id?: string;
  processing: {
    segmenter_version: string;
    generated_at: string;
    tts_provider?: string;
  };
  timestamps?: {
    precision: 'word' | 'sentence';
    items: { word: string; start: number; end: number }[];
  };
  source_context?: {
    prev_paragraph_id?: string;
    next_paragraph_id?: string;
  };
  audio_url: string;
}

/**
 * Fetch segments via the NDJSON /stream endpoint which includes audio_url
 * and timestamps that are not present in the regular /segments list.
 */
export async function fetchBookStream(
  bookId: string,
  after?: string,
): Promise<StreamSegment[]> {
  const url = getStreamUrl(bookId, after);
  const response = await authenticatedFetch(url);

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Stream request failed',
      code: 'STREAM_ERROR',
    }));
    throw new Error(error.error || 'Stream request failed');
  }

  const text = await response.text();
  const lines = text.split('\n').filter((line) => line.trim().length > 0);
  return lines.map((line) => JSON.parse(line) as StreamSegment);
}

// ── Upload with progress callback ───────────────────────────────────────

export function uploadBookWithProgress(
  fileSource: FileSource,
  metadata?: { title?: string; author?: string; language?: string },
  onProgress?: (percent: number) => void,
): Promise<BookMetadata> {
  return new Promise(async (resolve, reject) => {
    const formData = new FormData();

    // Get auth token before starting upload
    let authToken: string | null = null;
    try {
      const auth = await import('./auth');
      authToken = await auth.getSessionToken();
    } catch {
      // ignore
    }

    appendFileToFormData(formData, fileSource)
      .then(() => {
        appendMetadata(formData, metadata);

        const xhr = new XMLHttpRequest();
        xhr.open('POST', `${resolveApiBase()}/books`);

        // Attach auth token if available
        if (authToken) {
          xhr.setRequestHeader('Authorization', `Bearer ${authToken}`);
        }

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable && onProgress) {
            onProgress(Math.round((event.loaded / event.total) * 100));
          }
        };

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              const data = JSON.parse(xhr.responseText);
              resolve(BookMetadataSchema.parse(data));
            } catch (err: any) {
              reject(new Error('Invalid response from server'));
            }
          } else {
            try {
              const error = JSON.parse(xhr.responseText);
              reject(new Error(error.error || 'Upload failed'));
            } catch {
              reject(new Error('Upload failed'));
            }
          }
        };

        xhr.onerror = () => reject(new Error('Network error during upload'));
        xhr.send(formData);
      })
      .catch(reject);
  });
}

// ── User Profile (Milestone 2) ────────────────────────────────────────

export async function getUserProfile(): Promise<UserProfileResponse> {
  return apiRequest<UserProfileResponse>(
    `${resolveApiBase()}/user/profile`,
    {},
    UserProfileResponseSchema,
  );
}
