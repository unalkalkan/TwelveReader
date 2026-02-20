import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  Image,
  TouchableOpacity,
  StyleSheet,
  useWindowDimensions,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';

const TABS = ['Genres', 'Trending', 'New Releases', 'Audio-First'] as const;

const GENRE_BOOKS: Record<string, { title: string; author: string }[]> = {
  'Sci-Fi & Fantasy': [
    { title: 'The Void Beyond', author: 'Elena Kasen' },
    { title: 'Stolen Stars', author: 'Julian Deker' },
    { title: 'Neon Bloom', author: 'Fiona Grace' },
    { title: 'Silent Engine', author: 'Blake Pierce' },
  ],
  'Business & Tech': [
    { title: 'Neural Wealth', author: 'Sarah Chen' },
    { title: 'The AI Edge', author: 'Marcus Vane' },
    { title: 'Flow State', author: 'David Wright' },
  ],
};

const FEATURED = [
  {
    title: 'Literary\nClassics',
    subtitle: 'Enduring masterpieces with timeless themes',
    gradient: ['#F97316', '#E11D48'],
  },
  {
    title: 'Modern\nSci-Fi',
    subtitle: 'Visions of tomorrow, heard today',
    gradient: ['#6366F1', '#7C3AED'],
  },
];

export default function ExploreScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const { width } = useWindowDimensions();
  const [activeTab, setActiveTab] = useState<string>('Genres');

  const cardWidth = width * 0.85;

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Header ─── */}
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Explore</Text>
        <View style={styles.headerActions}>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="tune" size={20} color={colors.text} />
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="search" size={20} color={colors.text} />
          </TouchableOpacity>
        </View>
      </View>

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
        {/* ─── Featured carousel ─── */}
        <ScrollView
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          snapToInterval={cardWidth + 16}
          decelerationRate="fast"
          contentContainerStyle={{ paddingHorizontal: 20, gap: 16 }}
        >
          {FEATURED.map((f) => (
            <TouchableOpacity
              key={f.title}
              activeOpacity={0.9}
              style={[
                styles.featuredCard,
                {
                  width: cardWidth,
                  backgroundColor: f.gradient[0],
                },
              ]}
            >
              <View style={styles.featuredOverlay}>
                <Text style={styles.featuredTitle}>{f.title}</Text>
                <Text style={styles.featuredSubtitle}>{f.subtitle}</Text>
              </View>
            </TouchableOpacity>
          ))}
        </ScrollView>

        {/* ─── Genre sections ─── */}
        {Object.entries(GENRE_BOOKS).map(([genre, genreBooks]) => (
          <View key={genre} style={styles.genreSection}>
            <TouchableOpacity style={styles.genreHeader}>
              <Text style={[styles.genreTitle, { color: colors.text }]}>
                {genre}
              </Text>
              <MaterialIcons
                name="chevron-right"
                size={20}
                color={colors.textMuted}
              />
            </TouchableOpacity>

            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: 16 }}
            >
              {genreBooks.map((book) => (
                <TouchableOpacity
                  key={book.title}
                  activeOpacity={0.8}
                  style={styles.bookCard}
                >
                  <View
                    style={[
                      styles.bookCover,
                      { backgroundColor: colors.card },
                    ]}
                  >
                    <MaterialIcons
                      name="menu-book"
                      size={32}
                      color={colors.textMuted}
                    />
                  </View>
                  <Text
                    style={[styles.bookTitle, { color: colors.text }]}
                    numberOfLines={1}
                  >
                    {book.title}
                  </Text>
                  <Text
                    style={[styles.bookAuthor, { color: colors.textMuted }]}
                    numberOfLines={1}
                  >
                    {book.author}
                  </Text>
                </TouchableOpacity>
              ))}
            </ScrollView>
          </View>
        ))}

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
  tabRow: { paddingHorizontal: 20, gap: 8, paddingBottom: 16 },
  pill: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    borderRadius: 20,
  },
  pillText: { fontSize: 14, fontWeight: '500' },
  scrollContent: { paddingBottom: 32 },
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
    // Simplified gradient effect via bg color
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
  // Genre sections
  genreSection: { marginTop: 40, paddingHorizontal: 20 },
  genreHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 16,
  },
  genreTitle: { fontSize: 20, fontWeight: '700' },
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
});
