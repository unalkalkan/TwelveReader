import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  Image,
  TouchableOpacity,
  StyleSheet,
  FlatList,
  useWindowDimensions,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useBooks } from '../../src/api/hooks';

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
  const { data: books } = useBooks();

  // Pick latest book (if any) for "Continue listening"
  const latestBook = books?.[0];

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
        <View style={styles.section}>
          <Text style={[styles.sectionLabel, { color: colors.textMuted }]}>
            CONTINUE LISTENING
          </Text>
          <TouchableOpacity
            activeOpacity={0.8}
            onPress={() =>
              latestBook &&
              router.push(`/player?bookId=${latestBook.id}`)
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
                    { width: '74%', backgroundColor: colors.accent },
                  ]}
                />
              </View>
            </View>
            <View style={styles.continueMeta}>
              <Text style={[styles.continueTitle, { color: colors.text }]}>
                {latestBook?.title ?? 'Perfume'}
              </Text>
              <Text
                style={[styles.continueAuthor, { color: colors.textMuted }]}
              >
                {latestBook?.author ?? 'Patrick Süskind'}
              </Text>
              <View style={styles.continueStats}>
                <Text style={{ color: colors.accent, fontSize: 12, fontWeight: '500' }}>
                  74%
                </Text>
                <Text style={{ color: colors.textMuted, fontSize: 12, marginLeft: 8 }}>
                  118 mins left
                </Text>
              </View>
            </View>
            <TouchableOpacity
              style={[styles.moreBtn, { borderColor: colors.border }]}
            >
              <MaterialIcons name="more-vert" size={16} color={colors.textMuted} />
            </TouchableOpacity>
          </TouchableOpacity>
        </View>

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

        {/* ─── Trending Voices ─── */}
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
          {['Viraj', 'True'].map((name) => (
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
                    ? 'Rich, Confident And Expressive • Narrative & Story'
                    : 'Crime & Horror Narrator • Narrative & Story'}
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
});
