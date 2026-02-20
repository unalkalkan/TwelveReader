import React from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  FlatList,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useBooks } from '../../src/api/hooks';
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
};

export default function LibraryScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { data: books, isLoading, refetch } = useBooks();

  const renderBook = ({ item }: { item: BookMetadata }) => (
    <TouchableOpacity
      activeOpacity={0.8}
      onPress={() => router.push(`/player?bookId=${item.id}`)}
      style={[styles.bookRow, { borderBottomColor: colors.border }]}
    >
      <View style={[styles.bookCover, { backgroundColor: colors.card }]}>
        <MaterialIcons name="menu-book" size={24} color={colors.textMuted} />
      </View>
      <View style={styles.bookInfo}>
        <Text style={[styles.bookTitle, { color: colors.text }]} numberOfLines={1}>
          {item.title}
        </Text>
        <Text style={[styles.bookAuthor, { color: colors.textSecondary }]} numberOfLines={1}>
          {item.author}
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
      </View>
      <MaterialIcons name="chevron-right" size={20} color={colors.textMuted} />
    </TouchableOpacity>
  );

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Library</Text>
        <TouchableOpacity
          style={[styles.iconBtn, { backgroundColor: colors.card }]}
        >
          <MaterialIcons name="search" size={20} color={colors.text} />
        </TouchableOpacity>
      </View>

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
          <Text style={[styles.emptySubtitle, { color: colors.textSecondary }]}>
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
      ) : (
        <FlatList
          data={books}
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
