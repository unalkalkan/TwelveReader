import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { IconSearch, IconRefresh } from '@tabler/icons-react';
import type { BookJourney, SynthJob } from '../types';

function statusColor(status?: string) {
  if (!status) return 'secondary';
  if (['completed', 'success', 'attached'].includes(status)) return 'success';
  if (['running', 'in_progress', 'synthesizing'].includes(status)) return 'primary';
  if (['queued', 'stale', 'retrying', 'warning'].includes(status)) return 'warning';
  if (['failed', 'danger', 'error', 'exhausted'].includes(status)) return 'danger';
  return 'secondary';
}

function fmtTime(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export function SynthJobsPage({ journeys }: { journeys: BookJourney[] }) {
  const [query, setQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const allJobs = useMemo(() => {
    return journeys.flatMap((j) =>
      j.segments
        .filter((s) => s.synthJob || s.synthState !== 'not_created')
        .map((s) => ({
          id: s.synthJob?.id || `synth_${s.segment.book_id}_${s.segment.id}`,
          bookId: j.book.id,
          bookTitle: j.book.title || j.book.id,
          segmentId: s.segment.id,
          segmentIndex: s.index,
          status: s.synthJob?.status || s.synthState,
          provider: s.synthJob?.provider || s.segment.processing?.tts_provider || '—',
          voiceId: s.synthJob?.voice_id || s.segment.voice_id || '—',
          outputBytes: s.synthJob?.output_bytes ?? s.audioValidation?.bytes ?? null,
          outputPath: s.synthJob?.output_path || s.audioValidation?.path || '',
          retryCount: s.retryCount,
          error: s.synthJob?.error || s.blocker || '',
          updatedAt: s.synthJob?.updated_at || s.segment.processing?.generated_at || '',
        })),
    );
  }, [journeys]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return allJobs.filter((job) => {
      const matchSearch = !needle ||
        job.bookTitle.toLowerCase().includes(needle) ||
        job.segmentId.toLowerCase().includes(needle) ||
        job.status.toLowerCase().includes(needle) ||
        (job.error && job.error.toLowerCase().includes(needle));
      const matchStatus = filterStatus === 'all' || job.status === filterStatus;
      return matchSearch && matchStatus;
    });
  }, [allJobs, query, filterStatus]);

  const statusOptions = useMemo(() => {
    const statuses = new Set(allJobs.map((j) => j.status));
    return Array.from(statuses).sort();
  }, [allJobs]);

  const stats = useMemo(() => ({
    total: allJobs.length,
    completed: allJobs.filter((j) => j.status === 'completed').length,
    failed: allJobs.filter((j) => j.status === 'failed' || j.status === 'exhausted').length,
    running: allJobs.filter((j) => j.status === 'running' || j.status === 'retrying').length,
    queued: allJobs.filter((j) => j.status === 'queued' || j.status === 'not_created').length,
  }), [allJobs]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">Synth Jobs</h1>
        <div className="text-secondary">{filtered.length} job(s) across {journeys.length} book(s)</div>
      </div>

      {/* Stats Row */}
      <section className="row row-cards mb-3">
        <div className="col-sm-6 col-lg-3 col-xl-2">
          <div className="card metric-card">
            <div className="card-body">
              <div className="h4 mb-0">{stats.total}</div>
              <div className="text-secondary xsmall">Total</div>
            </div>
          </div>
        </div>
        <div className="col-sm-6 col-lg-3 col-xl-2">
          <div className="card metric-card">
            <div className="card-body">
              <div className="h4 mb-0 text-success">{stats.completed}</div>
              <div className="text-secondary xsmall">Completed</div>
            </div>
          </div>
        </div>
        <div className="col-sm-6 col-lg-3 col-xl-2">
          <div className="card metric-card">
            <div className="card-body">
              <div className="h4 mb-0 text-primary">{stats.running}</div>
              <div className="text-secondary xsmall">Running</div>
            </div>
          </div>
        </div>
        <div className="col-sm-6 col-lg-3 col-xl-2">
          <div className="card metric-card">
            <div className="card-body">
              <div className="h4 mb-0 text-warning">{stats.queued}</div>
              <div className="text-secondary xsmall">Queued</div>
            </div>
          </div>
        </div>
        <div className="col-sm-6 col-lg-3 col-xl-2">
          <div className="card metric-card">
            <div className="card-body">
              <div className="h4 mb-0 text-danger">{stats.failed}</div>
              <div className="text-secondary xsmall">Failed</div>
            </div>
          </div>
        </div>
      </section>

      {/* Jobs Table */}
      <div className="card">
        <div className="card-header flex-wrap gap-2">
          <h3 className="card-title">Jobs</h3>
          <div className="ms-auto d-flex gap-2 align-items-center">
            <select className="form-select form-select-sm" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} style={{ width: 'auto' }}>
              <option value="all">All statuses</option>
              {statusOptions.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
            <div className="input-icon segment-search" style={{ minWidth: '16rem' }}>
              <span className="input-icon-addon"><IconSearch size={16} /></span>
              <input className="form-control form-control-sm" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search book, segment, error..." />
            </div>
          </div>
        </div>
        <div className="table-responsive">
          <table className="table table-vcenter card-table table-hover">
            <thead>
              <tr>
                <th>Book</th>
                <th>Segment</th>
                <th>Status</th>
                <th>Provider</th>
                <th>Voice</th>
                <th>Output</th>
                <th>Retries</th>
                <th>Error</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 && (
                <tr><td colSpan={9} className="text-center py-5 text-secondary">
                  No synth jobs found. Synth jobs appear when a book is uploaded and segmented.
                </td></tr>
              )}
              {filtered.map((job) => (
                <tr key={job.id}>
                  <td>
                    <Link to={`/books/${job.bookId}`} className="text-white text-decoration-none">{job.bookTitle}</Link>
                  </td>
                  <td><span className="mono-id">{job.segmentId}</span></td>
                  <td><span className={`badge bg-${statusColor(job.status)}-lt`}>{job.status}</span></td>
                  <td className="text-secondary small">{job.provider}</td>
                  <td className="text-secondary small">{job.voiceId}</td>
                  <td className="text-secondary small">{job.outputBytes ? `${(job.outputBytes / 1024).toFixed(0)} KB` : '—'}</td>
                  <td className="text-secondary">{job.retryCount || '—'}</td>
                  <td className="blocker-cell">{job.error ? <span className="text-warning small">{job.error.slice(0, 60)}</span> : '—'}</td>
                  <td className="text-secondary small">{fmtTime(job.updatedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}
