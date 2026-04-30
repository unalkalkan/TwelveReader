import React, { useState, useMemo } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  useWindowDimensions,
  TextInput,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useBooks, useVoices } from '../../src/api/hooks';
import type { BookMetadata } from '../../src/types/api';

const TABS = ['All', 'Complete', 'In Progress', 'By Language'] as const;

const STATUS_COLORS: Record<string, string> = {
  synthesized: '#22C55E',
  ready: '#3B82F6',
  error: '#EF4444',
  synthesis_error: '#EF4444',
  synthesizing: '#8B5CF6',
  uploaded: '#F59E0B',
  parsing: '#F59E0B',
  segmenting: '#F59E0B',
  voice_mapping: '#F59E0B',
};

const COLLECTION_COLORS = [
  ['#F97316', '#E11D48'],
  ['#6366F1', '#7C3AED'],
  ['#059669', '#14B8A6'],
  ['#0EA5E9', '#3B82F6'],
];

export default function ExploreScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const { width } = useWindowDimensions();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<string>('All');
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const { data: books } = useBooks();
  const { data: voices } = useVoices();

  const cardWidth = width * 0.85;

  // ── Filter books by tab ────────────────────────────────────────────
  const filteredBooks = useMemo(() => {
    if (!books) return [];

    let result = books;

    // Search filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (b) =>
          b.title.toLowerCase().includes(q) ||
          b.author.toLowerCase().includes(q),
      );
    }

    switch (activeTab) {
      case 'Complete':
        return result.filter((b) => b.status === 'synthesized');
      case 'In Progress':
        return result.filter((b) =>
          ['uploaded', 'parsing', 'segmenting', 'synthesizing', 'voice_mapping', 'ready'].includes(b.status),
        );
      default:
        return result;
    }
  }, [books, activeTab, searchQuery]);

  // ── Group by language ──────────────────────────────────────────────
  const booksByLanguage = useMemo(() => {
    if (!books) return {};
    const map: Record<string, BookMetadata[]> = {};
    books.forEach((b) => {
      const lang = b.language || 'Unknown';
      if (!map[lang]) map[lang] = [];
      map[lang].push(b);
    });
    return map;
  }, [books]);

  // ── "Featured" = most recent synthesized books ─────────────────────
  const featured = useMemo(() => {
    if (!books) return [];
    return books
      .filter((b) => b.status === 'synthesized')
      .sort((a, b) => new Date(b.uploaded_at).getTime() - new Date(a.uploaded_at).getTime())
      .slice(0, 4);
  }, [books]);

  // Voice count stat
  const voiceCount = voices?.count ?? voices?.voices?.length ?? 0;

  const renderBookCard = (book: BookMetadata, idx: number) => (
    <TouchableOpacity
      key={book.id}
      activeOpacity={0.8}
      onPress={() => router.push(`/player?bookId=${book.id}`)}
      style={styles.bookCard}
    >
      <View
        style={[
          styles.bookCover,
          { backgroundColor: colors.card },
        ]}
      >
        <MaterialIcons name="menu-book" size={32} color={colors.textMuted} />
      </View>
      <Text
        style={[styles.bookTitle, { color: colors.text }]}
        numberOfLines={1}
      >
        {book.title || 'Untitled'}
      </Text>
      <Text
        style={[styles.bookAuthor, { color: colors.textMuted }]}
        numberOfLines={1}
      >
        {book.author || 'Unknown'}
      </Text>
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Header ─── */}
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Explore</Text>
        <View style={styles.headerActions}>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
            onPress={() => {
              setSearchVisible(!searchVisible);
              if (searchVisible) setSearchQuery('');
            }}
          >
            <MaterialIcons
              name={searchVisible ? 'close' : 'search'}
              size={20}
              color={colors.text}
            />
          </TouchableOpacity>
        </View>
      </View>

      {/* ─── Search bar ─── */}
      {searchVisible && (
        <View style={[styles.searchBar, { backgroundColor: colors.card }]}>
          <MaterialIcons name="search" size={20} color={colors.textMuted} />
          <TextInput
            style={[styles.searchInput, { color: colors.text }]}
            value={searchQuery}
            onChangeText={setSearchQuery}
            placeholder="Search books..."
            placeholderTextColor={colors.textMuted}
            autoFocus
          />
          {searchQuery.length > 0 && (
            <TouchableOpacity onPress={() => setSearchQuery('')}>
              <MaterialIcons name="close" size={18} color={colors.textMuted} />
            </TouchableOpacity>
          )}
        </View>
      )}

      {/* ─── Tab pills ─── */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.tabRow}
      >
        {TABS.map((t) => (
          <TouchableOpacity
            key={t}
            onPress={() => setActiveTab(t)}
            style={[
              styles.pill,
              activeTab === t
                ? { backgroundColor: colors.text }
                : { backgroundColor: colors.card },
            ]}
          >
            <Text
              style={[
                styles.pillText,
                {
                  color:
                    activeTab === t ? colors.background : colors.textSecondary,
                },
              ]}
            >
              {t}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.scrollContent}
      >
        {/* ─── Stats row ─── */}
        <View style={[styles.statsRow, { paddingHorizontal: 20 }]}>
          <View style={[styles.statCard, { backgroundColor: colors.card }]}>
            <Text style={[styles.statNumber, { color: colors.accent }]}>
              {books?.length ?? 0}
            </Text>
            <Text style={[styles.statLabel, { color: colors.textMuted }]}>
              Books
            </Text>
          </View>
          <View style={[styles.statCard, { backgroundColor: colors.card }]}>
            <Text style={[styles.statNumber, { color: '#22C55E' }]}>
              {books?.filter((b) => b.status === 'synthesized').length ?? 0}
            </Text>
            <Text style={[styles.statLabel, { color: colors.textMuted }]}>
              Complete
            </Text>
          </View>
          <View style={[styles.statCard, { backgroundColor: colors.card }]}>
            <Text style={[styles.statNumber, { color: '#8B5CF6' }]}>
              {voiceCount}
            </Text>
            <Text style={[styles.statLabel, { color: colors.textMuted }]}>
              Voices
            </Text>
          </View>
        </View>

        {/* ─── Featured carousel (synthesized books) ─── */}
        {featured.length > 0 && activeTab !== 'By Language' && (
          <>
            <View style={styles.sectionHeader}>
              <Text style={[styles.sectionTitle, { color: colors.text }]}>
                ✨ Featured
              </Text>
            </View>
            <ScrollView
              horizontal
              pagingEnabled
              showsHorizontalScrollIndicator={false}
              snapToInterval={cardWidth + 16}
              decelerationRate="fast"
              contentContainerStyle={{ paddingHorizontal: 20, gap: 16 }}
            >
              {featured.map((book, idx) => (
                <TouchableOpacity
                  key={book.id}
                  activeOpacity={0.9}
                  onPress={() => router.push(`/player?bookId=${book.id}`)}
                  style={[
                    styles.featuredCard,
                    {
                      width: cardWidth,
                      backgroundColor:
                        COLLECTION_COLORS[idx % COLLECTION_COLORS.length][0],
                    },
                  ]}
                >
                  <View style={styles.featuredOverlay}>
                    <Text style={styles.featuredTitle} numberOfLines={2}>
                      {book.title || 'Untitled'}
                    </Text>
                    <Text style={styles.featuredSubtitle}>
                      {book.author || 'Unknown author'} · {book.total_segments}{' '}
                      segments
                    </Text>
                  </View>
                </TouchableOpacity>
              ))}
            </ScrollView>
          </>
        )}

        {/* ─── By Language tab ─── */}
        {activeTab === 'By Language' ? (
          Object.keys(booksByLanguage).length === 0 ? (
            <View style={styles.empty}>
              <MaterialIcons
                name="language"
                size={48}
                color={colors.textMuted}
              />
              <Text style={[styles.emptyText, { color: colors.textMuted }]}>
                No books yet
              </Text>
            </View>
          ) : (
            Object.entries(booksByLanguage).map(([lang, langBooks]) => (
              <View key={lang} style={styles.genreSection}>
                <View style={styles.genreHeader}>
                  <Text style={[styles.genreTitle, { color: colors.text }]}>
                    {lang}
                  </Text>
                  <Text style={{ color: colors.textMuted, fontSize: 14 }}>
                    {langBooks.length} book{langBooks.length === 1 ? '' : 's'}
                  </Text>
                </View>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={{ gap: 16 }}
                >
                  {langBooks.map((b, i) => renderBookCard(b, i))}
                </ScrollView>
              </View>
            ))
          )
        ) : (
          /* ─── All / Complete / In Progress ─── */
          <>
            {filteredBooks.length === 0 ? (
              <View style={styles.empty}>
                <MaterialIcons
                  name={
                    activeTab === 'Complete'
                      ? 'check-circle'
                      : activeTab === 'In Progress'
                        ? 'hourglass-top'
                        : 'explore'
                  }
                  size={48}
                  color={colors.textMuted}
                />
                <Text style={[styles.emptyText, { color: colors.textMuted }]}>
                  {searchQuery
                    ? 'No matches found'
                    : activeTab === 'Complete'
                      ? 'No completed audiobooks yet'
                      : activeTab === 'In Progress'
                        ? 'No books being processed'
                        : 'Upload a book to get started'}
                </Text>
              </View>
            ) : (
              <View style={styles.gridSection}>
                <View style={styles.sectionHeader}>
                  <Text style={[styles.sectionTitle, { color: colors.text }]}>
                    {activeTab === 'Complete'
                      ? 'Completed Audiobooks'
                      : activeTab === 'In Progress'
                        ? 'Being Processed'
                        : 'All Books'}
                  </Text>
                  <Text style={{ color: colors.textMuted, fontSize: 14 }}>
                    {filteredBooks.length}
                  </Text>
                </View>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={{ paddingHorizontal: 20, gap: 16 }}
                >
                  {filteredBooks.map((b, i) => renderBookCard(b, i))}
                </ScrollView>
              </View>
            )}
          </>
        )}

        <View style={{ height: 180 }} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingTop: 8,
    paddingBottom: 16,
  },
  title: { fontSize: 30, fontWeight: '700', letterSpacing: -0.5 },
  headerActions: { flexDirection: 'row', gap: 12 },
  iconBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    marginHorizontal: 20,
    marginBottom: 8,
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 12,
    gap: 8,
  },
  searchInput: {
    flex: 1,
    fontSize: 16,
    paddingVertical: 0,
  },
  tabRow: { paddingHorizontal: 20, gap: 8, paddingBottom: 16 },
  pill: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    borderRadius: 20,
  },
  pillText: { fontSize: 14, fontWeight: '500' },
  scrollContent: { paddingBottom: 32 },

  // Stats
  statsRow: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 24,
  },
  statCard: {
    flex: 1,
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  statNumber: { fontSize: 24, fontWeight: '700' },
  statLabel: { fontSize: 12, marginTop: 4 },

  // Section headers
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 20,
    marginTop: 8,
    marginBottom: 16,
  },
  sectionTitle: { fontSize: 20, fontWeight: '700' },

  // Featured
  featuredCard: {
    aspectRatio: 16 / 10,
    borderRadius: 16,
    overflow: 'hidden',
    justifyContent: 'flex-end',
  },
  featuredOverlay: {
    padding: 24,
    paddingTop: 48,
    backgroundColor: 'rgba(0,0,0,0.35)',
  },
  featuredTitle: {
    fontSize: 24,
    fontWeight: '700',
    color: '#FFF',
    lineHeight: 28,
  },
  featuredSubtitle: {
    fontSize: 14,
    color: 'rgba(255,255,255,0.8)',
    marginTop: 4,
  },

  // Genre/language sections
  genreSection: { marginTop: 32, paddingHorizontal: 20 },
  genreHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 16,
  },
  genreTitle: { fontSize: 20, fontWeight: '700' },

  // Grid section
  gridSection: { marginTop: 8 },

  // Book cards
  bookCard: { width: 128 },
  bookCover: {
    aspectRatio: 2 / 3,
    borderRadius: 8,
    overflow: 'hidden',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  bookTitle: { fontSize: 14, fontWeight: '600' },
  bookAuthor: { fontSize: 12, marginTop: 2 },

  // Empty
  empty: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 80,
    gap: 12,
  },
  emptyText: { fontSize: 15, textAlign: 'center' },
});
