import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Platform,
  Alert,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import {
  useBook,
  useBookStatus,
  usePipelineStatus,
  useBookStreamSegments,
  useDeleteBook,
} from '../src/api/hooks';
import { usePlayback } from '../src/store/playbackStore';
import { useAudioPlayer } from '../src/hooks/useAudioPlayer';
import { ChapterList } from '../src/components/ChapterList';
import { VoiceMappingModal } from '../src/components/VoiceMappingModal';
import { getDownloadUrl } from '../src/api/client';

export default function PlayerScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { bookId } = useLocalSearchParams<{ bookId: string }>();

  const { data: book } = useBook(bookId);
  const { data: streamSegments } = useBookStreamSegments(bookId);
  const { data: status } = useBookStatus(bookId);
  const { data: pipeline } = usePipelineStatus(bookId);
  const deleteMutation = useDeleteBook();

  const {
    state,
    setBook,
    seekToSegment,
    togglePlayback,
    play,
    pause,
    cycleSpeed,
    reset,
  } = usePlayback();

  const [chapterListVisible, setChapterListVisible] = useState(false);
  const [voiceMappingVisible, setVoiceMappingVisible] = useState(false);
  const [moreMenuVisible, setMoreMenuVisible] = useState(false);

  // Use stream segments (includes audio_url + timestamps)
  const segments = streamSegments ?? [];

  // Wire audio player
  const { rewind10, forward30 } = useAudioPlayer(bookId, segments);

  // Set book in playback store on mount (if different)
  useEffect(() => {
    if (bookId && state.currentBookId !== bookId) {
      setBook(bookId);
    }
  }, [bookId, state.currentBookId, setBook]);

  const { currentSegmentIndex, isPlaying, playbackSpeed, elapsedMs, durationMs } =
    state;

  const currentSegment = segments?.[currentSegmentIndex];
  const totalSegments = segments?.length ?? 0;

  // Progress based on segments + current audio position
  const withinSegmentProgress =
    durationMs > 0 ? elapsedMs / durationMs : 0;
  const overallProgress =
    totalSegments > 0
      ? ((currentSegmentIndex + withinSegmentProgress) / totalSegments) * 100
      : 0;

  // Show processing overlay if book is still being processed
  const isProcessing =
    status?.status &&
    ['uploaded', 'parsing', 'segmenting', 'synthesizing', 'voice_mapping'].includes(
      status.status,
    );
  const isWaitingForMapping = status?.status === 'voice_mapping';
  const hasBookError = status?.status === 'error' || status?.status === 'synthesis_error';

  const handleDelete = useCallback(() => {
    if (!bookId) return;

    if (Platform.OS === 'web') {
      if (!window.confirm('Delete this book? This action cannot be undone.')) return;
      pause();
      reset();
      deleteMutation.mutate(bookId, {
        onSuccess: () => {
          router.back();
        },
      });
      return;
    }

    Alert.alert(
      'Delete Book',
      'This action cannot be undone. Delete this book and all associated audio?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: () => {
            pause();
            reset();
            deleteMutation.mutate(bookId, {
              onSuccess: () => {
                router.back();
              },
            });
          },
        },
      ],
    );
  }, [bookId, pause, reset, deleteMutation, router]);

  const handleDownload = useCallback(() => {
    if (!bookId) return;
    if (Platform.OS === 'web' && typeof window !== 'undefined') {
      window.open(getDownloadUrl(bookId), '_blank');
      return;
    }
    Alert.alert('Download ready', getDownloadUrl(bookId));
  }, [bookId]);

  // ── Word-level highlighting ──────────────────────────────────────────
  const renderSegmentText = useCallback(
    (seg: (typeof segments)[0], isCurrent: boolean) => {
      if (!isCurrent || !seg.timestamps?.items?.length) {
        return (
          <Text
            style={[
              styles.paragraph,
              isCurrent
                ? { color: colors.text }
                : { color: colors.textMuted, opacity: 0.4 },
            ]}
          >
            {seg.text}
          </Text>
        );
      }

      // Word-level highlight: compare elapsed audio time with word timestamps
      const elapsedSec = elapsedMs / 1000;

      return (
        <Text style={[styles.paragraph, { color: colors.text }]}>
          {seg.timestamps.items.map((item, i) => {
            const isActive =
              elapsedSec >= item.start && elapsedSec < item.end;
            const isPast = elapsedSec >= item.end;

            return (
              <Text
                key={i}
                style={[
                  isActive && {
                    backgroundColor: 'rgba(59, 130, 246, 0.3)',
                    borderRadius: 2,
                  },
                  isPast && !isActive && { opacity: 0.7 },
                ]}
              >
                {item.word}
                {i < seg.timestamps!.items.length - 1 ? ' ' : ''}
              </Text>
            );
          })}
        </Text>
      );
    },
    [elapsedMs, colors],
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
            onPress={() => setMoreMenuVisible((v) => !v)}
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
            <MaterialIcons
              name={isWaitingForMapping ? 'record-voice-over' : 'hourglass-top'}
              size={20}
              color={isWaitingForMapping ? '#F59E0B' : colors.accent}
            />
            <Text style={{ color: colors.textSecondary, marginLeft: 8, flex: 1 }}>
              {isWaitingForMapping
                ? 'Voice mapping required before synthesis can continue.'
                : `${status?.stage ?? 'Processing'}... ${Math.round(status?.progress ?? 0)}%`}
            </Text>
            {isWaitingForMapping && (
              <TouchableOpacity
                onPress={() => setVoiceMappingVisible(true)}
                style={[styles.bannerButton, { backgroundColor: colors.accent }]}
              >
                <Text style={styles.bannerButtonText}>Map</Text>
              </TouchableOpacity>
            )}
          </View>
        )}

        {hasBookError && (
          <View style={[styles.processingBanner, { backgroundColor: colors.surface }]}>
            <MaterialIcons name="error-outline" size={20} color="#EF4444" />
            <Text style={{ color: colors.textSecondary, marginLeft: 8, flex: 1 }}>
              Book processing failed. Check server logs or retry with another file.
            </Text>
          </View>
        )}

        {segments && segments.length > 0 ? (
          segments
            .slice(
              Math.max(0, currentSegmentIndex - 1),
              currentSegmentIndex + 3,
            )
            .map((seg, displayIdx) => {
              const actualIdx =
                Math.max(0, currentSegmentIndex - 1) + displayIdx;
              const isCurrent = actualIdx === currentSegmentIndex;

              return (
                <TouchableOpacity
                  key={seg.id}
                  activeOpacity={0.8}
                  onPress={() => seekToSegment(actualIdx)}
                >
                  {/* Show persona label if available */}
                  {isCurrent && seg.person && (
                    <Text style={[styles.personLabel, { color: colors.accent }]}>
                      {seg.person}
                    </Text>
                  )}
                  {renderSegmentText(seg, isCurrent)}
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
                  width: `${Math.min(overallProgress, 100)}%`,
                  backgroundColor: colors.accent,
                },
              ]}
            />
          </View>
          <View style={styles.timeRow}>
            <Text style={[styles.timeText, { color: colors.textMuted }]}>
              {formatTime(elapsedMs)}
            </Text>
            <Text style={[styles.timeText, { color: colors.textMuted }]}>
              -{formatTime(Math.max(0, durationMs - elapsedMs))}
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
          <TouchableOpacity onPress={rewind10}>
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
          <TouchableOpacity onPress={forward30}>
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
          <TouchableOpacity
            onPress={() => setVoiceMappingVisible(true)}
            disabled={!bookId}
            style={styles.bottomAction}
          >
            <View
              style={[
                styles.voiceAvatarSmall,
                { backgroundColor: isWaitingForMapping ? colors.accent : colors.card },
              ]}
            >
              <MaterialIcons
                name="record-voice-over"
                size={16}
                color={isWaitingForMapping ? '#FFFFFF' : colors.textMuted}
              />
            </View>
          </TouchableOpacity>
          <TouchableOpacity onPress={() => setChapterListVisible(true)}>
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
          <TouchableOpacity onPress={handleDownload} disabled={!bookId}>
            <MaterialIcons
              name="archive"
              size={24}
              color={bookId ? colors.textMuted : colors.border}
            />
          </TouchableOpacity>
        </View>
      </View>

      {/* ─── Chapter List ─── */}
      <ChapterList
        visible={chapterListVisible}
        onClose={() => setChapterListVisible(false)}
        segments={segments as any}
        currentSegmentIndex={currentSegmentIndex}
        onSelectSegment={(idx) => {
          seekToSegment(idx);
          setChapterListVisible(false);
        }}
      />
      <VoiceMappingModal
        bookId={bookId}
        visible={voiceMappingVisible}
        initialMapping={isWaitingForMapping}
        onClose={() => setVoiceMappingVisible(false)}
      />

      {/* ─── More Menu Dropdown ─── */}
      {moreMenuVisible && (
        <TouchableOpacity
          style={styles.dropdownBackdrop}
          activeOpacity={1}
          onPress={() => setMoreMenuVisible(false)}
        >
          <View
            style={[
              styles.dropdownMenu,
              { backgroundColor: colors.surface, borderColor: colors.border },
            ]}
          >
            <TouchableOpacity
              style={styles.dropdownItem}
              onPress={() => {
                setMoreMenuVisible(false);
                handleDelete();
              }}
            >
              <MaterialIcons name="delete-outline" size={18} color="#EF4444" />
              <Text style={[styles.dropdownItemText, { color: '#EF4444' }]}>
                Delete Book
              </Text>
            </TouchableOpacity>
          </View>
        </TouchableOpacity>
      )}
    </SafeAreaView>
  );
}

function formatTime(ms: number): string {
  if (!ms || ms < 0) return '00:00';
  const totalSeconds = Math.floor(ms / 1000);
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
  personLabel: {
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 1,
    textTransform: 'uppercase',
    marginBottom: 4,
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
  bannerButton: {
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 8,
  },
  bannerButtonText: {
    color: '#FFFFFF',
    fontSize: 12,
    fontWeight: '800',
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
  dropdownBackdrop: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    zIndex: 100,
  },
  dropdownMenu: {
    position: 'absolute',
    top: 60,
    right: 20,
    minWidth: 180,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.15,
    shadowRadius: 12,
    elevation: 8,
    overflow: 'hidden',
  },
  dropdownItem: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 10,
  },
  dropdownItemText: {
    fontSize: 15,
    fontWeight: '500',
  },
});
