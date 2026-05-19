// Authentication types for debug dashboard
export interface AuthUser {
  id: string;
  email: string;
  name?: string;
  role_name: string;
}

export interface AuthState {
  sessionToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  isAdmin: boolean;
}

// Storage keys (prefix to avoid collisions)
const STORAGE_KEY_SESSION = 'twelvereader_debug_session_token';
const STORAGE_KEY_REFRESH = 'twelvereader_debug_refresh_token';
const STORAGE_KEY_USER = 'twelvereader_debug_user';

export function loadAuthState(): AuthState {
  try {
    const sessionToken = localStorage.getItem(STORAGE_KEY_SESSION);
    const refreshToken = localStorage.getItem(STORAGE_KEY_REFRESH);
    const userRaw = localStorage.getItem(STORAGE_KEY_USER);
    const user = userRaw ? (JSON.parse(userRaw) as AuthUser) : null;
    return {
      sessionToken,
      refreshToken,
      user,
      isAdmin: user?.role_name === 'admin',
    };
  } catch {
    return { sessionToken: null, refreshToken: null, user: null, isAdmin: false };
  }
}

export function saveAuthState(state: AuthState): void {
  if (state.sessionToken) {
    localStorage.setItem(STORAGE_KEY_SESSION, state.sessionToken);
  } else {
    localStorage.removeItem(STORAGE_KEY_SESSION);
  }
  if (state.refreshToken) {
    localStorage.setItem(STORAGE_KEY_REFRESH, state.refreshToken);
  } else {
    localStorage.removeItem(STORAGE_KEY_REFRESH);
  }
  if (state.user) {
    localStorage.setItem(STORAGE_KEY_USER, JSON.stringify(state.user));
  } else {
    localStorage.removeItem(STORAGE_KEY_USER);
  }
}

export function clearAuthState(): void {
  localStorage.removeItem(STORAGE_KEY_SESSION);
  localStorage.removeItem(STORAGE_KEY_REFRESH);
  localStorage.removeItem(STORAGE_KEY_USER);
}
