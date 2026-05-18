import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import {
  IconAlertTriangle,
  IconActivity,
  IconBook2,
  IconHeadphones,
  IconShieldCheck,
  IconUserCheck,
  IconWaveSine,
} from '@tabler/icons-react';
import type { BookJourney, HealthResponse, LiveEvent, ProvidersResponse, SmokeVisibilityResponse } from '../types';

const activeStatuses = new Set(['uploaded', 'parsing', 'segmenting', 'voice_mapping', 'synthesizing']);

function statusColor(status?: string) {
  if (!status) return 'secondary';
  if (['healthy', 'ready', 'synthesized', 'completed', 'success'].includes(status)) return 'success';
  if (['synthesizing', 'segmenting', 'parsing', 'running', 'in_progress'].includes(status)) return 'primary';
  if (['voice_mapping', 'queued', 'stale', 'warning'].includes(status)) return 'warning';
  if (['error', 'synthesis_error', 'failed', 'danger', 'unhealthy'].includes(status)) return 'danger';
  return 'secondary';
}

export function OverviewPage({
  journeys,
  events,
  health,
  providers,
  readiness,
  error,
}: {
  journeys: BookJourney[];
  events: LiveEvent[];
  health?: HealthResponse;
  providers?: ProvidersResponse;
  readiness?: SmokeVisibilityResponse;
  error?: string;
}) {
  const overview = useMemo(() => {
    const totalBooks = journeys.length;
    const processing = journeys.filter((j) => activeStatuses.has(j.book.status)).length;
    const stuck = journeys.filter((j) => Boolean(j.blocker)).length;
    const missingAudio = journeys.reduce(
      (sum, j) => sum + Math.max(0, (j.book.total_segments || j.segments.length) - j.audioReadyCount),
      0,
    );
    const failedSynths = journeys.reduce((sum, j) => sum + j.failedAudioCount, 0);
    const activeUsers = journeys.filter((j) => j.userReadSegment > 0 || j.userListenedSegment > 0).length;
    return { totalBooks, processing, stuck, missingAudio, failedSynths, activeUsers };
  }, [journeys]);

  const recentEvents = useMemo(() => events.slice(0, 12), [events]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">End-to-end Book Journey Dashboard</h1>
        <div className="text-secondary">
          Upload → extraction → segmentation → voice mapping → synth/audio → user reading/listening
        </div>
      </div>

      {error && (
        <div className="alert alert-warning" role="alert">
          <IconAlertTriangle size={18} /> Live API unavailable: {error}. Showing reactive demo telemetry.
        </div>
      )}

      {/* Metric Cards */}
      <section className="row row-cards mb-3">
        <MetricCard title="Books" value={overview.totalBooks} icon={<IconBook2 size={22} />} tone="blue" detail="tracked" />
        <MetricCard title="Processing" value={overview.processing} icon={<IconActivity size={22} />} tone="azure" detail="active pipelines" />
        <MetricCard title="Stuck" value={overview.stuck} icon={<IconAlertTriangle size={22} />} tone="orange" detail="has user-facing blocker" />
        <MetricCard title="Missing Audio" value={overview.missingAudio} icon={<IconHeadphones size={22} />} tone="yellow" detail="segments not listenable" />
        <MetricCard title="Failed Synth" value={overview.failedSynths} icon={<IconWaveSine size={22} />} tone="red" detail="requires inspection" />
        <MetricCard title="Active Users" value={overview.activeUsers} icon={<IconUserCheck size={22} />} tone="green" detail="have progress" />
      </section>

      {/* Readiness Smoke Visibility */}
      {readiness && (
        <div className="card mb-3">
          <div className="card-header d-flex justify-content-between align-items-center">
            <h3 className="card-title">Readiness Smoke Checks</h3>
            <span className={`badge bg-${readyColor(readiness.overall)}-lt`}>
              <IconShieldCheck size={14} /> Overall: {readiness.overall.replace('_', ' ')}
            </span>
          </div>
          <div className="table-responsive">
            <table className="table table-vcenter card-table table-hover mb-0">
              <thead>
                <tr>
                  <th>Endpoint</th>
                  <th>Status</th>
                  <th>HTTP Code</th>
                  <th>Latency</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {readiness.checks.map((check) => (
                  <tr key={check.name}>
                    <td><code className="text-small">{check.path}</code></td>
                    <td><span className={`badge bg-${checkStatusColor(check.status)}-lt`}>{check.status}</span></td>
                    <td className="text-secondary">{check.http_code}</td>
                    <td className="text-secondary">{check.latency_ms.toFixed(2)} ms</td>
                    <td>{check.error ? <span className="text-danger small">{check.error}</span> : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="card-footer text-secondary xsmall">Last checked: {readiness.timestamp}</div>
        </div>
      )}

      {/* Quick Book Status */}
      {journeys.length > 0 ? (
        <div className="card mb-3">
          <div className="card-header d-flex justify-content-between align-items-center">
            <h3 className="card-title">Book Quick Status</h3>
            <Link to="/books" className="btn btn-sm btn-secondary">View All →</Link>
          </div>
          <div className="table-responsive">
            <table className="table table-vcenter card-table table-hover">
              <thead>
                <tr>
                  <th>Title</th>
                  <th>Status</th>
                  <th>Readiness</th>
                  <th>Text</th>
                  <th>Audio</th>
                  <th>User Read</th>
                  <th>User Listen</th>
                  <th>Blocker</th>
                </tr>
              </thead>
              <tbody>
                {journeys.map((j) => (
                  <tr key={j.book.id}>
                    <td>
                      <Link to={`/books/${j.book.id}`} className="fw-semibold text-white text-decoration-none">
                        {j.book.title || j.book.id}
                      </Link>
                      <div className="text-secondary small">{j.book.author}</div>
                    </td>
                    <td><span className={`badge bg-${statusColor(j.book.status)}-lt`}>{j.book.status}</span></td>
                    <td>
                      <div className="progress progress-sm" style={{ width: '80px' }}>
                        <div className={`progress-bar bg-${statusColor(j.readinessScore > 80 ? 'success' : j.readinessScore > 40 ? 'primary' : 'warning')}`} style={{ width: `${j.readinessScore}%` }} />
                      </div>
                    </td>
                    <td className="text-secondary">{j.textReadyCount}</td>
                    <td className="text-secondary">{j.audioReadyCount}/{j.book.total_segments || j.segments.length}</td>
                    <td className="text-secondary">{j.userReadSegment || '—'}</td>
                    <td className="text-secondary">{j.userListenedSegment || '—'}</td>
                    <td>{j.blocker ? <span className="text-warning small">{j.blocker}</span> : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="card mb-3"><div className="card-body text-center py-5">
          <div className="text-secondary mb-2">No books found</div>
          <div className="text-secondary small">Upload a book to TwelveReader to see it here. The dashboard connects directly to the backend API.</div>
        </div></div>
      )}

      {/* Recent Events */}
      {recentEvents.length > 0 && (
        <div className="row row-cards">
          <div className="col-12">
            <div className="card">
              <div className="card-header d-flex justify-content-between align-items-center">
                <h3 className="card-title">Recent Events</h3>
                <Link to="/user-activity" className="btn btn-sm btn-secondary">View All →</Link>
              </div>
              <div className="list-group list-group-flush event-feed">
                {recentEvents.map((event) => (
                  <div className="list-group-item" key={event.id}>
                    <div className="row align-items-center">
                      <div className="col-auto">
                        <span className={`status-dot status-${statusColor(event.severity)}`} />
                      </div>
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
          </div>
        </div>
      )}
    </>
  );
}

function MetricCard({ title, value, icon, tone, detail }: { title: string; value: number | string; icon: React.ReactNode; tone: string; detail: string }) {
  return (
    <div className="col-sm-6 col-lg-4 col-xl-4">
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

function fmtTime(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function readyColor(overall: string) {
  switch (overall) {
    case 'all_ok': return 'success';
    case 'degraded': return 'warning';
    case 'unhealthy': return 'danger';
    default: return 'secondary';
  }
}

function checkStatusColor(status: string) {
  switch (status) {
    case 'ok': return 'success';
    case 'warning': return 'warning';
    case 'error': return 'danger';
    default: return 'secondary';
  }
}
