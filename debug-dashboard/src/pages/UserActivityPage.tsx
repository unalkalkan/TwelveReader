import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { IconSearch } from '@tabler/icons-react';
import type { BookJourney, PlaybackEvent } from '../types';

function fmtTime(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', month: 'short', day: 'numeric' });
}

function eventTypeColor(type?: string) {
  if (!type) return 'secondary';
  if (['play', 'read', 'book_opened', 'segment_opened'].includes(type)) return 'primary';
  if (['complete', 'completed'].includes(type)) return 'success';
  if (['failed', 'error'].includes(type)) return 'danger';
  if (['pause', 'paused'].includes(type)) return 'warning';
  return 'secondary';
}

export function UserActivityPage({ journeys, events }: { journeys: BookJourney[]; events: any[] }) {
  const [query, setQuery] = useState('');
  const [bookFilter, setBookFilter] = useState<string>('all');

  const allEvents = useMemo(() => {
    return events
      .filter((e) => e.scope === 'user' || e.source === 'playback')
      .sort((a: any, b: any) => new Date(b.created_at || b.at).getTime() - new Date(a.created_at || a.at).getTime());
  }, [events]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return allEvents.filter((e: any) => {
      const matchSearch = !needle ||
        (e.title || '').toLowerCase().includes(needle) ||
        (e.detail || '').toLowerCase().includes(needle) ||
        (e.book_id || e.bookId || '').toLowerCase().includes(needle) ||
        (e.event_type || e.EventType || '').toLowerCase().includes(needle);
      const matchBook = bookFilter === 'all' || e.book_id === bookFilter || e.bookId === bookFilter;
      return matchSearch && matchBook;
    });
  }, [allEvents, query, bookFilter]);

  const bookOptions = useMemo(() => journeys.map((j) => j.book.id), [journeys]);

  // Build a per-book summary from journeys
  const bookSummaries = useMemo(() => {
    return journeys.filter((j) => j.userReadSegment > 0 || j.userListenedSegment > 0).map((j) => ({
      bookId: j.book.id,
      bookTitle: j.book.title || j.book.id,
      journeyState: j.userProgress?.journey_state || 'derived',
      userReadSegment: j.userReadSegment,
      userListenedSegment: j.userListenedSegment,
      playbackFailures: j.failedAudioCount,
      totalSegments: j.book.total_segments || j.segments.length,
    }));
  }, [journeys]);

  return (
    <>
      <div className="page-header d-print-none mb-3">
        <h1 className="page-title">User Activity</h1>
        <div className="text-secondary">{filtered.length} event(s) across {bookSummaries.length} active user journey(s)</div>
      </div>

      {/* User Journey Summary */}
      {bookSummaries.length > 0 && (
        <div className="card mb-3">
          <div className="card-header"><h3 className="card-title">User Journeys</h3></div>
          <div className="table-responsive">
            <table className="table table-vcenter card-table">
              <thead>
                <tr>
                  <th>Book</th>
                  <th>Journey State</th>
                  <th>User Read</th>
                  <th>User Listen</th>
                  <th>Playback Failures</th>
                  <th>Total Segments</th>
                </tr>
              </thead>
              <tbody>
                {bookSummaries.map((s) => (
                  <tr key={s.bookId}>
                    <td><Link to={`/books/${s.bookId}`} className="text-white text-decoration-none fw-semibold">{s.bookTitle}</Link></td>
                    <td><span className={`badge bg-${journeyStateColor(s.journeyState)}-lt`}>{s.journeyState}</span></td>
                    <td className="text-secondary">{s.userReadSegment || '—'}</td>
                    <td className="text-secondary">{s.userListenedSegment || '—'}</td>
                    <td>{s.playbackFailures > 0 ? <span className="badge bg-danger-lt">{s.playbackFailures}</span> : '0'}</td>
                    <td className="text-secondary">{s.totalSegments}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Events Timeline */}
      <div className="card">
        <div className="card-header flex-wrap gap-2">
          <h3 className="card-title">Playback & Activity Events</h3>
          <div className="ms-auto d-flex gap-2 align-items-center">
            <select className="form-select form-select-sm" value={bookFilter} onChange={(e) => setBookFilter(e.target.value)} style={{ width: 'auto' }}>
              <option value="all">All books</option>
              {bookOptions.map((id) => (
                <option key={id} value={id}>{journeys.find((j) => j.book.id === id)?.book.title || id}</option>
              ))}
            </select>
            <div className="input-icon segment-search" style={{ minWidth: '16rem' }}>
              <span className="input-icon-addon"><IconSearch size={16} /></span>
              <input className="form-control form-control-sm" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search events..." />
            </div>
          </div>
        </div>
        <div className="table-responsive">
          <table className="table table-vcenter card-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Type</th>
                <th>Book</th>
                <th>Segment</th>
                <th>Detail</th>
                <th>User ID</th>
              </tr>
            </thead>
            <tbody>
              {filtered.slice(0, 100).map((e: any, i) => (
                <tr key={`${e.id || e.book_id}-${i}`}>
                  <td className="text-secondary small">{fmtTime(e.created_at || e.at)}</td>
                  <td><span className={`badge bg-${eventTypeColor(e.event_type || e.EventType)}-lt`}>{e.event_type || e.EventType || 'info'}</span></td>
                  <td>{e.book_id || e.bookId ? <Link to={`/books/${e.book_id || e.bookId}`} className="text-white text-decoration-none">{e.book_id || e.bookId}</Link> : '—'}</td>
                  <td><span className="mono-id small">{e.segment_id || e.SegmentId || '—'}</span></td>
                  <td className="text-secondary small">{e.error || e.detail || '—'}</td>
                  <td className="text-secondary small">{e.user_id || e.UserID || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

function journeyStateColor(state: string) {
  if (['completed', 'finished'].includes(state)) return 'success';
  if (['reading', 'listening', 'opened'].includes(state)) return 'primary';
  if (['stuck', 'failed'].includes(state)) return 'danger';
  if (['paused', 'abandoned'].includes(state)) return 'warning';
  return 'secondary';
}
