import React, {
  createContext,
  useContext,
  useReducer,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

// ── Types ───────────────────────────────────────────────────────────────

export interface PlaybackState {
  currentBookId: string | null;
  currentSegmentIndex: number;
  isPlaying: boolean;
  playbackSpeed: number;
  /** milliseconds elapsed in current segment audio */
  elapsedMs: number;
  /** total duration of current segment audio in ms */
  durationMs: number;
}

type PlaybackAction =
  | { type: 'SET_BOOK'; bookId: string; segmentIndex?: number }
  | { type: 'SEEK_SEGMENT'; index: number }
  | { type: 'PLAY' }
  | { type: 'PAUSE' }
  | { type: 'TOGGLE' }
  | { type: 'STOP' }
  | { type: 'SET_SPEED'; speed: number }
  | { type: 'SET_ELAPSED'; ms: number }
  | { type: 'SET_DURATION'; ms: number }
  | { type: 'RESTORE'; state: Partial<PlaybackState> };

// ── Reducer ─────────────────────────────────────────────────────────────

const initialState: PlaybackState = {
  currentBookId: null,
  currentSegmentIndex: 0,
  isPlaying: false,
  playbackSpeed: 1.0,
  elapsedMs: 0,
  durationMs: 0,
};

function playbackReducer(
  state: PlaybackState,
  action: PlaybackAction,
): PlaybackState {
  switch (action.type) {
    case 'SET_BOOK':
      return {
        ...state,
        currentBookId: action.bookId,
        currentSegmentIndex: action.segmentIndex ?? 0,
        isPlaying: false,
        elapsedMs: 0,
        durationMs: 0,
      };
    case 'SEEK_SEGMENT':
      return {
        ...state,
        currentSegmentIndex: action.index,
        elapsedMs: 0,
        durationMs: 0,
      };
    case 'PLAY':
      return { ...state, isPlaying: true };
    case 'PAUSE':
      return { ...state, isPlaying: false };
    case 'TOGGLE':
      return { ...state, isPlaying: !state.isPlaying };
    case 'STOP':
      return {
        ...state,
        isPlaying: false,
        elapsedMs: 0,
      };
    case 'SET_SPEED':
      return { ...state, playbackSpeed: action.speed };
    case 'SET_ELAPSED':
      return { ...state, elapsedMs: action.ms };
    case 'SET_DURATION':
      return { ...state, durationMs: action.ms };
    case 'RESTORE':
      return { ...state, ...action.state };
    default:
      return state;
  }
}

// ── Context ─────────────────────────────────────────────────────────────

interface PlaybackContextValue {
  state: PlaybackState;
  dispatch: React.Dispatch<PlaybackAction>;
  setBook: (bookId: string, segmentIndex?: number) => void;
  seekToSegment: (index: number) => void;
  play: () => void;
  pause: () => void;
  togglePlayback: () => void;
  stop: () => void;
  setSpeed: (speed: number) => void;
  cycleSpeed: () => void;
  setElapsed: (ms: number) => void;
  setDuration: (ms: number) => void;
}

const PlaybackContext = createContext<PlaybackContextValue | null>(null);

const STORAGE_KEY = 'playback_state';
const SPEED_STEPS = [0.5, 0.75, 1.0, 1.25, 1.5, 2.0];

// ── Provider ────────────────────────────────────────────────────────────

export function PlaybackProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(playbackReducer, initialState);
  const restored = useRef(false);

  // Restore persisted state on mount
  useEffect(() => {
    (async () => {
      try {
        const raw = await AsyncStorage.getItem(STORAGE_KEY);
        if (raw) {
          const saved = JSON.parse(raw);
          dispatch({
            type: 'RESTORE',
            state: {
              currentBookId: saved.currentBookId ?? null,
              currentSegmentIndex: saved.currentSegmentIndex ?? 0,
              playbackSpeed: saved.playbackSpeed ?? 1.0,
              // Don't restore isPlaying — start paused
            },
          });
        }
      } catch {
        // Ignore restore errors
      } finally {
        restored.current = true;
      }
    })();
  }, []);

  // Persist when key fields change
  useEffect(() => {
    if (!restored.current) return;
    AsyncStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        currentBookId: state.currentBookId,
        currentSegmentIndex: state.currentSegmentIndex,
        playbackSpeed: state.playbackSpeed,
      }),
    ).catch(() => {});
  }, [state.currentBookId, state.currentSegmentIndex, state.playbackSpeed]);

  // ── Actions ─────────────────────────────────────────────────────────
  const setBook = useCallback(
    (bookId: string, segmentIndex?: number) =>
      dispatch({ type: 'SET_BOOK', bookId, segmentIndex }),
    [],
  );

  const seekToSegment = useCallback(
    (index: number) => dispatch({ type: 'SEEK_SEGMENT', index }),
    [],
  );

  const play = useCallback(() => dispatch({ type: 'PLAY' }), []);
  const pause = useCallback(() => dispatch({ type: 'PAUSE' }), []);
  const togglePlayback = useCallback(() => dispatch({ type: 'TOGGLE' }), []);
  const stop = useCallback(() => dispatch({ type: 'STOP' }), []);

  const setSpeed = useCallback(
    (speed: number) => dispatch({ type: 'SET_SPEED', speed }),
    [],
  );

  const cycleSpeed = useCallback(() => {
    dispatch({
      type: 'SET_SPEED',
      speed: (() => {
        const idx = SPEED_STEPS.indexOf(state.playbackSpeed);
        return SPEED_STEPS[(idx + 1) % SPEED_STEPS.length];
      })(),
    });
  }, [state.playbackSpeed]);

  const setElapsed = useCallback(
    (ms: number) => dispatch({ type: 'SET_ELAPSED', ms }),
    [],
  );

  const setDuration = useCallback(
    (ms: number) => dispatch({ type: 'SET_DURATION', ms }),
    [],
  );

  return (
    <PlaybackContext.Provider
      value={{
        state,
        dispatch,
        setBook,
        seekToSegment,
        play,
        pause,
        togglePlayback,
        stop,
        setSpeed,
        cycleSpeed,
        setElapsed,
        setDuration,
      }}
    >
      {children}
    </PlaybackContext.Provider>
  );
}

// ── Hook ────────────────────────────────────────────────────────────────

export function usePlayback(): PlaybackContextValue {
  const ctx = useContext(PlaybackContext);
  if (!ctx) {
    throw new Error('usePlayback must be used inside <PlaybackProvider>');
  }
  return ctx;
}
