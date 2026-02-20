import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Platform,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import {
  useBook,
  useBookSegments,
  useBookStatus,
  usePipelineStatus,
} from '../src/api/hooks';
import { getAudioUrl } from '../src/api/client';

export default function PlayerScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { bookId } = useLocalSearchParams<{ bookId: string }>();

  const { data: book } = useBook(bookId);
  const { data: segments } = useBookSegments(bookId);
  const { data: status } = useBookStatus(bookId);
  const { data: pipeline } = usePipelineStatus(bookId);

  const [isPlaying, setIsPlaying] = useState(false);
  const [currentSegmentIdx, setCurrentSegmentIdx] = useState(0);
  const [playbackSpeed, setPlaybackSpeed] = useState(1.0);

  const currentSegment = segments?.[currentSegmentIdx];
  const totalSegments = segments?.length ?? 0;
  const progress =
    totalSegments > 0 ? ((currentSegmentIdx + 1) / totalSegments) * 100 : 0;

  const togglePlayback = useCallback(() => {
    setIsPlaying((p) => !p);
    // Audio playback would be integrated with expo-av here
  }, []);

  const skipForward = useCallback(() => {
    if (currentSegmentIdx < totalSegments - 1) {
      setCurrentSegmentIdx((i) => i + 1);
    }
  }, [currentSegmentIdx, totalSegments]);

  const skipBackward = useCallback(() => {
    if (currentSegmentIdx > 0) {
      setCurrentSegmentIdx((i) => i - 1);
    }
  }, [currentSegmentIdx]);

  const cycleSpeed = useCallback(() => {
    setPlaybackSpeed((s) => {
      const speeds = [0.5, 0.75, 1.0, 1.2, 1.5, 2.0];
      const idx = speeds.indexOf(s);
      return speeds[(idx + 1) % speeds.length];
    });
  }, []);

  // Show processing overlay if book is still being processed
  const isProcessing =
    status?.status &&
    ['uploaded', 'parsing', 'segmenting', 'synthesizing', 'voice_mapping'].includes(
      status.status,
    );

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Top bar ─── */}
      <View style={styles.topBar}>
        <TouchableOpacity
          onPress={() => router.back()}
          style={[styles.topBtn, { backgroundColor: colors.surface }]}
        >
          <MaterialIcons name="expand-more" size={24} color={colors.textSecondary} />
        </TouchableOpacity>
        <View style={styles.topCenter}>
          <Text
            style={[styles.bookTitle, { color: colors.text }]}
            numberOfLines={1}
          >
            {book?.title ?? 'Loading...'}
          </Text>
          <Text style={[styles.bookAuthor, { color: colors.textMuted }]}>
            {book?.author?.toUpperCase() ?? ''}
          </Text>
        </View>
        <View style={styles.topActions}>
          <TouchableOpacity
            style={[styles.topBtn, { backgroundColor: colors.surface }]}
          >
            <MaterialIcons name="ios-share" size={18} color={colors.textSecondary} />
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.topBtn, { backgroundColor: colors.surface }]}
          >
            <MaterialIcons name="more-horiz" size={18} color={colors.textSecondary} />
          </TouchableOpacity>
        </View>
      </View>

      {/* ─── Reading area ─── */}
      <ScrollView
        style={styles.readingArea}
        contentContainerStyle={styles.readingContent}
        showsVerticalScrollIndicator={false}
      >
        {isProcessing && (
          <View style={[styles.processingBanner, { backgroundColor: colors.surface }]}>
            <MaterialIcons name="hourglass-top" size={20} color={colors.accent} />
            <Text style={{ color: colors.textSecondary, marginLeft: 8, flex: 1 }}>
              {status?.stage ?? 'Processing'}... {Math.round(status?.progress ?? 0)}%
            </Text>
          </View>
        )}

        {segments && segments.length > 0 ? (
          segments
            .slice(
              Math.max(0, currentSegmentIdx - 1),
              currentSegmentIdx + 3,
            )
            .map((seg, displayIdx) => {
              const actualIdx =
                Math.max(0, currentSegmentIdx - 1) + displayIdx;
              const isCurrent = actualIdx === currentSegmentIdx;
              const isPast = actualIdx < currentSegmentIdx;

              return (
                <TouchableOpacity
                  key={seg.id}
                  activeOpacity={0.8}
                  onPress={() => setCurrentSegmentIdx(actualIdx)}
                >
                  <Text
                    style={[
                      styles.paragraph,
                      isPast && { opacity: 0.4 },
                      isCurrent && {
                        color: colors.text,
                      },
                      !isCurrent &&
                        !isPast && {
                          color: colors.textMuted,
                        },
                    ]}
                  >
                    {isCurrent ? (
                      <Text
                        style={[
                          styles.highlightedText,
                          {
                            backgroundColor: 'rgba(59, 130, 246, 0.2)',
                            color: colors.text,
                          },
                        ]}
                      >
                        {seg.text}
                      </Text>
                    ) : (
                      seg.text
                    )}
                  </Text>
                </TouchableOpacity>
              );
            })
        ) : (
          <View style={styles.emptyReading}>
            <MaterialIcons
              name="auto-stories"
              size={48}
              color={colors.textMuted}
            />
            <Text style={{ color: colors.textMuted, marginTop: 12, fontSize: 16 }}>
              {isProcessing
                ? 'Waiting for segments...'
                : 'No segments available'}
            </Text>
          </View>
        )}
      </ScrollView>

      {/* ─── AI Fab ─── */}
      <TouchableOpacity
        style={[
          styles.aiFab,
          {
            backgroundColor: colors.surface,
            borderColor: colors.border,
          },
        ]}
      >
        <MaterialIcons name="auto-awesome" size={24} color={colors.accent} />
      </TouchableOpacity>

      {/* ─── Playback controls footer ─── */}
      <View
        style={[
          styles.footer,
          {
            backgroundColor: colors.playerBg,
            borderTopColor: colors.border,
          },
        ]}
      >
        {/* Progress bar */}
        <View style={styles.progressSection}>
          <View
            style={[styles.progressTrack, { backgroundColor: colors.card }]}
          >
            <View
              style={[
                styles.progressFill,
                {
                  width: `${progress}%`,
                  backgroundColor: colors.accent,
                },
              ]}
            />
          </View>
          <View style={styles.timeRow}>
            <Text style={[styles.timeText, { color: colors.textMuted }]}>
              {formatSegmentTime(currentSegmentIdx)}
            </Text>
            <Text style={[styles.timeText, { color: colors.textMuted }]}>
              {formatSegmentTime(totalSegments - currentSegmentIdx)}
            </Text>
          </View>
        </View>

        {/* Main controls */}
        <View style={styles.controls}>
          <TouchableOpacity>
            <MaterialIcons
              name="bookmark-border"
              size={24}
              color={colors.textMuted}
            />
          </TouchableOpacity>
          <TouchableOpacity onPress={skipBackward}>
            <MaterialIcons
              name="replay-10"
              size={28}
              color={colors.textMuted}
            />
          </TouchableOpacity>
          <TouchableOpacity
            onPress={togglePlayback}
            style={[
              styles.playButton,
              {
                backgroundColor: theme === 'dark' ? '#FFFFFF' : '#1E1E1E',
              },
            ]}
          >
            <MaterialIcons
              name={isPlaying ? 'pause' : 'play-arrow'}
              size={36}
              color={theme === 'dark' ? '#000000' : '#FFFFFF'}
            />
          </TouchableOpacity>
          <TouchableOpacity onPress={skipForward}>
            <MaterialIcons
              name="forward-30"
              size={28}
              color={colors.textMuted}
            />
          </TouchableOpacity>
          <TouchableOpacity onPress={cycleSpeed}>
            <Text
              style={[
                styles.speedLabel,
                { color: colors.textMuted },
              ]}
            >
              {playbackSpeed}x
            </Text>
          </TouchableOpacity>
        </View>

        {/* Bottom actions */}
        <View style={styles.bottomActions}>
          <TouchableOpacity style={styles.bottomAction}>
            <View
              style={[
                styles.voiceAvatarSmall,
                { backgroundColor: colors.card },
              ]}
            >
              <MaterialIcons name="person" size={16} color={colors.textMuted} />
            </View>
          </TouchableOpacity>
          <TouchableOpacity>
            <MaterialIcons
              name="format-list-bulleted"
              size={24}
              color={colors.textMuted}
            />
          </TouchableOpacity>
          <TouchableOpacity>
            <MaterialIcons
              name="volume-up"
              size={24}
              color={colors.textMuted}
            />
          </TouchableOpacity>
          <TouchableOpacity>
            <MaterialIcons
              name="picture-as-pdf"
              size={24}
              color={colors.textMuted}
            />
          </TouchableOpacity>
        </View>
      </View>
    </SafeAreaView>
  );
}

