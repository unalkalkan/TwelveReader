import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

const SERVER_URL_KEY = 'twelvereader_server_url';

// Official default server
export const OFFICIAL_SERVER_URL = 'https://twelvereader.com';

interface ServerConfigContextValue {
  /** Currently configured API base URL (without /api/v1 suffix). */
  serverUrl: string;
  /** Full API v1 base: serverUrl + '/api/v1'. */
  apiBase: string;
  /** Is the server currently validated (reachable)? null = unknown/not checked. */
  validated: boolean | null;
  /** Has the provider finished loading persisted config from storage. */
  initialized: boolean;
  /** Set a new server URL and persist it. Does not auto-validate. */
  setServerUrl: (url: string) => void;
  /** Validate the current server by calling /api/v1/server-info. Returns parsed response or throws. */
  validateServer: () => Promise<Record<string, unknown>>;
  /** Clear persisted server URL (reset to default). */
  resetServerUrl: () => void;
}

const ServerConfigContext = createContext<ServerConfigContextValue | null>(null);

function normalizeBaseUrl(raw: string): string {
  return raw.trim().replace(/\/+$/, '');
}

export function ServerConfigProvider({ children }: { children: ReactNode }) {
  const [serverUrl, setServerUrlState] = useState<string>(OFFICIAL_SERVER_URL);
  const [validated, setValidated] = useState<boolean | null>(null);
  const [initialized, setInitialized] = useState(false);

  // Load persisted server URL on mount
  useEffect(() => {
    (async () => {
      try {
        const stored = await AsyncStorage.getItem(SERVER_URL_KEY);
        if (stored && stored.trim().length > 0) {
          setServerUrlState(normalizeBaseUrl(stored));
        }
      } catch {
        // ignore load errors, use default
      } finally {
        setInitialized(true);
      }
    })();
  }, []);

  const setServerUrl = useCallback((url: string) => {
    const normalized = normalizeBaseUrl(url);
    setServerUrlState(normalized);
    setValidated(null);
    Promise.all([
      AsyncStorage.setItem(SERVER_URL_KEY, normalized),
      AsyncStorage.removeItem('twelvereader_server_validated'),
    ]).catch(() => {});
  }, []);

  const validateServer = useCallback(async (): Promise<Record<string, unknown>> => {
    // Use client.validateServerUrl which validates response via V1ServerInfoSchema
    const { validateServerUrl } = await import('../api/client');
    const info = await validateServerUrl(serverUrl);
    setValidated(true);
    return info as Record<string, unknown>;
  }, [serverUrl]);

  const resetServerUrl = useCallback(() => {
    setServerUrlState(OFFICIAL_SERVER_URL);
    setValidated(null);
    Promise.all([
      AsyncStorage.setItem(SERVER_URL_KEY, OFFICIAL_SERVER_URL),
      AsyncStorage.removeItem('twelvereader_server_validated'),
    ]).catch(() => {});
  }, []);

  return (
    <ServerConfigContext.Provider
      value={{
        serverUrl,
        apiBase: `${serverUrl}/api/v1`,
        validated,
        initialized,
        setServerUrl,
        validateServer,
        resetServerUrl,
      }}
    >
      {children}
    </ServerConfigContext.Provider>
  );
}

export function useServerConfig(): ServerConfigContextValue {
  const ctx = useContext(ServerConfigContext);
  if (!ctx) {
    throw new Error('useServerConfig must be used inside <ServerConfigProvider>');
  }
  return ctx;
}
