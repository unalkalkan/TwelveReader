import React, { useState, useMemo } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  FlatList,
  TextInput,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useBooksWithFastPolling } from '../../src/api/hooks';
import type { BookMetadata } from '../../src/types/api';

const STATUS_LABELS: Record<string, string> = {
  uploaded: 'Processing...',
  parsing: 'Parsing...',
  segmenting: 'Segmenting...',
  voice_mapping: 'Mapping voices...',
  ready: 'Ready to play',
  synthesizing: 'Synthesizing...',
  synthesized: 'Complete',
  synthesis_error: 'Error',
  error: 'Error',
};

const STATUS_COLORS: Record<string, string> = {
  synthesized: '#22C55E',
  ready: '#3B82F6',
  error: '#EF4444',
  synthesis_error: '#EF4444',
  uploaded: '#F59E0B',
  parsing: '#F59E0B',
  segmenting: '#F59E0B',
  synthesizing: '#8B5CF6',
  voice_mapping: '#F59E0B',
};

export default function LibraryScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { data: books, isLoading, refetch } = useBooksWithFastPolling();
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Filter books by search query
  const filteredBooks = useMemo(() => {
    if (!books) return [];
    if (!searchQuery.trim()) return books;
    const q = searchQuery.toLowerCase();
    return books.filter(
      (b) =>
        b.title.toLowerCase().includes(q) ||
        b.author.toLowerCase().includes(q),
    );
  }, [books, searchQuery]);

  const renderBook = ({ item }: { item: BookMetadata }) => {
    const isActive = [
      'uploaded',
      'parsing',
      'segmenting',
      'synthesizing',
      'voice_mapping',
    ].includes(item.status);

    return (
      <TouchableOpacity
        activeOpacity={0.8}
        onPress={() => router.push(`/player?bookId=${item.id}`)}
        style={[styles.bookRow, { borderBottomColor: colors.border }]}
      >
        <View style={[styles.bookCover, { backgroundColor: colors.card }]}>
          <MaterialIcons name="menu-book" size={24} color={colors.textMuted} />
        </View>
        <View style={styles.bookInfo}>
          <Text
            style={[styles.bookTitle, { color: colors.text }]}
            numberOfLines={1}
          >
            {item.title || 'Untitled'}
          </Text>
          <Text
            style={[styles.bookAuthor, { color: colors.textSecondary }]}
            numberOfLines={1}
          >
            {item.author || 'Unknown author'}
          </Text>
          <View style={styles.bookMeta}>
            <View
              style={[
                styles.statusDot,
                {
                  backgroundColor:
                    STATUS_COLORS[item.status] ?? colors.textMuted,
                },
              ]}
            />
            <Text style={[styles.statusText, { color: colors.textMuted }]}>
              {STATUS_LABELS[item.status] ?? item.status}
            </Text>
            <Text style={{ color: colors.textMuted, fontSize: 12 }}>
              {' '}· {item.total_segments} segments
            </Text>
          </View>
          {/* Show progress bar for active processing */}
          {isActive && (
            <View style={[styles.bookProgressTrack, { backgroundColor: colors.border }]}>
              <View
                style={[
                  styles.bookProgressFill,
                  {
                    backgroundColor: STATUS_COLORS[item.status] ?? colors.accent,
                    width: `${getBookProgressPercent(item)}%`,
                  },
                ]}
              />
            </View>
          )}
        </View>
        <MaterialIcons name="chevron-right" size={20} color={colors.textMuted} />
      </TouchableOpacity>
    );
  };

  return (
    <SafeAreaView
      style={[styles.safe, { backgroundColor: colors.background }]}
    >
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Library</Text>
        <TouchableOpacity
          onPress={() => {
            setSearchVisible(!searchVisible);
            if (searchVisible) setSearchQuery('');
          }}
          style={[styles.iconBtn, { backgroundColor: colors.card }]}
        >
          <MaterialIcons
            name={searchVisible ? 'close' : 'search'}
            size={20}
            color={colors.text}
          />
        </TouchableOpacity>
      </View>

      {/* ─── Search bar ─── */}
      {searchVisible && (
        <View style={[styles.searchBar, { backgroundColor: colors.card }]}>
          <MaterialIcons name="search" size={20} color={colors.textMuted} />
          <TextInput
            style={[styles.searchInput, { color: colors.text }]}
            value={searchQuery}
            onChangeText={setSearchQuery}
            placeholder="Search by title or author..."
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

      {isLoading ? (
        <View style={styles.empty}>
          <Text style={{ color: colors.textMuted }}>Loading...</Text>
        </View>
      ) : !books?.length ? (
        <View style={styles.empty}>
          <MaterialIcons name="headset" size={64} color={colors.textMuted} />
          <Text style={[styles.emptyTitle, { color: colors.text }]}>
            Your library is empty
          </Text>
          <Text
            style={[styles.emptySubtitle, { color: colors.textSecondary }]}
          >
            Upload a book to get started
          </Text>
          <TouchableOpacity
            onPress={() => router.push('/(tabs)/add')}
            style={[styles.ctaBtn, { backgroundColor: colors.accent }]}
          >
            <MaterialIcons name="add" size={20} color="#FFF" />
            <Text style={styles.ctaBtnText}>Upload Book</Text>
          </TouchableOpacity>
        </View>
      ) : filteredBooks.length === 0 && searchQuery ? (
        <View style={styles.empty}>
          <MaterialIcons name="search-off" size={48} color={colors.textMuted} />
          <Text style={[styles.emptyTitle, { color: colors.text }]}>
            No matches
          </Text>
          <Text style={[styles.emptySubtitle, { color: colors.textSecondary }]}>
            Try a different search term
          </Text>
        </View>
      ) : (
        <FlatList
          data={filteredBooks}
          keyExtractor={(b) => b.id}
          renderItem={renderBook}
          contentContainerStyle={{ paddingBottom: 200, paddingHorizontal: 20 }}
          onRefresh={refetch}
          refreshing={isLoading}
        />
      )}
    </SafeAreaView>
  );
}

function getBookProgressPercent(book: BookMetadata): number {
  switch (book.status) {
    case 'uploaded':
      return 8;
    case 'parsing':
      return 15;
    case 'segmenting': {
      const total = book.total_paragraphs ?? 0;
      const done = book.segmented_paragraphs ?? 0;
      return total > 0 ? Math.max(15, Math.min(65, Math.round((done / total) * 65))) : 35;
    }
    case 'voice_mapping':
      return 68;
    case 'ready':
      return 75;
    case 'synthesizing': {
      const total = book.total_segments ?? 0;
      const done = book.synthesized_segments ?? 0;
      return total > 0 ? Math.max(75, Math.min(98, 75 + Math.round((done / total) * 23))) : 82;
    }
    case 'synthesized':
      return 100;
    default:
      return 10;
  }
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
  // Book row
  bookRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: 16,
  },
  bookCover: {
    width: 56,
    height: 72,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
  },
  bookInfo: { flex: 1 },
  bookTitle: { fontSize: 16, fontWeight: '600', marginBottom: 2 },
  bookAuthor: { fontSize: 14, marginBottom: 4 },
  bookMeta: { flexDirection: 'row', alignItems: 'center' },
  statusDot: { width: 8, height: 8, borderRadius: 4, marginRight: 6 },
  statusText: { fontSize: 12 },
  bookProgressTrack: {
    height: 3,
    borderRadius: 1.5,
    marginTop: 6,
    overflow: 'hidden',
  },
  bookProgressFill: {
    height: '100%',
    borderRadius: 1.5,
  },
  // Empty state
  empty: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 12,
    padding: 40,
  },
  emptyTitle: { fontSize: 20, fontWeight: '700', marginTop: 8 },
  emptySubtitle: { fontSize: 15, textAlign: 'center' },
  ctaBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    paddingHorizontal: 24,
    paddingVertical: 14,
    borderRadius: 28,
    marginTop: 16,
  },
  ctaBtnText: { color: '#FFF', fontSize: 16, fontWeight: '600' },
});
