import { useState } from 'react';
import { IconMail, IconKey } from '@tabler/icons-react';
import { apiRequestMagicLink, apiVerifyMagicLink } from '../api';
import { useAuth } from '../context/AuthContext';

type Step = 'email' | 'link-sent' | 'manual-token';

export function LoginPage() {
  const { login } = useAuth();
  const [step, setStep] = useState<Step>('email');
  const [email, setEmail] = useState('');
  const [manualToken, setManualToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleRequestLink() {
    setError(null);
    if (!email.trim()) {
      setError('Email is required');
      return;
    }
    setLoading(true);
    try {
      await apiRequestMagicLink(email.trim());
      setStep('link-sent');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send magic link');
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyToken() {
    setError(null);
    if (!manualToken.trim()) {
      setError('Token is required');
      return;
    }
    setLoading(true);
    try {
      const result = await apiVerifyMagicLink(manualToken.trim());
      const user = result.user as Record<string, unknown>;
      login(
        result.session_token,
        result.refresh_token,
        {
          id: (user.id as string) || '',
          email: (user.email as string) || email,
          name: (user.name as string) || undefined,
          role_name: (user.role_name as string) || 'user',
        },
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid or expired token');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page">
      <div className="container-center container container-tight py-4">
        <div className="card card-md" style={{ maxWidth: '440px', margin: '3rem auto' }}>
          <div className="card-body">
            <h2 className="card-title text-center mb-3">
              <IconKey size={24} className="me-2" />
              Admin Dashboard
            </h2>
            <p className="text-secondary text-center small mb-3">TwelveReader Debug &amp; Administration</p>

            {error && (
              <div className="alert alert-sm alert-danger" role="alert">{error}</div>
            )}

            {step === 'email' && (
              <>
                <label className="form-label">Email address</label>
                <div className="form-floating mb-3">
                  <input
                    type="email"
                    className="form-control"
                    placeholder="admin@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleRequestLink()}
                    autoFocus
                  />
                  <label>Email address</label>
                </div>
                <button
                  className="btn btn-primary w-100"
                  onClick={handleRequestLink}
                  disabled={loading}
                >
                  {loading ? 'Sending...' : (
                    <>
                      <IconMail size={16} className="me-1" />
                      Send Magic Link
                    </>
                  )}
                </button>
              </>
            )}

            {step === 'link-sent' && (
              <div className="text-center">
                <div className="mb-3">
                  <div className="text-success mb-2">Magic link requested for <strong>{email}</strong></div>
                  <p className="small text-secondary mb-3">
                    In dev mode, the token is logged to the server console. Find the raw token from the log output and paste it below.
                  </p>
                </div>

                <label className="form-label text-start">Paste magic link token</label>
                <input
                  type="text"
                  className="form-control mb-2"
                  placeholder="64-character hex token from server logs"
                  value={manualToken}
                  onChange={(e) => setManualToken(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleVerifyToken()}
                />
                <button
                  className="btn btn-success w-100 mb-2"
                  onClick={handleVerifyToken}
                  disabled={loading || !manualToken.trim()}
                >
                  {loading ? 'Verifying...' : 'Verify Token & Sign In'}
                </button>
                <button
                  className="btn btn-link btn-sm"
                  onClick={() => setStep('email')}
                >
                  Use a different email
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
