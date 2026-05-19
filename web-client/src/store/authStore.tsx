import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import type { AuthUser, AuthLoginResponse } from '../types/api';
import {
  requestMagicLink,
  verifyMagicLink,
  logoutApi,
  authMe,
  setAuthApiBase,
  clearAuthTokens,
} from '../api/auth';

// AsyncStorage keys
const SESSION_TOKEN_KEY = 'twelvereader_session_token';
const REFRESH_TOKEN_KEY = 'twelvereader_refresh_token';
const AUTH_USER_KEY = 'twelvereader_auth_user';

interface AuthContextValue {
  /** Currently authenticated user, or null if not logged in. */
  user: AuthUser | null;
  /** Role name (resolved by server), if available. */
  roleName: string | null;
  /** Is the user considered authenticated? */
  isAuthenticated: boolean;
  /** Has the provider finished initial load from storage. */
  initialized: boolean;
  /** Send a magic link to the given email address. */
  loginRequestMagicLink: (email: string) => Promise<void>;
  /** Verify a magic link token and complete login. */
  loginVerifyToken: (token: string) => Promise<void>;
  /** Log out: invalidate server session + clear local storage. */
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [roleName, setRoleName] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  // Prevent concurrent refresh calls
  const refreshingRef = useRef(false);
  // Store tokens in refs so they're always fresh inside callbacks
  const sessionTokenRef = useRef<string | null>(null);
  const refreshTokenRef = useRef<string | null>(null);

  // ── Persist / load ────────────────────────────────────────────────

  /** Save tokens + user to AsyncStorage. */
  const persistAuth = useCallback(async (loginResp: AuthLoginResponse) => {
    const results = await Promise.allSettled([
      AsyncStorage.setItem(SESSION_TOKEN_KEY, loginResp.session_token),
      AsyncStorage.setItem(REFRESH_TOKEN_KEY, loginResp.refresh_token),
      AsyncStorage.setItem(AUTH_USER_KEY, JSON.stringify(loginResp.user)),
    ]);

    // Fail fast if token storage fails
    const sessionResult = results[0];
    if (sessionResult.status === 'rejected') {
      throw sessionResult.reason;
    }

    sessionTokenRef.current = loginResp.session_token;
    refreshTokenRef.current = loginResp.refresh_token;
    setUser(loginResp.user);
  }, []);

  /** Clear all auth data from storage and state. */
  const clearAuth = useCallback(async () => {
    await clearAuthTokens();
    sessionTokenRef.current = null;
    refreshTokenRef.current = null;
    setUser(null);
    setRoleName(null);
  }, []);

  // ── Token refresh ────────────────────────────────────────────────

  /**
   * Attempt to refresh the current session using the stored refresh token.
   * Returns the new session token on success, or null on failure.
   */
  const tryRefresh = useCallback(async (): Promise<string | null> => {
    if (refreshingRef.current) {
      // Already refreshing — wait a bit and retry once (simple debounce guard)
      return null;
    }

    refreshingRef.current = true;
    try {
      // Delegate to the shared auth module
      const auth = await import('../api/auth');
      const newToken = await auth.attemptRefresh();
      if (newToken) {
        sessionTokenRef.current = newToken;
      }
      return newToken;
    } catch {
      return null;
    } finally {
      refreshingRef.current = false;
    }
  }, []);

  /**
   * Get the current session token, auto-refreshing if needed.
   * This is called by the API client before each authenticated request.
   */
  const getSessionToken = useCallback((): string | null => {
    return sessionTokenRef.current;
  }, []);

  // ── Public methods ───────────────────────────────────────────────

  const loginRequestMagicLink = useCallback(async (email: string) => {
    await requestMagicLink(email.trim().toLowerCase());
    // Note: on success, the user will receive a magic link in email.
    // They click it, the app opens via deep link with token, and we call loginVerifyToken.
    // Alternatively, for development / manual flow, they can paste the token.
  }, []);

  const loginVerifyToken = useCallback(async (token: string) => {
    const resp = await verifyMagicLink(token);
    await persistAuth(resp);

    // Resolve role name
    try {
      const meResp = await authMe(resp.session_token);
      setRoleName(meResp.role_name ?? null);
    } catch {
      // Non-fatal: we already have the user object from login
      setRoleName(null);
    }
  }, [persistAuth]);

  const logout = useCallback(async () => {
    // Try server-side invalidation first (best effort)
    if (sessionTokenRef.current) {
      try {
        await logoutApi(sessionTokenRef.current);
      } catch {
        // Ignore: still clear local state even if server call fails
      }
    }
    await clearAuth();
    // Clear the authenticated flag so routing redirects to login/server-select
    await AsyncStorage.removeItem('twelvereader_authenticated');
    // Also clear server validation so user can change server after logout
    await AsyncStorage.removeItem('twelvereader_server_validated');
  }, [clearAuth]);

  // ── Sync apiBase from the shared mutable reference ────────────────

  useEffect(() => {
    // Keep auth API base in sync with the main client's setApiBase calls.
    // We read it from the same module so they share state.
    (async () => {
      const { setApiBase, resolveApiBaseSync } = await import('../api/client');
      // If there's a sync resolver, use it; otherwise poll via our own copy
      if (resolveApiBaseSync) {
        setAuthApiBase(resolveApiBaseSync());
      }
    })().catch(() => {});
  }, []);

  // ── Initial load: restore session from AsyncStorage ───────────────

  useEffect(() => {
    (async () => {
      try {
        const [sessionToken, refreshToken, userJson] = await Promise.all([
          AsyncStorage.getItem(SESSION_TOKEN_KEY),
          AsyncStorage.getItem(REFRESH_TOKEN_KEY),
          AsyncStorage.getItem(AUTH_USER_KEY),
        ]);

        if (sessionToken && refreshToken && userJson) {
          sessionTokenRef.current = sessionToken;
          refreshTokenRef.current = refreshToken;
          setUser(JSON.parse(userJson) as AuthUser);

          // Try to verify session is still valid by calling /auth/me
          try {
            const meResp = await authMe(sessionToken);
            setUser(meResp.user);
            setRoleName(meResp.role_name ?? null);
          } catch {
            // Session expired or invalid — will be cleared on next attempt to use
            // Don't clear immediately; let the refresh mechanism try first
          }
        }
      } catch {
        // Storage load failed — start unauthenticated
      } finally {
        setInitialized(true);
      }
    })();
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        roleName,
        isAuthenticated: !!user && !!sessionTokenRef.current,
        initialized,
        loginRequestMagicLink,
        loginVerifyToken,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used inside <AuthProvider>');
  }
  return ctx;
}

// Re-export standalone token functions from the auth API module.
export { getSessionToken, attemptRefresh, clearAuthTokens } from '../api/auth';

/**
 * Check if user has a stored session token.
 */
export async function hasStoredSession(): Promise<boolean> {
  const token = await AsyncStorage.getItem(SESSION_TOKEN_KEY);
  return !!token;
}
