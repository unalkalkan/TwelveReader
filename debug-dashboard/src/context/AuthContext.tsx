import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import type { AuthState, AuthUser } from '../auth';
import { loadAuthState, saveAuthState, clearAuthState } from '../auth';
import { apiMe, apiRefreshSession } from '../api';

interface AuthContextValue extends AuthState {
  isLoading: boolean;
  login: (sessionToken: string, refreshToken: string, user: AuthUser) => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() => loadAuthState());
  const [isLoading, setIsLoading] = useState(true);

  // Validate stored session on mount
  useEffect(() => {
    let cancelled = false;
    async function validate() {
      if (state.sessionToken) {
        try {
          const me = await apiMe(state.sessionToken);
          if (!cancelled) {
            const role_name = (me as any).role_name || '';
            setState((prev) => ({
              ...prev,
              user: prev.user ? { ...prev.user, role_name } : { id: '', email: '', role_name },
              isAdmin: role_name === 'admin',
            }));
          }
        } catch {
          // Session invalid - try refresh
          if (!cancelled && state.refreshToken) {
            try {
              const refreshed = await apiRefreshSession(state.refreshToken);
              setState({
                sessionToken: refreshed.session_token,
                refreshToken: refreshed.refresh_token,
                user: state.user,
                isAdmin: state.isAdmin,
              });
              saveAuthState({
                sessionToken: refreshed.session_token,
                refreshToken: refreshed.refresh_token,
                user: state.user,
                isAdmin: state.isAdmin,
              });
            } catch {
              // Both failed - clear auth
              setState({ sessionToken: null, refreshToken: null, user: null, isAdmin: false });
              clearAuthState();
            }
          } else {
            setState({ sessionToken: null, refreshToken: null, user: null, isAdmin: false });
            clearAuthState();
          }
        }
      }
      if (!cancelled) setIsLoading(false);
    }
    validate();
    return () => { cancelled = true; };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const login = useCallback((sessionToken: string, refreshToken: string, user: AuthUser) => {
    const newState: AuthState = {
      sessionToken,
      refreshToken,
      user,
      isAdmin: user.role_name === 'admin',
    };
    setState(newState);
    saveAuthState(newState);
  }, []);

  const logout = useCallback(async () => {
    if (state.sessionToken) {
      try {
        await fetch(`${(import.meta.env as any).VITE_TWELVEREADER_API_URL?.replace(/\/$/, '') || window.location.origin}/api/v1/auth/logout`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${state.sessionToken}` },
        });
      } catch { /* ignore - we clear locally anyway */ }
    }
    setState({ sessionToken: null, refreshToken: null, user: null, isAdmin: false });
    clearAuthState();
  }, [state.sessionToken]);

  return (
    <AuthContext.Provider value={{ ...state, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
