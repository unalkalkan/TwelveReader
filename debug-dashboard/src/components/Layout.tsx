import { useState, type ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  IconActivity,
  IconBook2,
  IconDatabase,
  IconHeadphones,
  IconPlayerPlay,
  IconRefresh,
  IconServer,
  IconUserCheck,
  IconWaveSine,
} from '@tabler/icons-react';

import type { ProvidersResponse } from '../types';

type LayoutProps = {
  children: ReactNode;
  apiConnected: boolean;
  sseConnected?: boolean;
  health?: HealthStatus;
  providers?: ProvidersResponse;
  lastUpdated: string;
};

interface HealthStatus { status: string }

function fmtTime(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

const navSections = [
  {
    label: 'System',
    items: [
      { to: '/', label: 'Overview', icon: IconActivity },
      { to: '/books', label: 'Books', icon: IconBook2 },
    ],
  },
  {
    label: 'Inspection',
    items: [
      { to: '/segments', label: 'Segments', icon: IconDatabase },
      { to: '/synth-jobs', label: 'Synth Jobs', icon: HeadphonesIcon },
      { to: '/audio-artifacts', label: 'Audio Artifacts', icon: IconPlayerPlay },
      { to: '/user-activity', label: 'User Activity', icon: IconUserCheck },
    ],
  },
];

function HeadphonesIcon(props: React.SVGProps<SVGSVGElement>) {
  return <IconHeadphones {...props} />;
}

export function Layout({ children, apiConnected, sseConnected, health, providers, lastUpdated }: LayoutProps) {
  const location = useLocation();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <div className="page debug-shell">
      <aside className={`debug-sidebar navbar navbar-vertical navbar-expand-lg ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="container-fluid flex-column align-items-stretch">
          <div className="navbar-brand navbar-brand-autodark justify-content-start gap-2">
            <span className="brand-glyph"><IconWaveSine size={18} /></span>
            <div>
              <div className="fw-bold">TwelveReader</div>
              <div className="text-secondary small">Debug Dashboard</div>
            </div>
          </div>

          {navSections.map((section) => (
            <div key={section.label}>
              <div className="nav-section-label">{section.label}</div>
              <div className="navbar-nav">
                {section.items.map((item) => {
                  const isActive = location.pathname === item.to || (item.to !== '/' && location.pathname.startsWith(item.to));
                  return (
                    <Link key={item.to} to={item.to} className={`nav-link ${isActive ? 'active' : ''}`}>
                      <span className="nav-link-icon d-md-none d-lg-inline-block">
                        <item.icon size={17} />
                      </span>
                      <span className="nav-link-title">{item.label}</span>
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}

          <div className="mt-auto p-3 sidebar-status">
            <span className={`status status-dot ${apiConnected ? 'status-green' : 'status-red'}`} />
            <div>
              <div className="small fw-semibold">{apiConnected ? 'API connected' : 'API disconnected'}</div>
              {sseConnected && <div className="text-success xsmall"><span className="status-dot status-green" /> SSE push active</div>}
              <div className="text-secondary small">Updated {fmtTime(lastUpdated)}</div>
            </div>
          </div>
        </div>
      </aside>

      <div className="page-wrapper debug-main">
        <header className="navbar navbar-expand-md debug-topbar">
          <div className="container-xl">
            <div className="text-secondary small d-none d-xl-inline">Live State Inspector</div>
            <div className="ms-auto d-flex align-items-center gap-2 flex-wrap">
              {health && (
                <span className={`badge bg-${statusColor(health.status)}-lt`}>
                  <IconServer size={14} /> API {health.status}
                </span>
              )}
              {providers && (
                <>
                  <span className="badge bg-blue-lt">TTS {providers?.tts?.length ?? 0}</span>
                  <span className="badge bg-secondary-lt">LLM {providers?.llm?.length ?? 0}</span>
                </>
              )}
              <button className="btn btn-primary btn-sm" onClick={() => window.location.reload()}>
                <IconRefresh size={15} /> Refresh
              </button>
            </div>
          </div>
        </header>

        <main className="page-body">
          <div className="container-xl">{children}</div>
        </main>
      </div>
    </div>
  );
}

function statusColor(status?: string) {
  if (!status) return 'secondary';
  if (['healthy', 'ready', 'synthesized', 'completed', 'success'].includes(status)) return 'success';
  if (['synthesizing', 'segmenting', 'parsing', 'running', 'in_progress'].includes(status)) return 'primary';
  if (['voice_mapping', 'queued', 'stale', 'warning'].includes(status)) return 'warning';
  if (['error', 'synthesis_error', 'failed', 'danger', 'unhealthy'].includes(status)) return 'danger';
  return 'secondary';
}
