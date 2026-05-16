import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

const FAVORITES_KEY = 'voice_favorites';
const RECENTS_KEY = 'voice_recents';
const MAX_RECENTS = 20;

interface FavoritesContextValue {
  favoriteIds: Set<string>;
  recentIds: string[];
  isFavorite: (voiceId: string) => boolean;
  toggleFavorite: (voiceId: string) => void;
  addRecent: (voiceId: string) => void;
}

const FavoritesContext = createContext<FavoritesContextValue | null>(null);

export function FavoritesProvider({ children }: { children: ReactNode }) {
  const [favoriteIds, setFavoriteIds] = useState<Set<string>>(new Set());
  const [recentIds, setRecentIds] = useState<string[]>([]);

  // Load from storage on mount
  useEffect(() => {
    (async () => {
      try {
        const [favRaw, recRaw] = await Promise.all([
          AsyncStorage.getItem(FAVORITES_KEY),
          AsyncStorage.getItem(RECENTS_KEY),
        ]);
        if (favRaw) setFavoriteIds(new Set(JSON.parse(favRaw)));
        if (recRaw) setRecentIds(JSON.parse(recRaw));
      } catch {
        // ignore load errors
      }
    })();
  }, []);

  const isFavorite = useCallback(
    (voiceId: string) => favoriteIds.has(voiceId),
    [favoriteIds],
  );

  const toggleFavorite = useCallback((voiceId: string) => {
    setFavoriteIds((prev) => {
      const next = new Set(prev);
      if (next.has(voiceId)) {
        next.delete(voiceId);
      } else {
        next.add(voiceId);
      }
      AsyncStorage.setItem(FAVORITES_KEY, JSON.stringify([...next])).catch(
        () => {},
      );
      return next;
    });
  }, []);

  const addRecent = useCallback((voiceId: string) => {
    setRecentIds((prev) => {
      const next = [voiceId, ...prev.filter((id) => id !== voiceId)].slice(
        0,
        MAX_RECENTS,
      );
      AsyncStorage.setItem(RECENTS_KEY, JSON.stringify(next)).catch(() => {});
      return next;
    });
  }, []);

  return (
    <FavoritesContext.Provider
      value={{ favoriteIds, recentIds, isFavorite, toggleFavorite, addRecent }}
    >
      {children}
    </FavoritesContext.Provider>
  );
}

export function useFavorites(): FavoritesContextValue {
  const ctx = useContext(FavoritesContext);
  if (!ctx) {
    throw new Error('useFavorites must be used inside <FavoritesProvider>');
  }
  return ctx;
}
