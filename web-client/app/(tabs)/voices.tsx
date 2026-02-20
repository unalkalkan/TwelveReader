import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useVoices } from '../../src/api/hooks';
import type { Voice } from '../../src/types/api';

const VOICE_TABS = ['Recents', 'Favorites', 'Explore'] as const;
const VOICE_TAB_ICONS: Record<string, keyof typeof MaterialIcons.glyphMap> = {
  Recents: 'schedule',
  Favorites: 'favorite-border',
  Explore: 'explore',
};

const GRADIENT_COLORS = [
  ['#06B6D4', '#2563EB'], // cyan → blue
  ['#84CC16', '#16A34A'], // lime → green
  ['#EC4899', '#9333EA'], // pink → purple
  ['#F59E0B', '#EA580C'], // amber → orange
  ['#8B5CF6', '#6D28D9'], // violet
  ['#EF4444', '#DC2626'], // red
];

const COLLECTIONS = [
  { name: 'Best for Audiobook Narrators', color: '#065F46' },
  { name: 'Top Sci-Fi Voices', color: '#312E81' },
];

export default function VoicesScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const [activeTab, setActiveTab] = useState<string>('Explore');
  const { data: voicesData } = useVoices();

  const voices = voicesData?.voices ?? [];

  // Split into trending (first 2) and language-recommended (rest)
  const trendingVoices = voices.slice(0, 2);
  const languageVoices = voices.slice(2, 4);

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Header ─── */}
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Voices</Text>
        <View style={styles.headerActions}>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="search" size={20} color={colors.text} />
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="tune" size={20} color={colors.text} />
          </TouchableOpacity>
        </View>
      </View>

      {/* ─── Filter tabs ─── */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.tabRow}
      >
        {VOICE_TABS.map((t) => (
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
            <MaterialIcons
              name={VOICE_TAB_ICONS[t]}
              size={14}
              color={activeTab === t ? colors.background : colors.textMuted}
            />
            <Text
              style={[
                styles.pillText,
                {
                  color: activeTab === t ? colors.background : colors.textMuted,
                  fontWeight: activeTab === t ? '700' : '500',
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
        {/* ─── Trending Voices ─── */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Trending voices
          </Text>
          {(trendingVoices.length > 0
            ? trendingVoices
            : PLACEHOLDER_VOICES.slice(0, 2)
          ).map((voice, idx) => (
            <VoiceRow
              key={voice.id ?? voice.name}
              voice={voice}
              gradientIdx={idx}
              colors={colors}
            />
          ))}
        </View>

        {/* ─── Voice Collections ─── */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Voice Collections
          </Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={{ gap: 16 }}
          >
            {COLLECTIONS.map((c) => (
              <TouchableOpacity
                key={c.name}
                activeOpacity={0.8}
                style={[
                  styles.collectionCard,
                  { backgroundColor: c.color },
                ]}
              >
                <View style={styles.collectionInner}>
                  <Text style={styles.collectionLabel}>{c.name}</Text>
                </View>
              </TouchableOpacity>
            ))}
          </ScrollView>
        </View>

        {/* ─── Best for your language ─── */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Best for your language 🇺🇸
          </Text>
          {(languageVoices.length > 0
            ? languageVoices
            : PLACEHOLDER_VOICES.slice(2, 4)
          ).map((voice, idx) => (
            <VoiceRow
              key={voice.id ?? voice.name}
              voice={voice}
              gradientIdx={idx + 2}
              colors={colors}
              favorited={idx === 1}
            />
          ))}
        </View>

        <View style={{ height: 200 }} />
      </ScrollView>
    </SafeAreaView>
  );
}

// ─── Voice row component ────────────────────────────────────────────────

function VoiceRow({
  voice,
  gradientIdx,
  colors,
  favorited = false,
}: {
  voice: Voice | { id?: string; name: string; description?: string };
  gradientIdx: number;
  colors: typeof Colors.dark;
  favorited?: boolean;
}) {
  const gc = GRADIENT_COLORS[gradientIdx % GRADIENT_COLORS.length];
  const [fav, setFav] = useState(favorited);

  return (
    <View style={styles.voiceRow}>
      <View style={[styles.voiceAvatar, { backgroundColor: gc[0] }]}>
        <MaterialIcons name="person" size={24} color="#FFF" />
      </View>
      <View style={{ flex: 1 }}>
        <Text style={[styles.voiceName, { color: colors.text }]}>
          {voice.name}
        </Text>
        <Text
          style={[styles.voiceDesc, { color: colors.textMuted }]}
          numberOfLines={2}
        >
          {'description' in voice && voice.description
            ? voice.description
            : 'AI Voice'}
        </Text>
      </View>
      <TouchableOpacity onPress={() => setFav(!fav)}>
        <MaterialIcons
          name={fav ? 'favorite' : 'favorite-border'}
          size={22}
          color={fav ? '#EF4444' : colors.textMuted}
        />
      </TouchableOpacity>
    </View>
  );
}

// Placeholders when API isn't connected yet
const PLACEHOLDER_VOICES = [
  { name: 'Viraj', description: 'Rich, Confident And Expressive · Narrative & Story' },
  { name: 'True', description: 'Crime & Horror Narrator · Narrative & Story' },
  { name: 'Brian', description: 'Deep, Resonant And Comforting · Social Media' },
  { name: 'Sarah', description: 'Velvety Actress · Informative & Educational' },
];

// ─── Styles ─────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  safe: { flex: 1 },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 24,
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
  tabRow: { paddingHorizontal: 24, gap: 8, paddingBottom: 8 },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
  },
  pillText: { fontSize: 14 },
  scrollContent: { paddingHorizontal: 24 },
  section: { marginTop: 28 },
  sectionTitle: { fontSize: 20, fontWeight: '700', marginBottom: 16 },
  // Voice row
  voiceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
    marginBottom: 16,
  },
  voiceAvatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
  },
  voiceName: { fontSize: 16, fontWeight: '600', marginBottom: 2 },
  voiceDesc: { fontSize: 12, lineHeight: 16 },
  // Collections
  collectionCard: {
    width: 288,
    height: 160,
    borderRadius: 16,
    overflow: 'hidden',
  },
  collectionInner: {
    flex: 1,
    justifyContent: 'center',
    padding: 20,
  },
  collectionLabel: {
    color: '#FFF',
    fontSize: 18,
    fontWeight: '700',
    lineHeight: 22,
  },
});
