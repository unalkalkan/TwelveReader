import React, { useState, useMemo } from 'react';
import {
  View,
  Text,
  ScrollView,
  Image,
  TouchableOpacity,
  StyleSheet,
  Alert,
  Platform,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useBooks, useVoices, useBookStatus, useDeleteBook } from '../../src/api/hooks';
import { usePlayback } from '../../src/store/playbackStore';
import type { BookMetadata } from '../../src/types/api';

const FILTERS = ['For you', 'Following', 'Recents'] as const;
const FILTER_ICONS: Record<string, keyof typeof MaterialIcons.glyphMap> = {
  'For you': 'auto-awesome',
  Following: 'person-outline',
  Recents: 'history',
};

const UPLOAD_ACTIONS = [
  { icon: 'text-fields' as const, label: 'Write\ntext' },
  { icon: 'file-upload' as const, label: 'Upload a\nfile' },
  { icon: 'document-scanner' as const, label: 'Scan\ntext' },
  { icon: 'link' as const, label: 'Paste a\nlink' },
];

export default function HomeScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const [activeFilter, setActiveFilter] = useState<string>('For you');
  const [continueMenuVisible, setContinueMenuVisible] = useState(false);
  const { data: books } = useBooks();
  const { data: voicesData } = useVoices();
  const { state: playbackState, pause, reset } = usePlayback();
  const deleteMutation = useDeleteBook();

  // Get the book that's currently in the player (if any)
  const { data: currentBookStatus } = useBookStatus(
    playbackState.currentBookId ?? undefined,
  );

  // Determine which book to show in "Continue Listening"
  const continueBook = useMemo(() => {
    if (!books?.length) return null;
    // If something is playing, find it
    if (playbackState.currentBookId) {
      const playing = books.find((b) => b.id === playbackState.currentBookId);
      if (playing) return playing;
    }
    // Otherwise show most recent
    return books[0];
  }, [books, playbackState.currentBookId]);

  const runContinueDelete = (bookId: string) => {
    if (playbackState.currentBookId === bookId) {
      pause();
      reset();
    }
    deleteMutation.mutate(bookId);
  };

  const handleContinueDelete = (bookId: string) => {
    setContinueMenuVisible(false);

    if (Platform.OS === 'web') {
      if (!window.confirm('Delete this book? This action cannot be undone.')) return;
    } else {
      Alert.alert(
        'Delete Book',
        'This action cannot be undone. Delete this book and all associated audio?',
        [
          { text: 'Cancel', style: 'cancel' },
          {
            text: 'Delete',
            style: 'destructive',
            onPress: () => {
              runContinueDelete(bookId);
            },
          },
        ],
      );
      return;
    }

    runContinueDelete(bookId);
  };

  // Compute real progress for continue card
  const totalSegments = currentBookStatus?.total_segments ?? continueBook?.total_segments ?? 0;
  const progressPercent =
    totalSegments > 0
      ? Math.round(
          ((playbackState.currentSegmentIndex + 1) / totalSegments) * 100,
        )
      : 0;
  const estimatedMinutesLeft =
    totalSegments > 0
      ? Math.round(
          ((totalSegments - playbackState.currentSegmentIndex) * 15) / 60,
        )
      : 0;

  // Filter books based on active filter
  const filteredBooks = useMemo(() => {
    if (!books) return [];
    switch (activeFilter) {
      case 'Recents':
        return [...books].sort(
          (a, b) =>
            new Date(b.uploaded_at).getTime() - new Date(a.uploaded_at).getTime(),
        );
      case 'Following':
        // Future feature — show empty for now
        return [];
      case 'For you':
      default:
        return books;
    }
  }, [books, activeFilter]);

  // Get real voices for trending section
  const trendingVoices = voicesData?.voices?.slice(0, 2) ?? [];

  const GRADIENT_COLORS = ['#0EA5E9', '#84CC16', '#EC4899', '#F59E0B'];

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Header ─── */}
      <View style={styles.header}>
        <Text style={[styles.greeting, { color: colors.text }]}>
          Listen in
        </Text>
        <View style={styles.headerActions}>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="search" size={20} color={colors.text} />
          </TouchableOpacity>
          <TouchableOpacity
            onPress={() => router.push('/modal')}
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="person" size={20} color={colors.text} />
          </TouchableOpacity>
        </View>
      </View>

      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* ─── Filter pills ─── */}
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.filterRow}
        >
          {FILTERS.map((f) => (
            <TouchableOpacity
              key={f}
              onPress={() => setActiveFilter(f)}
              style={[
                styles.pill,
                activeFilter === f
                  ? { backgroundColor: colors.text }
                  : { backgroundColor: colors.card },
              ]}
            >
              <MaterialIcons
                name={FILTER_ICONS[f]}
                size={14}
                color={activeFilter === f ? colors.background : colors.textMuted}
              />
              <Text
                style={[
                  styles.pillText,
                  {
                    color:
                      activeFilter === f
                        ? colors.background
                        : colors.textMuted,
                  },
                ]}
              >
                {f}
              </Text>
            </TouchableOpacity>
          ))}
        </ScrollView>

        {/* ─── Continue Listening ─── */}
        {continueBook && (
          <View style={styles.section}>
            <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>
              CONTINUE LISTENING
            </Text>
            <TouchableOpacity
              activeOpacity={0.8}
              onPress={() =>
                router.push(`/player?bookId=${continueBook.id}`)
              }
              style={[styles.continueCard, { backgroundColor: colors.card }]}
            >
              <View style={styles.continueThumb}>
                <Image
                  source={require('../../assets/images/icon.png')}
                  style={styles.continueImg}
                />
                <View style={styles.progressBar}>
                  <View
                    style={[
                      styles.progressFill,
                      {
                        width: `${Math.max(progressPercent, 1)}%`,
                        backgroundColor: colors.accent,
                      },
                    ]}
                  />
                </View>
              </View>
              <View style={styles.continueMeta}>
                <Text style={[styles.continueTitle, { color: colors.text }]}>
                  {continueBook.title || 'Untitled'}
                </Text>
                <Text
                  style={[styles.continueAuthor, { color: colors.textMuted }]}
                >
                  {continueBook.author || 'Unknown author'}
                </Text>
                <View style={styles.continueStats}>
                  <Text style={{ color: colors.accent, fontSize: 12, fontWeight: '500' }}>
                    {progressPercent}%
                  </Text>
                  {estimatedMinutesLeft > 0 && (
                    <Text style={{ color: colors.textMuted, fontSize: 12, marginLeft: 8 }}>
                      {estimatedMinutesLeft} mins left
                    </Text>
                  )}
                </View>
              </View>
              <TouchableOpacity
                style={[styles.moreBtn, { borderColor: colors.border }]}
                onPress={(event) => {
                  event.stopPropagation?.();
                  setContinueMenuVisible(true);
                }}
              >
                <MaterialIcons name="more-vert" size={16} color={colors.textMuted} />
              </TouchableOpacity>
            </TouchableOpacity>
          </View>
        )}

        {/* ─── Following filter: empty state ─── */}
        {activeFilter === 'Following' && (
          <View style={[styles.section, { alignItems: 'center', paddingVertical: 40 }]}>
            <MaterialIcons name="group" size={48} color={colors.textMuted} />
            <Text style={{ color: colors.textMuted, marginTop: 12, fontSize: 15 }}>
              Following feature coming soon
            </Text>
          </View>
        )}

        {/* ─── Upload & Listen ─── */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Upload & listen
          </Text>
          <View style={styles.uploadGrid}>
            {UPLOAD_ACTIONS.map((action) => (
              <TouchableOpacity
                key={action.icon}
                activeOpacity={0.7}
                onPress={() => router.push('/(tabs)/add')}
                style={[styles.uploadCard, { backgroundColor: colors.card }]}
              >
                <MaterialIcons
                  name={action.icon}
                  size={28}
                  color={colors.textMuted}
                />
                <Text
                  style={[
                    styles.uploadLabel,
                    { color: colors.text },
                  ]}
                >
                  {action.label}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>

        {/* ─── Recommended Collections ─── */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Recommended collections
          </Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={{ gap: 16 }}
          >
            {['New Voices Worth Discovering', 'Educational Excellence'].map(
              (name) => (
                <TouchableOpacity
                  key={name}
                  activeOpacity={0.8}
                  style={styles.collectionCard}
                >
                  <View
                    style={[
                      styles.collectionBg,
                      {
                        backgroundColor:
                          name.includes('New') ? '#1E3A5F' : '#5F4B1E',
                      },
                    ]}
                  >
                    <Text style={styles.collectionTitle}>{name}</Text>
                  </View>
                </TouchableOpacity>
              ),
            )}
          </ScrollView>
        </View>

        {/* ─── Trending Voices (from API) ─── */}
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <Text style={[styles.sectionTitle, { color: colors.text }]}>
              Trending voices
            </Text>
            <TouchableOpacity onPress={() => router.push('/(tabs)/voices')}>
              <Text style={{ color: colors.accent, fontSize: 14, fontWeight: '500' }}>
                See all
              </Text>
            </TouchableOpacity>
          </View>
          {trendingVoices.length > 0
            ? trendingVoices.map((voice, idx) => (
                <View key={voice.id} style={styles.voiceRow}>
                  <View
                    style={[
                      styles.voiceAvatar,
                      {
                        backgroundColor:
                          GRADIENT_COLORS[idx % GRADIENT_COLORS.length],
                      },
                    ]}
                  >
                    <MaterialIcons name="person" size={24} color="#FFF" />
                  </View>
                  <View style={{ flex: 1 }}>
                    <Text style={[styles.voiceName, { color: colors.text }]}>
                      {voice.name}
                    </Text>
                    <Text
                      style={[styles.voiceDesc, { color: colors.textMuted }]}
                      numberOfLines={1}
                    >
                      {voice.description || `${voice.gender ?? 'AI'} Voice · ${voice.provider}`}
                    </Text>
                  </View>
                  <TouchableOpacity>
                    <MaterialIcons
                      name="favorite-border"
                      size={20}
                      color={colors.textMuted}
                    />
                  </TouchableOpacity>
                </View>
              ))
            : /* Fallback placeholders */
              ['Viraj', 'True'].map((name) => (
                <View key={name} style={styles.voiceRow}>
                  <View
                    style={[
                      styles.voiceAvatar,
                      {
                        backgroundColor:
                          name === 'Viraj' ? '#0EA5E9' : '#84CC16',
                      },
                    ]}
                  >
                    <MaterialIcons name="person" size={24} color="#FFF" />
                  </View>
                  <View style={{ flex: 1 }}>
                    <Text style={[styles.voiceName, { color: colors.text }]}>
                      {name}
                    </Text>
                    <Text
                      style={[styles.voiceDesc, { color: colors.textMuted }]}
                      numberOfLines={1}
                    >
                      {name === 'Viraj'
                        ? 'Rich, Confident And Expressive · Narrative & Story'
                        : 'Crime & Horror Narrator · Narrative & Story'}
                    </Text>
                  </View>
                  <TouchableOpacity>
                    <MaterialIcons
                      name="favorite-border"
                      size={20}
                      color={colors.textMuted}
                    />
                  </TouchableOpacity>
                </View>
              ))}
        </View>

        {/* Bottom spacer for mini player */}
        <View style={{ height: 160 }} />
      </ScrollView>

      {/* ─── Continue Menu Dropdown ─── */}
      {continueMenuVisible && continueBook && (
        <TouchableOpacity
          style={styles.dropdownBackdrop}
          activeOpacity={1}
          onPress={() => setContinueMenuVisible(false)}
        >
          <View
            style={[
              styles.dropdownMenu,
              { backgroundColor: colors.surface, borderColor: colors.border },
            ]}
          >
            <TouchableOpacity
              style={styles.dropdownItem}
              onPress={() => handleContinueDelete(continueBook.id)}
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

const styles = StyleSheet.create({
  safe: { flex: 1 },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 24,
    paddingTop: 12,
    paddingBottom: 16,
  },
  greeting: { fontSize: 24, fontWeight: '700', letterSpacing: -0.5 },
  headerActions: { flexDirection: 'row', gap: 12 },
  iconBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scrollContent: { paddingHorizontal: 24, paddingBottom: 32 },
  // Filters
  filterRow: { gap: 8, paddingVertical: 8 },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
  },
  pillText: { fontSize: 13, fontWeight: '600' },
  // Sections
  section: { marginTop: 28 },
  sectionLabel: {
    fontSize: 12,
    fontWeight: '500',
    letterSpacing: 1,
    marginBottom: 12,
  },
  sectionTitle: { fontSize: 20, fontWeight: '700', marginBottom: 16 },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-end',
    marginBottom: 16,
  },
  // Continue listening
  continueCard: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
    borderRadius: 16,
    gap: 16,
  },
  continueThumb: {
    width: 96,
    height: 128,
    borderRadius: 8,
    overflow: 'hidden',
  },
  continueImg: { width: '100%', height: '100%', resizeMode: 'cover' },
  progressBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    height: 3,
    backgroundColor: 'rgba(0,0,0,0.3)',
  },
  progressFill: { height: '100%' },
  continueMeta: { flex: 1 },
  continueTitle: { fontSize: 18, fontWeight: '700', marginBottom: 4 },
  continueAuthor: { fontSize: 14, marginBottom: 8 },
  continueStats: { flexDirection: 'row', alignItems: 'center' },
  moreBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    borderWidth: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  // Upload grid
  uploadGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
  },
  uploadCard: {
    width: '47%',
    paddingVertical: 24,
    borderRadius: 16,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 12,
  },
  uploadLabel: {
    fontSize: 14,
    fontWeight: '500',
    textAlign: 'center',
    lineHeight: 18,
  },
  // Collections
  collectionCard: { width: 280 },
  collectionBg: {
    borderRadius: 24,
    aspectRatio: 4 / 3,
    justifyContent: 'flex-end',
    padding: 24,
    overflow: 'hidden',
  },
  collectionTitle: {
    color: '#FFF',
    fontSize: 18,
    fontWeight: '700',
    lineHeight: 22,
  },
  // Voices
  voiceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 16,
  },
  voiceAvatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
  },
  voiceName: { fontSize: 16, fontWeight: '700', marginBottom: 2 },
  voiceDesc: { fontSize: 12 },
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
    bottom: 200,
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
