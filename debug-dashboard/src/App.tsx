import { useEffect, useMemo, useState } from 'react';
import {
  IconActivity,
  IconAlertTriangle,
  IconBook2,
  IconDatabase,
  IconHeadphones,
  IconHeartbeat,
  IconPlayerPlay,
  IconRefresh,
  IconSearch,
  IconServer,
  IconUserCheck,
  IconWaveSine,
} from '@tabler/icons-react';
import { fetchBookStream, getAudioValidation, getBookStatus, getBooks, getDebugEvents, getHealth, getPersonas, getPipelineStatus, getPlaybackEvents, getProviders, getSegments, getSynthJobs, getUserProgress } from './api';
import { buildEvents, deriveJourney, makeDemoBooks } from './state';
import type { BookJourney, HealthResponse, LiveEvent, ProvidersResponse, SegmentInspection } from './types';

const activeStatuses = new Set(['uploaded', 'parsing', 'segmenting', 'voice_mapping', 'synthesizing']);

function cls(...parts: Array<string | false | undefined>) {
  return parts.filter(Boolean).join(' ');
}

function statusColor(status?: string) {
  if (!status) return 'secondary';
  if (['healthy', 'ready', 'synthesized', 'completed', 'success'].includes(status)) return 'success';
  if (['synthesizing', 'segmenting', 'parsing', 'running', 'in_progress'].includes(status)) return 'primary';
  if (['voice_mapping', 'queued', 'stale', 'warning'].includes(status)) return 'warning';
  if (['error', 'synthesis_error', 'failed', 'danger', 'unhealthy'].includes(status)) return 'danger';
  return 'secondary';
}

function fmtTime(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function pct(n: number) {
  return `${Math.max(0, Math.min(100, Math.round(n)))}%`;
}

function useLiveDashboard() {
  const [journeys, setJourneys] = useState<BookJourney[]>(makeDemoBooks());
  const [events, setEvents] = useState<LiveEvent[]>(buildEvents(makeDemoBooks()));
  const [health, setHealth] = useState<HealthResponse | undefined>();
  const [providers, setProviders] = useState<ProvidersResponse | undefined>();
  const [mode, setMode] = useState<'live' | 'demo'>('demo');
  const [lastUpdated, setLastUpdated] = useState<string>(new Date().toISOString());
  const [error, setError] = useState<string | undefined>();
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;

    async function load(nextTick: number) {
      try {
        const [healthResult, providersResult, books] = await Promise.all([
          getHealth().catch((err) => {
            throw new Error(`health: ${err.message}`);
          }),
          getProviders().catch(() => undefined),
          getBooks(),
        ]);

        const selectedBooks = [...books]
          .sort((a, b) => Date.parse(b.uploaded_at) - Date.parse(a.uploaded_at))
          .slice(0, 8);

        const loadedJourneys = await Promise.all(
          selectedBooks.map(async (book) => {
            const [status, segments, streamSegments, pipeline, personas, synthJobs, audioValidations, playbackEvents, userProgress] = await Promise.all([
              getBookStatus(book.id).catch(() => undefined),
              getSegments(book.id).catch(() => []),
              fetchBookStream(book.id).catch(() => []),
              getPipelineStatus(book.id).catch(() => undefined),
              getPersonas(book.id).catch(() => undefined),
              getSynthJobs(book.id).catch(() => []),
              getAudioValidation(book.id).catch(() => []),
              getPlaybackEvents(book.id).catch(() => []),
              getUserProgress(book.id).catch(() => undefined),
            ]);
            const mergedSegments = streamSegments.length ? streamSegments : segments;
            return deriveJourney(book, status, pipeline, personas, mergedSegments, { synthJobs, audioValidations, playbackEvents, userProgress });
          }),
        );

        if (cancelled) return;
        const finalJourneys = loadedJourneys.length ? loadedJourneys : makeDemoBooks(nextTick);
        setMode(loadedJourneys.length ? 'live' : 'demo');
        setHealth(healthResult);
        setProviders(providersResult);
        setJourneys(finalJourneys);
        const apiEvents = await getDebugEvents().catch(() => []);
        setEvents(apiEvents.length ? apiEvents : buildEvents(finalJourneys, healthResult, nextTick));
        setLastUpdated(new Date().toISOString());
        setError(undefined);
      } catch (err) {
        if (cancelled) return;
        const demo = makeDemoBooks(nextTick);
        setMode('demo');
        setJourneys(demo);
        setEvents(buildEvents(demo, undefined, nextTick));
        setLastUpdated(new Date().toISOString());
        setError(err instanceof Error ? err.message : 'Unknown API error');
      }
    }

    load(tick);
    const id = window.setInterval(() => {
      setTick((prev) => {
        const next = prev + 1;
        load(next);
        return next;
      });
    }, 2500);

    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, []);

  return { journeys, events, health, providers, mode, lastUpdated, error, tick };
}

