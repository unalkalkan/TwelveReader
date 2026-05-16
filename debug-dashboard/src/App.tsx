import { useEffect, useMemo, useState } from 'react';
import { BrowserRouter, Routes, Route, useParams } from 'react-router-dom';
import { Layout } from './components/Layout';
import { OverviewPage } from './pages/OverviewPage';
import { BooksListPage } from './pages/BooksListPage';
import { BookDetailPage } from './pages/BookDetailPage';
import { SynthJobsPage } from './pages/SynthJobsPage';
import { UserActivityPage } from './pages/UserActivityPage';
import { AudioArtifactsPage } from './pages/AudioArtifactsPage';
import { getAudioValidation, getBookStatus, getBooks, getDebugEvents, getHealth, getPersonas, getPipelineStatus, getPlaybackEvents, getProviders, getSegments, getSynthJobs, getUserProgress } from './api';
import { deriveJourney } from './state';
import type { BookJourney, HealthResponse, LiveEvent, ProvidersResponse } from './types';

function useLiveDashboard() {
  const [journeys, setJourneys] = useState<BookJourney[]>([]);
  const [events, setEvents] = useState<LiveEvent[]>([]);
  const [health, setHealth] = useState<HealthResponse | undefined>();
  const [providers, setProviders] = useState<ProvidersResponse | undefined>();
  const [apiConnected, setApiConnected] = useState(false);
  const [sseConnected, setSseConnected] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<string>(new Date().toISOString());
  const [error, setError] = useState<string | undefined>();

  // SSE connection for live event push
  useEffect(() => {
    let es: EventSource | null = null;
    try {
      es = new EventSource(`${window.location.origin}/api/v1/debug/stream`);
      es.onopen = () => setSseConnected(true);
      es.onerror = () => setSseConnected(false);
      es.addEventListener('debug-state', (e) => {
        try {
          const data = JSON.parse(e.data) as LiveEvent[];
          setEvents(data);
          setLastUpdated(new Date().toISOString());
        } catch { /* ignore parse errors */ }
      });
    } catch { /* SSE unavailable — falls back to polling */ }

    return () => { es?.close(); };
  }, []);

  // REST polling for journeys (complex multi-endpoint aggregation)
  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [healthResult, providersResult, books] = await Promise.all([
          getHealth().catch((err) => { throw new Error(`health: ${err.message}`); }),
          getProviders().catch(() => undefined),
          getBooks(),
        ]);

        setApiConnected(true);
        const selectedBooks = [...books]
          .sort((a, b) => Date.parse(b.uploaded_at) - Date.parse(a.uploaded_at))
          .slice(0, 20);

        if (selectedBooks.length === 0) {
          if (cancelled) return;
          setJourneys([]);
          setHealth(healthResult);
          setProviders(providersResult);
          const apiEvents = await getDebugEvents().catch(() => []);
          setEvents(apiEvents);
          setLastUpdated(new Date().toISOString());
          setError(undefined);
          return;
        }

        const loadedJourneys = await Promise.all(
          selectedBooks.map(async (book) => {
            const [status, pipeline, personas, synthJobs, audioValidations, playbackEvents, userProgress] = await Promise.all([
              getBookStatus(book.id).catch(() => undefined),
              getPipelineStatus(book.id).catch(() => undefined),
              getPersonas(book.id).catch(() => undefined),
              getSynthJobs(book.id).catch(() => []),
              getAudioValidation(book.id).catch(() => []),
              getPlaybackEvents(book.id).catch(() => []),
              getUserProgress(book.id).catch(() => undefined),
            ]);

            const [segments, streamSegments] = await Promise.all([
              getSegments(book.id).catch(() => []),
              fetch(`${(import.meta.env.VITE_TWELVEREADER_API_URL || window.location.origin)}/api/v1/books/${book.id}/stream`)
                .then((r) => { if (!r.ok) throw new Error('stream'); return r.text(); })
                .then((t) => t.split('\n').filter(Boolean).map((l) => JSON.parse(l)))
                .catch(() => []),
            ]);

            const mergedSegments = streamSegments.length ? streamSegments : segments;
            return deriveJourney(book, status, pipeline, personas, mergedSegments, { synthJobs, audioValidations, playbackEvents, userProgress });
          }),
        );

        if (cancelled) return;
        setJourneys(loadedJourneys);
        setHealth(healthResult);
        setProviders(providersResult);

        const apiEvents = await getDebugEvents().catch(() => []);
        setEvents(apiEvents);
        setLastUpdated(new Date().toISOString());
        setError(undefined);
      } catch (err) {
        if (cancelled) return;
        setApiConnected(false);
        setJourneys([]);
        setEvents([]);
        setLastUpdated(new Date().toISOString());
        setError(err instanceof Error ? err.message : 'Unknown API error');
      }
    }

    load();
    const id = window.setInterval(load, 5000);
    return () => { cancelled = true; window.clearInterval(id); };
  }, []);

  return { journeys, events, health, providers, apiConnected, sseConnected, lastUpdated, error };
}

