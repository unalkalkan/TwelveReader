import type {
  BookMetadata,
  HealthResponse,
  PersonaDiscovery,
  PipelineStatus,
  ProcessingStatus,
  ProvidersResponse,
  Segment,
} from './types';

const env = import.meta.env as Record<string, string | undefined>;
export const API_ORIGIN = (env.VITE_TWELVEREADER_API_URL || 'http://localhost:8080').replace(/\/$/, '');
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
