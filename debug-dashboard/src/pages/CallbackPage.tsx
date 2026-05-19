import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { IconKey } from '@tabler/icons-react';
import { apiVerifyMagicLink } from '../api';
import { useAuth } from '../context/AuthContext';

export function CallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { login, isAdmin } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    // Handle two flows:
    // Flow A: ?token=XXX (direct magic link token) — verify via API
    // Flow B: ?session=XXX&refresh=YYY&user=ZZZ (pre-verified from callback.html)

    const rawToken = searchParams.get('token');
    const sessionToken = searchParams.get('session');

    async function process() {
      if (rawToken) {
        // Flow A: Verify magic link token via API
        try {
          const result = await apiVerifyMagicLink(rawToken);
          if (cancelled) return;
          const user = result.user as Record<string, unknown>;
          login(
            result.session_token,
            result.refresh_token,
            {
              id: (user.id as string) || '',
              email: (user.email as string) || '',
              name: (user.name as string) || undefined,
              role_name: (user.role_name as string) || 'user',
            },
          );
        } catch (err) {
          if (!cancelled) {
            setError(err instanceof Error ? err.message : 'Invalid or expired token');
          }
        } finally {
          if (!cancelled) setLoading(false);
        }
      } else if (sessionToken) {
        // Flow B: Pre-verified session from callback.html — store and navigate
        const refreshToken = searchParams.get('refresh');
        const userParam = searchParams.get('user');

        if (!refreshToken || !userParam) {
          setError('Incomplete authentication data. Please use the login page.');
          setLoading(false);
          return;
        }

        try {
          const user = JSON.parse(decodeURIComponent(userParam));
          login(sessionToken, refreshToken, {
            id: user.id || '',
            email: user.email || '',
            name: user.name || undefined,
            role_name: user.role_name || 'user',
          });
        } catch (err) {
          setError('Invalid authentication data. Please use the login page.');
        } finally {
          setLoading(false);
        }
      } else {
        setError('No magic link token found. Please use the login page.');
        setLoading(false);
      }
    }

    process();
    return () => { cancelled = true; };
  }, [searchParams, login]);

  // After successful auth (login called), navigate based on role
  useEffect(() => {
    if (!loading) {
      if (isAdmin) {
        navigate('/');
      } else {
        navigate('/forbidden');
      }
    }
  }, [loading, isAdmin, navigate]);

  return (
    <div className="page">
      <div className="container-center container container-tight py-4">
        <div className="card card-md text-center" style={{ maxWidth: '440px', margin: '3rem auto' }}>
          <div className="card-body">
            {loading ? (
              <>
                <h2 className="card-title mb-3">
                  <IconKey size={24} className="me-2" />
                  Verifying magic link...
                </h2>
                <div className="text-secondary">Please wait while we authenticate you.</div>
              </>
            ) : error ? (
              <>
                <h2 className="card-title text-danger mb-3">Authentication failed</h2>
                <p className="text-secondary small">{error}</p>
                <button className="btn btn-link" onClick={() => navigate('/login')}>
                  Back to login
                </button>
              </>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}
