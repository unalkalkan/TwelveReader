import { useEffect, useRef, useCallback } from 'react';
import { Audio, AVPlaybackStatus } from 'expo-av';
import { Platform } from 'react-native';

import { usePlayback } from '../store/playbackStore';
import { getAudioUrl } from '../api/client';

/**
 * useAudioPlayer – manages expo-av Sound instances driven by PlaybackContext.
 *
 * - Loads the current segment's audio from the backend.
 * - Plays / pauses based on store state.
 * - Reports elapsed / duration back to the store.
 * - Auto-advances to next segment when current finishes.
 * - Preloads the *next* segment for gapless playback.
 * - Respects playback speed.
 *
 * Call this once in the player screen. The hook is passive otherwise.
 */
export function useAudioPlayer(
  bookId: string | undefined,
  segments: { id: string }[] | undefined,
) {
  const {
    state,
    play,
    pause,
    seekToSegment,
    setElapsed,
    setDuration,
  } = usePlayback();

  const soundRef = useRef<Audio.Sound | null>(null);
  const preloadRef = useRef<Audio.Sound | null>(null);
  const loadedSegmentId = useRef<string | null>(null);
  const isUnmounted = useRef(false);

  const { currentSegmentIndex, isPlaying, playbackSpeed } = state;

  const currentSegment = segments?.[currentSegmentIndex];
  const nextSegment = segments?.[currentSegmentIndex + 1];
  const totalSegments = segments?.length ?? 0;

  // ── Audio mode ──────────────────────────────────────────────────────
  useEffect(() => {
    Audio.setAudioModeAsync({
      allowsRecordingIOS: false,
      playsInSilentModeIOS: true,
      staysActiveInBackground: true,
      shouldDuckAndroid: true,
    }).catch(() => {});

    return () => {
      isUnmounted.current = true;
    };
  }, []);

  // ── Load current segment audio ─────────────────────────────────────
  useEffect(() => {
    if (!bookId || !currentSegment) return;
    if (loadedSegmentId.current === currentSegment.id) return;

    let cancelled = false;

    (async () => {
      // Unload previous
      if (soundRef.current) {
        await soundRef.current.stopAsync().catch(() => {});
        await soundRef.current.unloadAsync().catch(() => {});
        soundRef.current = null;
      }

      // Check if preloaded sound matches
      if (
        preloadRef.current &&
        loadedSegmentId.current === `pre:${currentSegment.id}`
      ) {
        soundRef.current = preloadRef.current;
        preloadRef.current = null;
      } else {
        // Load fresh
        try {
          const { sound } = await Audio.Sound.createAsync(
            { uri: getAudioUrl(bookId, currentSegment.id) },
            { shouldPlay: false, rate: playbackSpeed, shouldCorrectPitch: true },
          );
          if (cancelled || isUnmounted.current) {
            await sound.unloadAsync();
            return;
          }
          soundRef.current = sound;
        } catch (err) {
          console.warn('Failed to load audio:', err);
          return;
        }
      }

      loadedSegmentId.current = currentSegment.id;

      // Attach status listener
      soundRef.current!.setOnPlaybackStatusUpdate(onPlaybackStatus);

      // Get duration
      const status = await soundRef.current!.getStatusAsync();
      if (status.isLoaded && status.durationMillis) {
        setDuration(status.durationMillis);
      }

      // If we should already be playing, start
      if (isPlaying && !cancelled && !isUnmounted.current) {
        await soundRef.current!.playAsync().catch(() => {});
      }
    })();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bookId, currentSegment?.id]);

  // ── Play/Pause sync ────────────────────────────────────────────────
  useEffect(() => {
    if (!soundRef.current) return;
    (async () => {
      try {
        const status = await soundRef.current!.getStatusAsync();
        if (!status.isLoaded) return;

        if (isPlaying && !status.isPlaying) {
          await soundRef.current!.playAsync();
        } else if (!isPlaying && status.isPlaying) {
          await soundRef.current!.pauseAsync();
        }
      } catch {
        // sound may have been unloaded
      }
    })();
  }, [isPlaying]);

  // ── Speed sync ─────────────────────────────────────────────────────
  useEffect(() => {
    if (!soundRef.current) return;
    soundRef.current
      .setRateAsync(playbackSpeed, true)
      .catch(() => {});
  }, [playbackSpeed]);

  // ── Preload next segment ───────────────────────────────────────────
  useEffect(() => {
    if (!bookId || !nextSegment) return;
    let cancelled = false;

    (async () => {
      // Unload previous preload
      if (preloadRef.current) {
        await preloadRef.current.unloadAsync().catch(() => {});
        preloadRef.current = null;
      }

      try {
        const { sound } = await Audio.Sound.createAsync(
          { uri: getAudioUrl(bookId, nextSegment.id) },
          { shouldPlay: false },
        );
        if (cancelled || isUnmounted.current) {
          await sound.unloadAsync();
          return;
        }
        preloadRef.current = sound;
        // Mark so we can check later
        loadedSegmentId.current = `pre:${nextSegment.id}`;
      } catch {
        // preload failure is non-fatal
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [bookId, nextSegment?.id]);

  // ── Cleanup on unmount ─────────────────────────────────────────────
  useEffect(() => {
    return () => {
      soundRef.current?.unloadAsync().catch(() => {});
      preloadRef.current?.unloadAsync().catch(() => {});
    };
  }, []);

  // ── Status callback ────────────────────────────────────────────────
  const onPlaybackStatus = useCallback(
    (status: AVPlaybackStatus) => {
      if (isUnmounted.current) return;
      if (!status.isLoaded) return;

      setElapsed(status.positionMillis ?? 0);
      if (status.durationMillis) {
        setDuration(status.durationMillis);
      }

      // Auto-advance when segment finishes
      if (status.didJustFinish) {
        if (currentSegmentIndex < totalSegments - 1) {
          seekToSegment(currentSegmentIndex + 1);
          // isPlaying stays true, next segment will auto-play in load effect
        } else {
          pause();
        }
      }
    },
    [currentSegmentIndex, totalSegments, seekToSegment, pause, setElapsed, setDuration],
  );

  // ── Seek within current audio ──────────────────────────────────────
  const seekBy = useCallback(
    async (deltaMs: number) => {
      if (!soundRef.current) return;
      try {
        const status = await soundRef.current.getStatusAsync();
        if (!status.isLoaded) return;

        const newPos = Math.max(
          0,
          Math.min(
            (status.positionMillis ?? 0) + deltaMs,
            status.durationMillis ?? 0,
          ),
        );

        // If seeking past the end, advance segment
        if (deltaMs > 0 && newPos >= (status.durationMillis ?? Infinity)) {
          if (currentSegmentIndex < totalSegments - 1) {
            seekToSegment(currentSegmentIndex + 1);
          }
          return;
        }

        // If seeking before the start, go to prev segment
        if (deltaMs < 0 && (status.positionMillis ?? 0) + deltaMs < 0) {
          if (currentSegmentIndex > 0) {
            seekToSegment(currentSegmentIndex - 1);
          }
          return;
        }

        await soundRef.current.setPositionAsync(newPos);
      } catch {
        // sound may have been unloaded
      }
    },
    [currentSegmentIndex, totalSegments, seekToSegment],
  );

  const rewind10 = useCallback(() => seekBy(-10_000), [seekBy]);
  const forward30 = useCallback(() => seekBy(30_000), [seekBy]);

  return { rewind10, forward30, seekBy };
}
