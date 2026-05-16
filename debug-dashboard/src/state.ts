import type {
  BookJourney,
  BookMetadata,
  HealthResponse,
  LiveEvent,
  PersonaDiscovery,
  PipelineStatus,
  ProcessingStatus,
  Segment,
  SegmentInspection,
} from './types';

const nowIso = () => new Date().toISOString();

export function makeDemoBooks(tick = 0): BookJourney[] {
  const uploaded = new Date(Date.now() - 1000 * 60 * 42).toISOString();
  const book: BookMetadata = {
    id: 'book_demo_live_001',
    title: 'The Blue Library Incident',
    author: 'TwelveReader Demo',
    language: 'en',
    uploaded_at: uploaded,
    status: tick % 9 > 5 ? 'synthesis_error' : 'synthesizing',
    orig_format: 'pdf',
    total_chapters: 8,
    total_segments: 58,
    total_paragraphs: 174,
    segmented_paragraphs: 174,
    synthesized_segments: Math.min(45, 38 + (tick % 8)),
    discovered_personas: ['narrator', 'Ada', 'Inspector Vale'],
    unmapped_personas: [],
    pending_segment_count: 0,
    waiting_for_mapping: false,
  };

  const status: ProcessingStatus = {
    book_id: book.id,
    status: book.status,
    stage: book.status,
    progress: Math.round(((book.synthesized_segments || 0) / (book.total_segments || 1)) * 100),
    total_chapters: 8,
    parsed_chapters: 8,
    total_segments: 58,
    total_paragraphs: 174,
    segmented_paragraphs: 174,
    synthesized_segments: book.synthesized_segments,
    updated_at: nowIso(),
    error: book.status === 'synthesis_error' ? 'TTS timeout on segment seg_00042 after 3 retries' : undefined,
  };

  const pipeline: PipelineStatus = {
    book_id: book.id,
    updated_at: nowIso(),
    stages: [
      { stage: 'uploaded', status: 'completed', current: 1, total: 1 },
      { stage: 'parsing', status: 'completed', current: 8, total: 8 },
      { stage: 'segmenting', status: 'completed', current: 174, total: 174 },
      { stage: 'voice_mapping', status: 'completed', current: 3, total: 3 },
      { stage: 'synthesizing', status: 'in_progress', current: book.synthesized_segments || 0, total: 58 },
      { stage: 'ready', status: 'pending', current: 0, total: 1 },
    ],
  };

  const personas: PersonaDiscovery = {
    discovered: ['narrator', 'Ada', 'Inspector Vale'],
    mapped: { narrator: 'aiden', Ada: 'bella', 'Inspector Vale': 'derek' },
    unmapped: [],
    pending_segments: 0,
  };

  const segments: SegmentInspection[] = Array.from({ length: 58 }, (_, i) => {
    const index = i + 1;
    const synthesized = index <= (book.synthesized_segments || 0);
    const stale = [12, 13, 14].includes(index);
    const failed = [42, 43, 49].includes(index);
    const segment: Segment = {
      id: `seg_${String(index).padStart(5, '0')}`,
      book_id: book.id,
      chapter: `chapter_${Math.ceil(index / 8)}`,
      toc_path: [`Chapter ${Math.ceil(index / 8)}`, index % 8 === 1 ? 'Opening' : 'Scene'],
      text: `Segment ${index} text preview. This row represents extracted readable content and its generated narration state for inspection.`,
      language: 'en',
      person: index % 7 === 0 ? 'Inspector Vale' : index % 5 === 0 ? 'Ada' : 'narrator',
      voice_description: 'clear audiobook narration',
      voice_id: synthesized ? (index % 5 === 0 ? 'bella' : 'aiden') : undefined,
      audio_stale: stale,
      stale_voice_id: stale ? 'old-default' : undefined,
      audio_url: synthesized ? `/api/v1/books/${book.id}/audio/seg_${String(index).padStart(5, '0')}` : undefined,
      processing: {
        segmenter_version: 'hybrid-v1',
        generated_at: new Date(Date.now() - 1000 * 60 * (80 - index)).toISOString(),
        tts_provider: synthesized ? 'qwen3-tts' : undefined,
      },
    };

    return {
      index,
      segment,
      synthState: failed ? 'failed' : stale ? 'stale' : synthesized ? 'completed' : index <= (book.synthesized_segments || 0) + 3 ? 'queued' : 'not_created',
      audioState: failed ? 'playback_failed' : stale ? 'stale' : synthesized ? 'attached' : 'missing',
      readState: index < 28 ? 'read' : index === 28 ? 'current' : 'not_opened',
      listenState: failed && index === 42 ? 'stuck' : index < 24 ? 'completed' : index === 24 ? 'partial' : 'not_attempted',
      audioDurationSec: synthesized ? 32 + (index % 9) * 6 : undefined,
      playbackFailures: failed ? (index === 42 ? 3 : 1) : 0,
      retryCount: failed ? 3 : stale ? 1 : 0,
      lastUserEvent: index === 24 ? 'paused at 00:47' : index === 42 ? 'playback failed' : undefined,
      blocker: failed ? 'Playback failed after synth output was attached' : !synthesized ? 'Audio not generated yet' : stale ? 'Audio stale after remap' : undefined,
    };
  });

  return [deriveJourney(book, status, pipeline, personas, segments)];
}

