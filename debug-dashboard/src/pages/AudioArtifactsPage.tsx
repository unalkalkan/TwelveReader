import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { IconSearch } from '@tabler/icons-react';
import type { BookJourney, AudioArtifactValidation } from '../types';

function validationColor(status?: string) {
  if (!status) return 'secondary';
  if (['attached', 'completed'].includes(status)) return 'success';
  if (['stale'].includes(status)) return 'warning';
  if (['missing', 'invalid', 'playback_failed'].includes(status)) return 'danger';
  return 'secondary';
}

export function AudioArtifactsPage({ journeys }: { journeys: BookJourney[] }) {
  const [query, setQuery] = useState('');
  const [bookFilter, setBookFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const allArtifacts = useMemo(() => {
    return journeys.flatMap((j) =>
      j.segments
        .filter((s) => s.audioValidation || s.segment.audio_url || s.synthState !== 'not_created')
        .map((s) => ({
          bookId: j.book.id,
          bookTitle: j.book.title || j.book.id,
          segmentId: s.segment.id,
          segmentIndex: s.index,
          chapter: s.segment.chapter || '',
          person: s.segment.person || 'narrator',
          status: s.audioValidation?.status || s.audioState,
          format: s.audioValidation?.format || '—',
          path: s.audioValidation?.path || s.segment.audio_url || '',
          bytes: s.audioValidation?.bytes ?? 0,
          error: s.audioValidation?.error || s.blocker || '',
          checkedAt: s.audioValidation?.checked_at || '',
          voiceId: s.segment.voice_id || '',
          durationSec: s.audioDurationSec || 0,
        })),
    );
  }, [journeys]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return allArtifacts.filter((a) => {
      const matchSearch = !needle ||
        a.bookTitle.toLowerCase().includes(needle) ||
        a.segmentId.toLowerCase().includes(needle) ||
        (a.error && a.error.toLowerCase().includes(needle));
      const matchBook = bookFilter === 'all' || a.bookId === bookFilter;
      const matchStatus = statusFilter === 'all' || a.status === statusFilter;
      return matchSearch && matchBook && matchStatus;
    });
  }, [allArtifacts, query, bookFilter, statusFilter]);

  const stats = useMemo(() => ({
    total: allArtifacts.length,
    attached: allArtifacts.filter((a) => a.status === 'attached').length,
    stale: allArtifacts.filter((a) => a.status === 'stale').length,
    missing: allArtifacts.filter((a) => a.status === 'missing').length,
    invalid: allArtifacts.filter((a) => a.status === 'invalid' || a.status === 'playback_failed').length,
  }), [allArtifacts]);

  const statusOptions = useMemo(() => {
    const statuses = new Set(allArtifacts.map((a) => a.status));
    return Array.from(statuses).sort();
  }, [allArtifacts]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">Audio Artifacts</h1>
        <div className="text-secondary">{filtered.length} artifact(s) across {journeys.length} book(s)</div>
      </div>

      {/* Stats Row */}
      <section className="row row-cards mb-3">
        <StatCard title="Total" value={stats.total} tone="text-white" />
        <StatCard title="Attached" value={stats.attached} tone="text-success" />
        <StatCard title="Stale" value={stats.stale} tone="text-warning" />
        <StatCard title="Missing" value={stats.missing} tone="text-danger" />
        <StatCard title="Invalid/Failed" value={stats.invalid} tone="text-danger" />
      </section>

      {/* Artifacts Table */}
      <div className="card">
        <div className="card-header flex-wrap gap-2">
          <h3 className="card-title">Artifacts</h3>
          <div className="ms-auto d-flex gap-2 align-items-center">
            <select className="form-select form-select-sm" value={bookFilter} onChange={(e) => setBookFilter(e.target.value)} style={{ width: 'auto' }}>
              <option value="all">All books</option>
              {journeys.map((j) => (
                <option key={j.book.id} value={j.book.id}>{j.book.title || j.book.id}</option>
              ))}
            </select>
            <select className="form-select form-select-sm" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} style={{ width: 'auto' }}>
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
          <table className="table table-vcenter card-table">
            <thead>
              <tr>
                <th>Book</th>
                <th>Segment</th>
                <th>Chapter</th>
                <th>Persona</th>
                <th>Status</th>
                <th>Format</th>
                <th>Size</th>
                <th>Duration</th>
                <th>Voice</th>
                <th>Error</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 && (
                <tr><td colSpan={8} className="text-center py-5 text-secondary">
                  No audio artifacts found. Artifacts appear when TTS synthesis completes for book segments.
                </td></tr>
              )}
              {filtered.slice(0, 200).map((a) => (
                <tr key={`${a.bookId}-${a.segmentId}`}>
                  <td><Link to={`/books/${a.bookId}`} className="text-white text-decoration-none">{a.bookTitle}</Link></td>
                  <td><span className="mono-id">{a.segmentId}</span></td>
                  <td className="text-secondary small">{a.chapter || '—'}</td>
                  <td><span className="badge bg-secondary-lt">{a.person}</span></td>
                  <td><span className={`badge bg-${validationColor(a.status)}-lt`}>{a.status}</span></td>
                  <td className="text-secondary small">{a.format}</td>
                  <td className="text-secondary small">{a.bytes ? `${(a.bytes / 1024).toFixed(0)} KB` : '—'}</td>
                  <td className="text-secondary small">{a.durationSec ? `${a.durationSec}s` : '—'}</td>
                  <td className="text-secondary small">{a.voiceId || '—'}</td>
                  <td className="blocker-cell">{a.error ? <span className="text-warning small">{a.error.slice(0, 60)}</span> : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

function StatCard({ title, value, tone }: { title: string; value: number; tone: string }) {
  return (
    <div className="col-sm-6 col-lg-3 col-xl-2">
      <div className="card metric-card">
        <div className="card-body">
          <div className={`h4 mb-0 ${tone}`}>{value}</div>
          <div className="text-secondary xsmall">{title}</div>
        </div>
      </div>
    </div>
  );
}
