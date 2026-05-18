export type BookStatus =
  | 'uploaded'
  | 'parsing'
  | 'segmenting'
  | 'voice_mapping'
  | 'ready'
  | 'synthesizing'
  | 'synthesized'
  | 'synthesis_error'
  | 'error';

export interface BookMetadata {
  id: string;
  title: string;
  author: string;
  language: string;
  uploaded_at: string;
  status: BookStatus | string;
  orig_format: string;
  error?: string;
  total_chapters: number;
  total_segments: number;
  total_paragraphs?: number;
  segmented_paragraphs?: number;
  synthesized_segments?: number;
  discovered_personas?: string[];
  unmapped_personas?: string[];
  pending_segment_count?: number;
  waiting_for_mapping?: boolean;
}

export interface ProcessingStatus {
  book_id: string;
  status: BookStatus | string;
  stage: string;
  progress: number;
  total_chapters: number;
  parsed_chapters: number;
  total_segments: number;
  total_paragraphs?: number;
  segmented_paragraphs?: number;
  synthesized_segments?: number;
  error?: string;
  updated_at: string;
}

export interface Segment {
  id: string;
  book_id: string;
  chapter: string;
  toc_path: string[];
  text: string;
  language: string;
  person: string;
  voice_description: string;
  voice_id?: string;
  audio_stale?: boolean;
  stale_voice_id?: string;
  timestamps?: {
    precision: 'word' | 'sentence' | string;
    items: Array<{ word: string; start: number; end: number }>;
  };
  processing?: {
    segmenter_version?: string;
    generated_at?: string;
    tts_provider?: string;
  };
  source_context?: {
    prev_paragraph_id?: string;
    next_paragraph_id?: string;
  };
  audio_url?: string;
}

export interface PipelineStage {
  stage: string;
  status: string;
  current: number;
  total: number;
  message?: string;
}

export interface PipelineStatus {
  book_id: string;
  status?: string;
  stages: PipelineStage[];
  updated_at?: string;
}

export interface PersonaDiscovery {
  discovered: string[];
  mapped: Record<string, string>;
  unmapped: string[];
  pending_segments: number;
}

export interface HealthResponse {
  status: string;
  timestamp?: string;
  version?: string;
  checks?: Record<string, { status: string; error?: string }>;
}

export interface ProvidersResponse {
  llm?: string[];
  tts?: string[];
  ocr?: string[];
}

export type SegmentReadState = 'not_opened' | 'read' | 'current' | 'skipped';
export type SegmentListenState = 'not_attempted' | 'partial' | 'completed' | 'failed' | 'stuck';
export type SegmentSynthState = 'not_created' | 'queued' | 'running' | 'completed' | 'failed' | 'retrying' | 'stale';
export type SegmentAudioState = 'missing' | 'attached' | 'stale' | 'invalid' | 'playback_failed';


export interface SynthJob {
  id: string;
  book_id: string;
  segment_id: string;
  status: string;
  provider?: string;
  voice_id?: string;
  output_path?: string;
  output_format?: string;
  output_bytes?: number;
  retry_count: number;
  error?: string;
  updated_at: string;
}

export interface AudioArtifactValidation {
  book_id: string;
  segment_id: string;
  status: string;
  format?: string;
  path?: string;
  bytes?: number;
  content_type?: string;
  error?: string;
  checked_at: string;
}

export interface PlaybackEvent {
  id: string;
  book_id: string;
  segment_id?: string;
  user_id: string;
  event_type: string;
  playback_position_sec?: number;
  duration_sec?: number;
  success?: boolean;
  error?: string;
  client?: string;
  device?: string;
  created_at: string;
}

export interface UserProgress {
  book_id: string;
  user_id: string;
  journey_state: string;
  can_read: boolean;
  can_listen_all: boolean;
  last_opened_segment_id?: string;
  last_read_segment_id?: string;
  last_listened_segment_id?: string;
  stuck_segment_id?: string;
  playback_failures: number;
  completed_listen_segments: number;
  total_segments: number;
  updated_at: string;
}

export interface SegmentInspection {
  index: number;
  segment: Segment;
  synthState: SegmentSynthState;
  audioState: SegmentAudioState;
  readState: SegmentReadState;
  listenState: SegmentListenState;
  audioDurationSec?: number;
  playbackFailures: number;
  retryCount: number;
  lastUserEvent?: string;
  blocker?: string;
  synthJob?: SynthJob;
  audioValidation?: AudioArtifactValidation;
}

export interface BookJourney {
  book: BookMetadata;
  status?: ProcessingStatus;
  pipeline?: PipelineStatus;
  personas?: PersonaDiscovery;
  userProgress?: UserProgress;
  playbackEvents?: PlaybackEvent[];
  segments: SegmentInspection[];
  readinessScore: number;
  textReadyCount: number;
  audioReadyCount: number;
  staleAudioCount: number;
  failedAudioCount: number;
  userReadSegment: number;
  userListenedSegment: number;
  blocker?: string;
  perspective: string;
}

export interface LiveEvent {
  id: string;
  at: string;
  created_at?: string;
  scope: 'system' | 'book' | 'segment' | 'synth' | 'user' | 'health';
  severity: 'info' | 'success' | 'warning' | 'danger';
  title: string;
  detail: string;
  bookId?: string;
  segmentId?: string;
  book_id?: string;
  segment_id?: string;
}

// Readiness smoke check types (Milestone 0, Work 0.5)
export interface SmokeCheckResult {
  name: string;
  path: string;
  status: 'ok' | 'warning' | 'error';
  http_code: number;
  latency_ms: number;
  error?: string;
  data?: Record<string, unknown>;
}

export interface SmokeVisibilityResponse {
  timestamp: string;
  checks: SmokeCheckResult[];
  overall: 'all_ok' | 'degraded' | 'unhealthy';
}