function BookDetailRoute({ journeys, events }: { journeys: BookJourney[]; events: LiveEvent[] }) {
  const { bookId } = useParams<{ bookId: string }>();
  const journey = useMemo(() => journeys.find((j) => j.book.id === bookId) || journeys[0], [journeys, bookId]);
  if (!journey) return (
    <div className="card"><div className="card-body text-center py-5">
      <div className="text-secondary mb-2">No books found</div>
      <div className="text-secondary small">Upload a book to TwelveReader to inspect its journey here</div>
    </div></div>
  );
  return <BookDetailPage journey={journey} events={events} />;
}

export function App() {
  const { journeys, events, health, providers, apiConnected, sseConnected, lastUpdated, error } = useLiveDashboard();

  return (
    <BrowserRouter>
      <Layout apiConnected={apiConnected} sseConnected={sseConnected} health={health ? { status: health.status } : undefined} providers={providers ?? undefined} lastUpdated={lastUpdated}>
        <Routes>
          <Route path="/" element={<OverviewPage journeys={journeys} events={events} health={health} providers={providers} error={error} />} />
          <Route path="/books" element={<BooksListPage journeys={journeys} />} />
          <Route path="/books/:bookId" element={<BookDetailRoute journeys={journeys} events={events} />} />
          <Route path="/segments" element={<SegmentsQuickView journeys={journeys} />} />
          <Route path="/synth-jobs" element={<SynthJobsPage journeys={journeys} />} />
          <Route path="/user-activity" element={<UserActivityPage journeys={journeys} events={events} />} />
          <Route path="/audio-artifacts" element={<AudioArtifactsPage journeys={journeys} />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}

/* Segments quick view page - aggregates segments across all books */
function SegmentsQuickView({ journeys }: { journeys: BookJourney[] }) {
  const [query, setQuery] = useState('');

  function statusColor(status?: string) {
    if (!status) return 'secondary';
    if (['completed', 'success'].includes(status)) return 'success';
    if (['running', 'in_progress'].includes(status)) return 'primary';
    if (['queued', 'stale', 'retrying'].includes(status)) return 'warning';
    if (['failed', 'danger'].includes(status)) return 'danger';
    return 'secondary';
  }

  const allSegments = useMemo(() =>
    journeys.flatMap((j) => j.segments.map((s) => ({ ...s, bookId: j.book.id, bookTitle: j.book.title || j.book.id }))),
    [journeys]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return allSegments;
    return allSegments.filter((s) =>
      [s.segment.id, s.segment.person, s.synthState, s.audioState, (s as any).bookTitle].filter(Boolean).some((v) => String(v).toLowerCase().includes(needle)),
    );
  }, [query, allSegments]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">Segments</h1>
        <div className="text-secondary">{filtered.length} segment(s) across {journeys.length} book(s)</div>
      </div>

      {allSegments.length === 0 ? (
        <div className="card"><div className="card-body text-center py-5">
          <div className="text-secondary mb-2">No segments found</div>
          <div className="text-secondary small">Upload a book to TwelveReader to see its segments here</div>
        </div></div>
      ) : (
        <div className="card">
          <div className="card-header flex-wrap gap-2">
            <div className="ms-auto d-flex gap-2 align-items-center">
              <div className="input-icon segment-search" style={{ minWidth: '20rem' }}>
                <span className="input-icon-addon"><svg xmlns="http://www.w3.org/2000/svg" width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><circle cx={11} cy={11} r={8} /><path d="m21 21-4.3-4.3" /></svg></span>
                <input className="form-control form-control-sm" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search segment, persona, status..." />
              </div>
            </div>
          </div>
          <div className="table-responsive">
            <table className="table table-vcenter card-table table-hover">
              <thead><tr><th>#</th><th>Book</th><th>Segment</th><th>Persona</th><th>Synth</th><th>Audio</th><th>User</th><th>Blocker</th></tr></thead>
              <tbody>
                {filtered.slice(0, 300).map((row) => (
                  <tr key={(row as any).segment.id} style={{ cursor: 'pointer' }}>
                    <td className="text-secondary">{(row as any).index}</td>
                    <td><span className="mono-id small">{(row as any).bookTitle}</span></td>
                    <td><span className="mono-id">{(row as any).segment.id}</span></td>
                    <td><span className="badge bg-secondary-lt">{(row as any).segment.person || 'narrator'}</span></td>
                    <td><span className={`badge bg-${statusColor((row as any).synthState)}-lt`}>{(row as any).synthState}</span></td>
                    <td><span className={`badge bg-${statusColor((row as any).audioState)}-lt`}>{(row as any).audioState}</span></td>
                    <td><div className="small">read: {(row as any).readState}</div><div className="small text-secondary">listen: {(row as any).listenState}</div></td>
                    <td>{(row as any).blocker ? <span className="text-warning small">{(row as any).blocker.slice(0, 50)}</span> : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  );
}
