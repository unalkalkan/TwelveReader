import {
  BookMetadataSchema,
  ProcessingStatusSchema,
  SegmentSchema,
  VoiceMapSchema,
  VoicesResponseSchema,
  VoicePreviewResponseSchema,
  ServerInfoSchema,
  ProvidersSchema,
  PersonaDiscoverySchema,
  PipelineStatusSchema,
  type BookMetadata,
  type ProcessingStatus,
  type Segment,
  type VoiceMap,
  type VoicesResponse,
  type VoicePreviewResponse,
  type ServerInfo,
  type Providers,
  type PersonaDiscovery,
  type PipelineStatus,
} from '../types/api';

/** Configured via EXPO_PUBLIC_API_URL env var; falls back to localhost for dev. */
const configuredApiUrl =
  (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
    ?.env?.EXPO_PUBLIC_API_URL;
const inferredApiUrl =
  typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
const normalizedApiBase = (configuredApiUrl && configuredApiUrl.trim().length > 0
  ? configuredApiUrl
  : inferredApiUrl
).replace(/\/$/, '');
const API_BASE = `${normalizedApiBase}/api/v1`;

// ── helpers ─────────────────────────────────────────────────────────────

async function apiRequest<T>(
  url: string,
  options?: RequestInit,
  schema?: { parse: (d: unknown) => T },
): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
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
  return apiRequest<ServerInfo>(`${API_BASE}/info`, {}, ServerInfoSchema);
}

// ── Providers ───────────────────────────────────────────────────────────

export async function getProviders(): Promise<Providers> {
  return apiRequest<Providers>(`${API_BASE}/providers`, {}, ProvidersSchema);
}

// ── Voices ──────────────────────────────────────────────────────────────

export async function getVoices(provider?: string): Promise<VoicesResponse> {
  const params = new URLSearchParams();
  if (provider) params.append('provider', provider);
  const qs = params.toString();
  return apiRequest<VoicesResponse>(
    `${API_BASE}/voices${qs ? `?${qs}` : ''}`,
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
    `${API_BASE}/voices/preview`,
    {
      method: 'POST',
      body: JSON.stringify(params),
    },
    VoicePreviewResponseSchema,
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

  const response = await fetch(`${API_BASE}/books`, {
    method: 'POST',
    body: formData,
    // Let fetch set the multipart content-type header automatically
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
    `${API_BASE}/books`,
    {},
    BookMetadataSchema.array() as any,
  );
}

export async function getBook(bookId: string): Promise<BookMetadata> {
  return apiRequest<BookMetadata>(
    `${API_BASE}/books/${bookId}`,
    {},
    BookMetadataSchema,
  );
}

export async function getBookStatus(
  bookId: string,
): Promise<ProcessingStatus> {
  return apiRequest<ProcessingStatus>(
    `${API_BASE}/books/${bookId}/status`,
    {},
    ProcessingStatusSchema,
  );
}

export async function getBookSegments(bookId: string): Promise<Segment[]> {
  const data = await apiRequest<any[]>(`${API_BASE}/books/${bookId}/segments`);
  return data.map((item) => SegmentSchema.parse(item));
}

export async function deleteBook(bookId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/books/${bookId}`, {
    method: 'DELETE',
  });

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
    `${API_BASE}/books/${bookId}/voice-map`,
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
    `${API_BASE}/books/${bookId}/voice-map${qs ? `?${qs}` : ''}`,
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
    `${API_BASE}/books/${bookId}/pipeline/status`,
    {},
    PipelineStatusSchema,
  );
}

export async function getPersonas(bookId: string): Promise<PersonaDiscovery> {
  return apiRequest<PersonaDiscovery>(
    `${API_BASE}/books/${bookId}/personas`,
    {},
    PersonaDiscoverySchema,
  );
}

// ── URLs ────────────────────────────────────────────────────────────────

export function getStreamUrl(bookId: string, after?: string): string {
  const params = new URLSearchParams();
  if (after) params.append('after', after);
  return `${API_BASE}/books/${bookId}/stream${params.toString() ? `?${params}` : ''}`;
}

export function getDownloadUrl(bookId: string): string {
  return `${API_BASE}/books/${bookId}/download`;
}

export function getAudioUrl(bookId: string, segmentId: string): string {
  return `${API_BASE}/books/${bookId}/audio/${segmentId}`;
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
  const response = await fetch(url);

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
  return new Promise((resolve, reject) => {
    const formData = new FormData();

    appendFileToFormData(formData, fileSource)
      .then(() => {
        appendMetadata(formData, metadata);

        const xhr = new XMLHttpRequest();
        xhr.open('POST', `${API_BASE}/books`);

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