function formatSegmentTime(segments: number): string {
  // Rough estimate: ~15 seconds per segment
  const totalSeconds = segments * 15;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  // Top bar
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingVertical: 12,
  },
  topBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  topCenter: { alignItems: 'center', flex: 1 },
  bookTitle: {
    fontSize: 14,
    fontWeight: '700',
    letterSpacing: -0.3,
  },
  bookAuthor: {
    fontSize: 11,
    fontWeight: '500',
    letterSpacing: 2,
    marginTop: 2,
  },
  topActions: { flexDirection: 'row', gap: 8 },
  // Reading area
  readingArea: { flex: 1 },
  readingContent: {
    paddingHorizontal: 32,
    paddingTop: 16,
    paddingBottom: 80,
  },
  paragraph: {
    fontSize: 21,
    lineHeight: 32,
    fontFamily: Platform.OS === 'ios' ? 'Georgia' : 'serif',
    marginBottom: 24,
  },
  highlightedText: {
    borderRadius: 4,
    paddingHorizontal: 4,
  },
  emptyReading: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: 100,
  },
  processingBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    borderRadius: 12,
    marginBottom: 20,
  },
  // AI Fab
  aiFab: {
    position: 'absolute',
    bottom: 320,
    right: 24,
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 12,
    elevation: 8,
  },
  // Footer
  footer: {
    paddingTop: 16,
    paddingBottom: Platform.OS === 'ios' ? 0 : 16,
    paddingHorizontal: 24,
    borderTopWidth: StyleSheet.hairlineWidth,
  },
  progressSection: { marginBottom: 20 },
  progressTrack: {
    height: 4,
    borderRadius: 2,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    borderRadius: 2,
  },
  timeRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 8,
  },
  timeText: { fontSize: 11, fontWeight: '700', letterSpacing: -0.5 },
  // Controls
  controls: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 24,
    paddingHorizontal: 8,
  },
  playButton: {
    width: 64,
    height: 64,
    borderRadius: 32,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 6,
  },
  speedLabel: { fontSize: 14, fontWeight: '700' },
  // Bottom actions
  bottomActions: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    alignItems: 'center',
    paddingBottom: 8,
  },
  bottomAction: {},
  voiceAvatarSmall: {
    width: 32,
    height: 32,
    borderRadius: 16,
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
});
