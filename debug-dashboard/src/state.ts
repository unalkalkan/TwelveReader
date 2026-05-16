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
  SynthJob,
  AudioArtifactValidation,
  PlaybackEvent,
  UserProgress,
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
  debug?: { synthJobs?: SynthJob[]; audioValidations?: AudioArtifactValidation[]; playbackEvents?: PlaybackEvent[]; userProgress?: UserProgress },
): BookJourney {
  const synthBySegment = new Map((debug?.synthJobs || []).map((job) => [job.segment_id, job]));
  const audioBySegment = new Map((debug?.audioValidations || []).map((artifact) => [artifact.segment_id, artifact]));
  const playbackBySegment = new Map<string, PlaybackEvent[]>();
  for (const event of debug?.playbackEvents || []) {
    if (!event.segment_id) continue;
    playbackBySegment.set(event.segment_id, [...(playbackBySegment.get(event.segment_id) || []), event]);
  }
  // Track which segments have actually been synthesized (by audio artifact existence)
  const verifiedSynth = new Map<string, boolean>();
  if (debug?.audioValidations) {
    for (const v of debug.audioValidations) {
      verifiedSynth.set(v.segment_id, v.status === 'attached' || v.status === 'stale');
    }
  }

  // Fallback: use book.synthesized_segments count as authoritative source when audio validations aren't available
  const synthCount = book.synthesized_segments ?? 0;
  const hasVerifiedAudio = verifiedSynth.size > 0; // Only trust this if we actually got validation data

  const segments: SegmentInspection[] = rawSegments.map((item, idx) => {
    if ('segment' in item) return item;
    const segment = item as Segment;
    const stale = Boolean(segment.audio_stale);
    const synthJob = synthBySegment.get(segment.id);
    const audioValidation = audioBySegment.get(segment.id);
    const playbackEvents = playbackBySegment.get(segment.id) || [];
    const playbackFailures = playbackEvents.filter((event) => event.event_type === 'failed' || event.error).length;
    const completedListens = playbackEvents.filter((event) => event.event_type === 'complete').length;

    // Determine if audio actually exists — priority: explicit validation > timestamp proof > synthesized count fallback
    const verifiedHasAudio = verifiedSynth.get(segment.id);
    const hasTimestamps = !!(segment.timestamps && segment.timestamps.items?.length > 0);
    const isWithinSynthCount = synthCount > 0 && !hasVerifiedAudio && Boolean(segment.voice_id) && idx < synthCount;
    const hasAudio = (audioValidation?.status === 'attached' || audioValidation?.status === 'stale')
      ? true // Explicit validation says audio is present
      : verifiedHasAudio !== undefined
        ? verifiedHasAudio // Trust explicit verification result
        : hasTimestamps
          ? true // Timestamps only set after actual TTS generation
          : isWithinSynthCount; // Fallback: segment has voice_id and falls within synthesized count

    const audioState = audioValidation?.status === 'invalid' ? 'invalid' : audioValidation?.status === 'stale' ? 'stale' : playbackFailures > 0 ? 'playback_failed' : hasAudio ? 'attached' : 'missing';
    const synthState = synthJob?.status === 'completed' ? 'completed' : synthJob?.status === 'failed' || synthJob?.status === 'exhausted' ? 'failed' : synthJob?.status === 'running' ? 'running' : synthJob?.status === 'retrying' ? 'retrying' : stale ? 'stale' : hasAudio ? 'completed' : book.status === 'synthesizing' ? 'queued' : 'not_created';
    return {
      index: idx + 1,
      segment,
      synthState,
      audioState,
      readState: debug?.userProgress?.last_read_segment_id === segment.id ? 'current' : 'not_opened',
      listenState: playbackFailures > 0 ? 'failed' : completedListens > 0 ? 'completed' : 'not_attempted',
      audioDurationSec: segment.timestamps?.items?.length
        ? Math.round(segment.timestamps.items[segment.timestamps.items.length - 1].end)
        : undefined,
      playbackFailures,
      retryCount: synthJob?.retry_count || 0,
      blocker: playbackFailures > 0 ? 'Playback failed for this segment' : audioState === 'invalid' ? audioValidation?.error || 'Audio artifact invalid' : !hasAudio ? 'Audio missing or not attached' : undefined,
      synthJob,
      audioValidation,
    };
  });

  const total = Math.max(book.total_segments || segments.length, segments.length, 1);
  const textReadyCount = segments.length || book.total_segments || 0;
  const audioReadyCount = book.synthesized_segments ?? segments.filter((s) => s.audioState === 'attached').length;
  const staleAudioCount = segments.filter((s) => s.audioState === 'stale').length;
  const failedAudioCount = segments.filter((s) => s.audioState === 'playback_failed' || s.synthState === 'failed').length;
  const readinessScore = Math.round(((textReadyCount / total) * 0.4 + (audioReadyCount / total) * 0.6) * 100);
  const userReadSegment = debug?.userProgress?.last_read_segment_id ? segments.find((s) => s.segment.id === debug.userProgress?.last_read_segment_id)?.index || 0 : Math.max(0, ...segments.filter((s) => s.readState === 'read' || s.readState === 'current').map((s) => s.index));
  const userListenedSegment = debug?.userProgress?.last_listened_segment_id ? segments.find((s) => s.segment.id === debug.userProgress?.last_listened_segment_id)?.index || 0 : Math.max(0, ...segments.filter((s) => s.listenState === 'completed' || s.listenState === 'partial').map((s) => s.index));

  let blocker: string | undefined;
  if (book.error || status?.error) blocker = book.error || status?.error;
  else if (failedAudioCount > 0) blocker = `${failedAudioCount} segment(s) have failed synthesis or playback`;
  else if (audioReadyCount < total && ['synthesizing', 'synthesis_error'].includes(book.status)) blocker = `Audio is available through ${audioReadyCount}/${total} segments`;
  else if (book.waiting_for_mapping || (personas?.unmapped?.length || 0) > 0) blocker = 'Waiting for persona voice mapping';

  const perspective = debug?.userProgress
    ? `User journey is ${debug.userProgress.journey_state}. Last listened: ${debug.userProgress.last_listened_segment_id || 'none'}, playback failures: ${debug.userProgress.playback_failures}.`
    : blocker
      ? `User can read text, but listening is blocked or incomplete: ${blocker}.`
      : 'User can read and listen normally. No journey blocker detected.';

  return {
    book,
    status,
    pipeline,
    personas,
    userProgress: debug?.userProgress,
    playbackEvents: debug?.playbackEvents,
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
