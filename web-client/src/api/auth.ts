import {
  AuthLoginResponseSchema,
  AuthRefreshResponseSchema,
  AuthMeResponseSchema,
  type AuthLoginResponse,
  type AuthRefreshResponse,
  type AuthMeResponse,
} from '../types/api';

// Dynamic base URL — reads from the same mutable reference as client.ts.
function getApiBase(): string {
  // We import lazily to avoid circular deps at module init time.
  const configuredApiUrl =
    (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
      ?.env?.EXPO_PUBLIC_API_URL;
  const inferredApiUrl =
    typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';
  return (configuredApiUrl && configuredApiUrl.trim().length > 0
    ? configuredApiUrl
    : inferredApiUrl
  ).replace(/\/+$/, '') + '/api/v1';
}

/** Override detected API base so auth calls always hit the same server as the rest of the app. */
export function setAuthApiBase(base: string): void {
  // We use a module-level variable that mirrors what client.ts maintains.
  (_authApiBaseOverride as any) = base.replace(/\/+$/, '');
}

let _authApiBaseOverride: string | null = null;

function resolveAuthApiBase(): string {
  if (_authApiBaseOverride) return _authApiBaseOverride;
  return getApiBase();
}

// ── helpers ─────────────────────────────────────────────────────────────

async function authRequest<T>(
  url: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: 'Request failed',
      code: 'UNKNOWN_ERROR',
    }));
    throw new Error(error.error || `Request failed with status ${response.status}`);
  }

  return response.json();
}

// ── auth endpoints ──────────────────────────────────────────────────────

/** Request a magic link to be sent to the given email. */
export async function requestMagicLink(email: string): Promise<{ message: string }> {
  return authRequest<{ message: string }>(
    `${resolveAuthApiBase()}/auth/request`,
    {
      method: 'POST',
      body: JSON.stringify({ email }),
    },
  );
}

/**
 * Verify a magic link token (from deep link or manual input).
 * Returns user + session_token + refresh_token.
 */
export async function verifyMagicLink(token: string): Promise<AuthLoginResponse> {
  const data = await authRequest<unknown>(
    `${resolveAuthApiBase()}/auth/verify?token=${encodeURIComponent(token)}`,
    { method: 'GET' },
  );
  return AuthLoginResponseSchema.parse(data);
}

/** Refresh an expired session using a refresh token. Returns new session + refresh tokens. */
export async function refreshSession(refreshToken: string): Promise<AuthRefreshResponse> {
  const data = await authRequest<unknown>(
    `${resolveAuthApiBase()}/auth/refresh`,
    {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    },
  );
  return AuthRefreshResponseSchema.parse(data);
}

/** Logout: invalidate the current session. Requires a valid session token. */
export async function logoutApi(sessionToken: string): Promise<{ message: string }> {
  return authRequest<{ message: string }>(
    `${resolveAuthApiBase()}/auth/logout`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${sessionToken}`,
      },
    },
  );
}

/** Get current authenticated user info. Requires a valid session token. */
export async function authMe(sessionToken: string): Promise<AuthMeResponse> {
  const data = await authRequest<unknown>(
    `${resolveAuthApiBase()}/auth/me`,
    {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${sessionToken}`,
      },
    },
  );
  return AuthMeResponseSchema.parse(data);
}

// ── Token helpers (used by client.ts for auto-refresh) ─────────────

import AsyncStorage from '@react-native-async-storage/async-storage';

const SESSION_TOKEN_KEY = 'twelvereader_session_token';
const REFRESH_TOKEN_KEY = 'twelvereader_refresh_token';
const AUTH_USER_KEY = 'twelvereader_auth_user';

/** Read the current session token from storage. Used by client.ts for Bearer auth. */
export async function getSessionToken(): Promise<string | null> {
  try {
    return await AsyncStorage.getItem(SESSION_TOKEN_KEY);
  } catch {
    return null;
  }
}

/**
 * Attempt to refresh the session using the stored refresh token.
 * Returns new session token on success, null on failure.
 * Clears all auth data if refresh fails (session is dead).
 */
export async function attemptRefresh(): Promise<string | null> {
  const refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
  if (!refreshToken) return null;

  try {
    const resp = await refreshSession(refreshToken);
    // Persist new tokens
    await Promise.allSettled([
      AsyncStorage.setItem(SESSION_TOKEN_KEY, resp.session_token),
      AsyncStorage.setItem(REFRESH_TOKEN_KEY, resp.refresh_token),
    ]);
    return resp.session_token;
  } catch {
    // Refresh failed — session is dead, clear everything
    await clearAuthTokens();
    return null;
  }
}

/** Clear all auth tokens from storage. */
export async function clearAuthTokens(): Promise<void> {
  await Promise.allSettled([
    AsyncStorage.removeItem(SESSION_TOKEN_KEY),
    AsyncStorage.removeItem(REFRESH_TOKEN_KEY),
    AsyncStorage.removeItem(AUTH_USER_KEY),
  ]);
}
