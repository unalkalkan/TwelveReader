import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { IconSearch, IconAlertTriangle } from '@tabler/icons-react';
import type { BookJourney } from '../types';

function statusColor(status?: string) {
  if (!status) return 'secondary';
  if (['healthy', 'ready', 'synthesized', 'completed', 'success'].includes(status)) return 'success';
  if (['synthesizing', 'segmenting', 'parsing', 'running', 'in_progress'].includes(status)) return 'primary';
  if (['voice_mapping', 'queued', 'stale', 'warning'].includes(status)) return 'warning';
  if (['error', 'synthesis_error', 'failed', 'danger', 'unhealthy'].includes(status)) return 'danger';
  return 'secondary';
}

function readinessTone(score: number) {
  if (score >= 90) return 'success';
  if (score >= 60) return 'primary';
  if (score >= 30) return 'warning';
  return 'danger';
}

export function BooksListPage({ journeys }: { journeys: BookJourney[] }) {
  const [query, setQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return journeys.filter((j) => {
      const matchSearch = !needle ||
        (j.book.title || '').toLowerCase().includes(needle) ||
        (j.book.id || '').toLowerCase().includes(needle) ||
        (j.book.author || '').toLowerCase().includes(needle);
      const matchStatus = filterStatus === 'all' || j.book.status === filterStatus;
      return matchSearch && matchStatus;
    });
  }, [journeys, query, filterStatus]);

  const statusOptions = useMemo(() => {
    const statuses = new Set(journeys.map((j) => j.book.status).filter(Boolean));
    return Array.from(statuses).sort();
  }, [journeys]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">Books</h1>
        <div className="text-secondary">{filtered.length} book(s) in the system</div>
      </div>

      <div className="row row-cards">
        {filtered.map((j) => (
          <div key={j.book.id} className="col-sm-6 col-lg-4 col-xl-3">
            <BookCard journey={j} />
          </div>
        ))}

        {filtered.length === 0 && (
          <div className="col-12">
            <div className="card">
              <div className="card-body text-center py-5">
                <div className="text-secondary mb-2">No books found</div>
                <div className="text-secondary small">{journeys.length === 0 ? 'Upload a book to TwelveReader to see it here.' : 'Try adjusting your search or filters'}</div>
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

function BookCard({ journey }: { journey: BookJourney }) {
  const total = journey.book.total_segments || journey.segments.length || 1;

  return (
    <div className="card book-card">
      <div className="card-body p-3">
        <div className="d-flex justify-content-between align-items-start mb-2 gap-2">
          <Link to={`/books/${journey.book.id}`} className="fw-semibold text-white text-decoration-none flex-grow-1 text-truncate">
            {journey.book.title || journey.book.id}
          </Link>
          <span className={`badge bg-${statusColor(journey.book.status)}-lt`}>{journey.book.status}</span>
        </div>

        <div className="text-secondary small mb-3">
          {journey.book.author || 'Unknown'} · {journey.book.id}
        </div>

        <div className="row g-2 mb-3">
          <div className="col-6">
            <div className="mini-stat">
              <div className="fw-bold">{journey.readinessScore}%</div>
              <div className="text-secondary xsmall">Readiness</div>
            </div>
          </div>
          <div className="col-6">
            <div className="mini-stat">
              <div className="fw-bold">{journey.audioReadyCount}/{total}</div>
              <div className="text-secondary xsmall">Audio</div>
            </div>
          </div>
        </div>

        <div className="progress progress-sm mb-2">
          <div className={`progress-bar bg-${readinessTone(journey.readinessScore)}`} style={{ width: `${journey.readinessScore}%` }} />
        </div>

        <div className="d-flex justify-content-between text-secondary xsmall">
          <span>Text: {journey.textReadyCount}</span>
          <span>User read: {journey.userReadSegment || '—'}</span>
          <span>User listen: {journey.userListenedSegment || '—'}</span>
        </div>

        {journey.blocker && (
          <div className="mt-2">
            <span className="badge bg-warning-lt">
              <IconAlertTriangle size={10} /> {journey.blocker.slice(0, 50)}
              {journey.blocker.length > 50 ? '…' : ''}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