export function deriveJourney(
  book: BookMetadata,
  status: ProcessingStatus | undefined,
  pipeline: PipelineStatus | undefined,
  personas: PersonaDiscovery | undefined,
  rawSegments: Segment[] | SegmentInspection[],
): BookJourney {
  const segments: SegmentInspection[] = rawSegments.map((item, idx) => {
    if ('segment' in item) return item;
    const segment = item as Segment;
    const hasAudio = Boolean(segment.audio_url || segment.timestamps || segment.voice_id);
    const stale = Boolean(segment.audio_stale);
    return {
      index: idx + 1,
      segment,
      synthState: stale ? 'stale' : hasAudio ? 'completed' : book.status === 'synthesizing' ? 'queued' : 'not_created',
      audioState: stale ? 'stale' : hasAudio ? 'attached' : 'missing',
      readState: idx === 0 ? 'current' : 'not_opened',
      listenState: 'not_attempted',
      audioDurationSec: segment.timestamps?.items?.length
        ? Math.round(segment.timestamps.items[segment.timestamps.items.length - 1].end)
        : undefined,
      playbackFailures: 0,
      retryCount: 0,
      blocker: !hasAudio ? 'Audio missing or not attached' : undefined,
    };
  });

  const total = Math.max(book.total_segments || segments.length, segments.length, 1);
  const textReadyCount = segments.length || book.total_segments || 0;
  const audioReadyCount = segments.filter((s) => s.audioState === 'attached').length || book.synthesized_segments || 0;
  const staleAudioCount = segments.filter((s) => s.audioState === 'stale').length;
  const failedAudioCount = segments.filter((s) => s.audioState === 'playback_failed' || s.synthState === 'failed').length;
  const readinessScore = Math.round(((textReadyCount / total) * 0.4 + (audioReadyCount / total) * 0.6) * 100);
  const userReadSegment = Math.max(0, ...segments.filter((s) => s.readState === 'read' || s.readState === 'current').map((s) => s.index));
  const userListenedSegment = Math.max(0, ...segments.filter((s) => s.listenState === 'completed' || s.listenState === 'partial').map((s) => s.index));

  let blocker: string | undefined;
  if (book.error || status?.error) blocker = book.error || status?.error;
  else if (failedAudioCount > 0) blocker = `${failedAudioCount} segment(s) have failed synthesis or playback`;
  else if (audioReadyCount < total && ['synthesizing', 'synthesis_error'].includes(book.status)) blocker = `Audio is available through ${audioReadyCount}/${total} segments`;
  else if (book.waiting_for_mapping || (personas?.unmapped?.length || 0) > 0) blocker = 'Waiting for persona voice mapping';

  const perspective = blocker
    ? `User can read text, but listening is blocked or incomplete: ${blocker}.`
    : 'User can read and listen normally. No journey blocker detected.';

  return {
    book,
    status,
    pipeline,
    personas,
    segments,
    readinessScore,
    textReadyCount,
    audioReadyCount,
    staleAudioCount,
    failedAudioCount,
    userReadSegment,
    userListenedSegment,
    blocker,
    perspective,
  };
}

export function buildEvents(journeys: BookJourney[], health?: HealthResponse, tick = 0): LiveEvent[] {
  const events: LiveEvent[] = [];
  if (health) {
    events.push({
      id: `health-${tick}`,
      at: nowIso(),
      scope: 'health',
      severity: health.status === 'healthy' ? 'success' : 'warning',
      title: `Backend ${health.status}`,
      detail: health.version ? `Version ${health.version}` : 'Health endpoint responded',
    });
  }

  for (const journey of journeys) {
    events.push({
      id: `${journey.book.id}-status-${tick}`,
      at: journey.status?.updated_at || nowIso(),
      scope: 'book',
      severity: journey.blocker ? 'warning' : 'success',
      title: `${journey.book.title}: ${journey.book.status}`,
      detail: journey.perspective,
      bookId: journey.book.id,
    });
    const segment = journey.segments.find((s) => s.blocker) || journey.segments[journey.userListenedSegment - 1];
    if (segment) {
      events.push({
        id: `${segment.segment.id}-${tick}`,
        at: nowIso(),
        scope: segment.playbackFailures > 0 ? 'user' : 'segment',
        severity: segment.playbackFailures > 0 ? 'danger' : segment.blocker ? 'warning' : 'info',
        title: `Segment ${segment.index}: ${segment.synthState}/${segment.audioState}`,
        detail: segment.blocker || segment.lastUserEvent || 'Segment state refreshed',
        bookId: journey.book.id,
        segmentId: segment.segment.id,
      });
    }
  }

  return events.sort((a, b) => Date.parse(b.at) - Date.parse(a.at)).slice(0, 20);
}