export function App() {
  const { journeys, events, health, providers, mode, lastUpdated, error } = useLiveDashboard();
  const [selectedBookId, setSelectedBookId] = useState<string>('');
  const [query, setQuery] = useState('');
  const [selectedSegment, setSelectedSegment] = useState<SegmentInspection | undefined>();

  const selectedJourney = useMemo(() => {
    return journeys.find((j) => j.book.id === selectedBookId) || journeys[0];
  }, [journeys, selectedBookId]);

  const filteredSegments = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!selectedJourney) return [];
    if (!needle) return selectedJourney.segments;
    return selectedJourney.segments.filter((row) => {
      return [
        row.segment.id,
        row.segment.chapter,
        row.segment.person,
        row.segment.voice_id,
        row.synthState,
        row.audioState,
        row.blocker,
        row.segment.text,
      ]
        .filter(Boolean)
        .some((v) => String(v).toLowerCase().includes(needle));
    });
  }, [query, selectedJourney]);

  const overview = useMemo(() => {
    const totalBooks = journeys.length;
    const processing = journeys.filter((j) => activeStatuses.has(j.book.status)).length;
    const stuck = journeys.filter((j) => Boolean(j.blocker)).length;
    const missingAudio = journeys.reduce((sum, j) => sum + Math.max(0, (j.book.total_segments || j.segments.length) - j.audioReadyCount), 0);
    const failedSynths = journeys.reduce((sum, j) => sum + j.failedAudioCount, 0);
    const activeUsers = journeys.filter((j) => j.userReadSegment > 0 || j.userListenedSegment > 0).length;
    return { totalBooks, processing, stuck, missingAudio, failedSynths, activeUsers };
  }, [journeys]);

  return (
    <div className="page debug-shell">
      <aside className="debug-sidebar navbar navbar-vertical navbar-expand-lg">
        <div className="container-fluid flex-column align-items-stretch">
          <div className="navbar-brand navbar-brand-autodark justify-content-start gap-2">
            <span className="brand-glyph"><IconWaveSine size={18} /></span>
            <div>
              <div className="fw-bold">TwelveReader</div>
              <div className="text-secondary small">Debug Dashboard</div>
            </div>
          </div>
          <div className="nav-section-label">System</div>
          <div className="navbar-nav">
            {[
              ['Overview', IconActivity],
              ['Books', IconBook2],
              ['Segments', IconDatabase],
              ['Synth Jobs', IconHeadphones],
            ].map(([label, Icon]: any, index) => (
              <a key={label} className={cls('nav-link', index === 0 && 'active')} href={`#${String(label).toLowerCase().replace(' ', '-')}`}>
                <span className="nav-link-icon d-md-none d-lg-inline-block"><Icon size={17} /></span>
                <span className="nav-link-title">{label}</span>
              </a>
            ))}
          </div>
          <div className="nav-section-label">Inspection</div>
          <div className="navbar-nav">
            {[
              ['Audio Artifacts', IconPlayerPlay],
              ['User Journey', IconUserCheck],
              ['Events', IconHeartbeat],
            ].map(([label, Icon]: any) => (
              <a key={label} className="nav-link" href={`#${String(label).toLowerCase().replace(' ', '-')}`}>
                <span className="nav-link-icon d-md-none d-lg-inline-block"><Icon size={17} /></span>
                <span className="nav-link-title">{label}</span>
              </a>
            ))}
          </div>
          <div className="mt-auto p-3 sidebar-status">
            <span className={cls('status status-dot status-dot-animated', mode === 'live' ? 'status-green' : 'status-yellow')} />
            <div>
              <div className="small fw-semibold">{mode === 'live' ? 'Live API connected' : 'Demo fallback active'}</div>
              <div className="text-secondary small">Updated {fmtTime(lastUpdated)}</div>
            </div>
          </div>
        </div>
      </aside>

      <div className="page-wrapper debug-main">
        <header className="navbar navbar-expand-md debug-topbar">
          <div className="container-xl">
            <div className="navbar-nav flex-row">
              <a className="nav-link active" href="#overview"><IconActivity size={16} /> Overview</a>
              <a className="nav-link" href="#segments"><IconDatabase size={16} /> Segments</a>
              <a className="nav-link" href="#user-journey"><IconUserCheck size={16} /> Journey</a>
              <a className="nav-link" href="#events"><IconHeartbeat size={16} /> Events</a>
            </div>
            <div className="ms-auto d-flex align-items-center gap-2 flex-wrap">
              <span className="text-secondary small d-none d-xl-inline">Live State Inspector</span>
              <span className={cls('badge', `bg-${statusColor(health?.status)}-lt`)}>
                <IconServer size={14} /> API {health?.status || 'unknown'}
              </span>
              <span className="badge bg-blue-lt">TTS {providers?.tts?.length ?? 0}</span>
              <span className="badge bg-secondary-lt">LLM {providers?.llm?.length ?? 0}</span>
              <button className="btn btn-primary btn-sm" onClick={() => window.location.reload()}>
                <IconRefresh size={15} /> Refresh
              </button>
            </div>
          </div>
        </header>

        <main className="page-body">
          <div className="container-xl">
            <div className="page-header d-print-none mb-3">
              <div className="row align-items-center g-2">
                <div className="col">
                  <h1 className="page-title">End-to-end book journey dashboard</h1>
                  <div className="text-secondary">
                    Upload → extraction → segmentation → voice mapping → synth/audio → user reading/listening. Updates every 2.5s without page refresh.
                  </div>
                </div>
                <div className="col-auto">
                  <select
                    className="form-select"
                    value={selectedJourney?.book.id || ''}
                    onChange={(e) => setSelectedBookId(e.target.value)}
                  >
                    {journeys.map((journey) => (
                      <option key={journey.book.id} value={journey.book.id}>{journey.book.title || journey.book.id}</option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            {error && (
              <div className="alert alert-warning" role="alert">
                <IconAlertTriangle size={18} /> Live API unavailable or incomplete: {error}. Showing reactive demo telemetry with the same dashboard contract.
              </div>
            )}

            <section id="overview" className="row row-cards mb-3">
              <Metric title="Books" value={overview.totalBooks} icon={<IconBook2 size={22} />} tone="blue" detail="tracked in current window" />
              <Metric title="Processing" value={overview.processing} icon={<IconActivity size={22} />} tone="azure" detail="active lifecycle states" />
              <Metric title="Stuck" value={overview.stuck} icon={<IconAlertTriangle size={22} />} tone="orange" detail="has user-facing blocker" />
              <Metric title="Missing audio" value={overview.missingAudio} icon={<IconHeadphones size={22} />} tone="yellow" detail="segments not listenable" />
              <Metric title="Failed synth/playback" value={overview.failedSynths} icon={<IconWaveSine size={22} />} tone="red" detail="requires inspection" />
              <Metric title="Active users" value={overview.activeUsers} icon={<IconUserCheck size={22} />} tone="green" detail="read/listen progress exists" />
            </section>

            {selectedJourney && (
              <>
                <BookJourneyHeader journey={selectedJourney} />
                <div className="row row-cards mt-3">
                  <div className="col-lg-8">
                    <PipelineTimeline journey={selectedJourney} />
                  </div>
                  <div className="col-lg-4">
                    <UserPerspective journey={selectedJourney} />
                  </div>
                </div>

                <div className="row row-cards mt-3">
                  <div className="col-xl-8">
                    <SegmentsTable rows={filteredSegments} query={query} setQuery={setQuery} onSelect={setSelectedSegment} />
                  </div>
                  <div className="col-xl-4">
                    <EventFeed events={events} />
                  </div>
                </div>
              </>
            )}
          </div>
        </main>
      </div>

      {selectedSegment && <SegmentDrawer segment={selectedSegment} onClose={() => setSelectedSegment(undefined)} />}
    </div>
  );
}

function Metric({ title, value, icon, tone, detail }: { title: string; value: number | string; icon: React.ReactNode; tone: string; detail: string }) {
  return (
    <div className="col-sm-6 col-lg-4 col-xxl-2">
      <div className="card metric-card">
        <div className="card-body">
          <div className={`metric-icon text-${tone}`}>{icon}</div>
          <div className="metric-content min-w-0">
            <div className="h2 mb-0">{value}</div>
            <div className="fw-medium text-truncate">{title}</div>
            <div className="text-secondary xsmall text-truncate">{detail}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

function BookJourneyHeader({ journey }: { journey: BookJourney }) {
  const total = journey.book.total_segments || journey.segments.length || 1;
  return (
    <section className="card journey-card">
      <div className="card-body">
        <div className="row g-3 align-items-center">
          <div className="col-lg">
            <div className="d-flex align-items-center gap-2 mb-2 flex-wrap">
              <span className={cls('badge', `bg-${statusColor(journey.book.status)}-lt`)}>{journey.book.status}</span>
              <span className="badge bg-secondary-lt">{journey.book.orig_format}</span>
              <span className="badge bg-blue-lt">{journey.book.language}</span>
              {journey.blocker && <span className="badge bg-warning-lt">blocked</span>}
            </div>
            <h2 className="mb-1">{journey.book.title || journey.book.id}</h2>
            <div className="text-secondary">
              {journey.book.author || 'Unknown author'} · {journey.book.id} · uploaded {fmtTime(journey.book.uploaded_at)}
            </div>
          </div>
          <div className="col-lg-5">
            <div className="row g-2">
              <MiniStat label="Readiness" value={pct(journey.readinessScore)} />
              <MiniStat label="Text" value={`${journey.textReadyCount}/${total}`} />
              <MiniStat label="Audio" value={`${journey.audioReadyCount}/${total}`} />
              <MiniStat label="User listened" value={`${journey.userListenedSegment}/${total}`} />
            </div>
            <div className="progress progress-sm mt-3">
              <div className="progress-bar bg-primary" style={{ width: pct(journey.readinessScore) }} />
            </div>
          </div>
        </div>
        {journey.blocker && <div className="alert alert-warning mt-3 mb-0"><IconAlertTriangle size={18} /> {journey.blocker}</div>}
      </div>
    </section>
  );
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return <div className="col-6"><div className="mini-stat"><div className="fw-bold">{value}</div><div className="text-secondary small">{label}</div></div></div>;
}

function PipelineTimeline({ journey }: { journey: BookJourney }) {
  const stages = journey.pipeline?.stages?.length
    ? journey.pipeline.stages
    : [
        { stage: 'uploaded', status: 'completed', current: 1, total: 1 },
        { stage: 'parsing', status: journey.book.status === 'parsing' ? 'in_progress' : 'completed', current: 1, total: 1 },
        { stage: 'segmenting', status: journey.book.status === 'segmenting' ? 'in_progress' : 'completed', current: journey.book.segmented_paragraphs || 0, total: journey.book.total_paragraphs || 0 },
        { stage: 'synthesizing', status: journey.book.status === 'synthesizing' ? 'in_progress' : journey.audioReadyCount ? 'completed' : 'pending', current: journey.audioReadyCount, total: journey.book.total_segments || journey.segments.length },
      ];
  return (
    <div className="card" id="user-journey">
      <div className="card-header"><h3 className="card-title">Pipeline timeline</h3></div>
      <div className="card-body">
        <div className="timeline timeline-simple">
          {stages.map((stage) => {
            const total = stage.total || 1;
            const progress = Math.round((stage.current / total) * 100);
            return (
              <div className="timeline-event" key={stage.stage}>
                <div className={cls('timeline-event-icon', `bg-${statusColor(stage.status)}-lt`)}>
                  <span className={cls('status', `status-${statusColor(stage.status)}`)} />
                </div>
                <div className="card timeline-card">
                  <div className="card-body py-2">
                    <div className="d-flex justify-content-between gap-3">
                      <div>
                        <div className="fw-semibold text-capitalize">{stage.stage.replace('_', ' ')}</div>
                        <div className="text-secondary small">{stage.status} · {stage.current}/{stage.total || '—'}</div>
                      </div>
                      <span className={cls('badge align-self-start', `bg-${statusColor(stage.status)}-lt`)}>{pct(progress)}</span>
                    </div>
                    <div className="progress progress-sm mt-2"><div className={`progress-bar bg-${statusColor(stage.status)}`} style={{ width: pct(progress) }} /></div>
                    {stage.message && <div className="text-secondary small mt-1">{stage.message}</div>}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function UserPerspective({ journey }: { journey: BookJourney }) {
  const total = journey.book.total_segments || journey.segments.length || 1;
  const nextSegment = journey.segments[journey.userListenedSegment] || journey.segments[journey.userReadSegment] || journey.segments[0];
  return (
    <div className="card h-100">
      <div className="card-header"><h3 className="card-title">User perspective</h3></div>
      <div className="card-body">
        <p className="text-secondary mb-3">{journey.perspective}</p>
        <div className="datagrid">
          <div className="datagrid-item"><div className="datagrid-title">Can read</div><div className="datagrid-content"><span className="badge bg-success-lt">yes</span></div></div>
          <div className="datagrid-item"><div className="datagrid-title">Can listen all</div><div className="datagrid-content"><span className={cls('badge', journey.audioReadyCount >= total ? 'bg-success-lt' : 'bg-warning-lt')}>{journey.audioReadyCount >= total ? 'yes' : 'partial'}</span></div></div>
          <div className="datagrid-item"><div className="datagrid-title">Last read</div><div className="datagrid-content">Segment {journey.userReadSegment || '—'}</div></div>
          <div className="datagrid-item"><div className="datagrid-title">Last listened</div><div className="datagrid-content">Segment {journey.userListenedSegment || '—'}</div></div>
          <div className="datagrid-item"><div className="datagrid-title">Next segment</div><div className="datagrid-content">{nextSegment ? `${nextSegment.index} · ${nextSegment.audioState}` : '—'}</div></div>
          <div className="datagrid-item"><div className="datagrid-title">Playback failures</div><div className="datagrid-content">{journey.userProgress?.playback_failures ?? journey.failedAudioCount}</div></div>
          <div className="datagrid-item"><div className="datagrid-title">Journey state</div><div className="datagrid-content">{journey.userProgress?.journey_state || 'derived'}</div></div>
        </div>
      </div>
    </div>
  );
}

function SegmentsTable({ rows, query, setQuery, onSelect }: { rows: SegmentInspection[]; query: string; setQuery: (v: string) => void; onSelect: (s: SegmentInspection) => void }) {
  return (
    <div className="card" id="segments">
      <div className="card-header flex-wrap gap-2">
        <h3 className="card-title">Segments</h3>
        <div className="ms-auto input-icon segment-search">
          <span className="input-icon-addon"><IconSearch size={16} /></span>
          <input className="form-control form-control-sm" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search segment, voice, status, error" />
        </div>
      </div>
      <div className="table-responsive">
        <table className="table table-vcenter card-table table-hover">
          <thead>
            <tr>
              <th>#</th><th>Segment</th><th>Persona</th><th>Synth</th><th>Audio</th><th>Artifact</th><th>User</th><th>Playback</th><th>Blocker</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.segment.id} onClick={() => onSelect(row)} className="segment-row">
                <td className="text-secondary">{row.index}</td>
                <td><div className="fw-semibold mono-id">{row.segment.id}</div><div className="text-secondary small text-truncate segment-text">{row.segment.text}</div></td>
                <td><span className="badge bg-secondary-lt">{row.segment.person || 'narrator'}</span></td>
                <td><span className={cls('badge', `bg-${statusColor(row.synthState)}-lt`)}>{row.synthState}</span></td>
                <td><span className={cls('badge', `bg-${statusColor(row.audioState)}-lt`)}>{row.audioState}</span></td>
                <td className="text-secondary small">{row.audioValidation?.bytes ? `${row.audioValidation.bytes} B` : row.audioValidation?.status || '—'}</td>
                <td><div className="small">read: {row.readState}</div><div className="small text-secondary">listen: {row.listenState}</div></td>
                <td>{row.playbackFailures ? <span className="badge bg-danger-lt">{row.playbackFailures} failed</span> : <span className="text-secondary">{row.audioDurationSec ? `${row.audioDurationSec}s` : '—'}</span>}</td>
                <td className="blocker-cell">{row.blocker ? <span className="text-warning">{row.blocker}</span> : <span className="text-secondary">—</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function EventFeed({ events }: { events: LiveEvent[] }) {
  return (
    <div className="card" id="events">
      <div className="card-header"><h3 className="card-title">Live event feed</h3></div>
      <div className="list-group list-group-flush event-feed">
        {events.map((event) => (
          <div className="list-group-item" key={event.id}>
            <div className="row align-items-center">
              <div className="col-auto"><span className={cls('status-dot', `status-${statusColor(event.severity)}`)} /></div>
              <div className="col text-truncate">
                <div className="fw-semibold text-truncate">{event.title}</div>
                <div className="text-secondary small text-truncate">{event.detail}</div>
              </div>
              <div className="col-auto text-secondary small">{fmtTime(event.at)}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function SegmentDrawer({ segment, onClose }: { segment: SegmentInspection; onClose: () => void }) {
  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <aside className="segment-drawer" onClick={(e) => e.stopPropagation()}>
        <div className="drawer-header">
          <div>
            <div className="text-secondary small">Segment {segment.index}</div>
            <h3 className="mb-0 mono-id">{segment.segment.id}</h3>
          </div>
          <button className="btn btn-icon btn-ghost-secondary" onClick={onClose}>×</button>
        </div>
        <div className="drawer-body">
          <div className="d-flex gap-2 flex-wrap mb-3">
            <span className={cls('badge', `bg-${statusColor(segment.synthState)}-lt`)}>{segment.synthState}</span>
            <span className={cls('badge', `bg-${statusColor(segment.audioState)}-lt`)}>{segment.audioState}</span>
            <span className="badge bg-blue-lt">{segment.segment.voice_id || 'no voice'}</span>
          </div>
          <div className="datagrid mb-3">
            <div className="datagrid-item"><div className="datagrid-title">Book</div><div className="datagrid-content mono-id">{segment.segment.book_id}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Chapter</div><div className="datagrid-content">{segment.segment.chapter}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Persona</div><div className="datagrid-content">{segment.segment.person}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Voice</div><div className="datagrid-content">{segment.segment.voice_id || '—'}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Audio duration</div><div className="datagrid-content">{segment.audioDurationSec ? `${segment.audioDurationSec}s` : '—'}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Retries</div><div className="datagrid-content">{segment.retryCount}</div></div>
          </div>
          {segment.blocker && <div className="alert alert-warning"><IconAlertTriangle size={18} /> {segment.blocker}</div>}
          <h4>Text preview</h4>
          <div className="text-preview">{segment.segment.text}</div>
          <h4 className="mt-4">Synth job</h4>
          <div className="datagrid mb-3">
            <div className="datagrid-item"><div className="datagrid-title">Job status</div><div className="datagrid-content">{segment.synthJob?.status || 'derived'}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Provider</div><div className="datagrid-content">{segment.synthJob?.provider || '—'}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Output</div><div className="datagrid-content mono-id">{segment.synthJob?.output_path || segment.audioValidation?.path || '—'}</div></div>
            <div className="datagrid-item"><div className="datagrid-title">Bytes</div><div className="datagrid-content">{segment.synthJob?.output_bytes || segment.audioValidation?.bytes || '—'}</div></div>
          </div>
          <h4 className="mt-4">Raw inspection</h4>
          <pre className="raw-json">{JSON.stringify(segment, null, 2)}</pre>
        </div>
      </aside>
    </div>
  );
}
