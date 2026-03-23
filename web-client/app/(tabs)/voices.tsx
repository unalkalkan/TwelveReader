import React, { useState, useMemo, useEffect, useRef } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  TextInput,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useFocusEffect } from '@react-navigation/native';
import { Audio, type AVPlaybackStatus } from 'expo-av';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useVoices } from '../../src/api/hooks';
import { useFavorites } from '../../src/store/favoritesStore';
import { previewVoice } from '../../src/api/client';
import type { Voice } from '../../src/types/api';

const VOICE_TABS = ['Explore', 'Favorites', 'Recents'] as const;
const VOICE_TAB_ICONS: Record<string, keyof typeof MaterialIcons.glyphMap> = {
  Explore: 'explore',
  Favorites: 'favorite-border',
  Recents: 'schedule',
};

const GRADIENT_COLORS = [
  '#06B6D4', '#84CC16', '#EC4899', '#F59E0B',
  '#8B5CF6', '#EF4444', '#10B981', '#F97316',
];

const COLLECTIONS = [
  { name: 'Best for Audiobook Narrators', color: '#065F46' },
  { name: 'Top Sci-Fi Voices', color: '#312E81' },
];

const PREVIEW_TEXT = `In my life, why do I give valuable time
To people who don't care if I live or die?`;

export default function VoicesScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const [activeTab, setActiveTab] = useState<string>('Explore');
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [loadingVoiceId, setLoadingVoiceId] = useState<string | null>(null);
  const [playingVoiceId, setPlayingVoiceId] = useState<string | null>(null);
  const previewSoundRef = useRef<Audio.Sound | null>(null);
  const { data: voicesData, isLoading, refetch } = useVoices('qwen3-tts');
  const { favoriteIds, isFavorite, toggleFavorite, recentIds, addRecent } = useFavorites();

  useFocusEffect(
    React.useCallback(() => {
      refetch();
    }, [refetch]),
  );

  const allVoices = voicesData?.voices ?? [];

  // Filter voices based on active tab + search
  const filteredVoices = useMemo(() => {
    let list: Voice[] = [];

    switch (activeTab) {
      case 'Favorites':
        list = allVoices.filter((v) => favoriteIds.has(v.id));
        break;
      case 'Recents':
        // Show voices in recent order
        list = recentIds
          .map((id) => allVoices.find((v) => v.id === id))
          .filter(Boolean) as Voice[];
        break;
      case 'Explore':
      default:
        list = allVoices;
        break;
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(
        (v) =>
          v.name.toLowerCase().includes(q) ||
          v.description?.toLowerCase().includes(q) ||
          v.provider.toLowerCase().includes(q),
      );
    }

    return list;
  }, [allVoices, activeTab, favoriteIds, recentIds, searchQuery]);

  // Split voices for display sections (only in Explore tab)
  const trendingVoices = activeTab === 'Explore' ? filteredVoices.slice(0, 4) : [];
  const languageVoices = activeTab === 'Explore' ? filteredVoices.slice(4, 8) : [];

  useEffect(() => {
    return () => {
      const sound = previewSoundRef.current;
      if (sound) {
        sound.unloadAsync().catch(() => {});
        previewSoundRef.current = null;
      }
    };
  }, []);

  const handlePreviewVoice = async (voice: Voice) => {
    if (loadingVoiceId) return;

    if (playingVoiceId === voice.id && previewSoundRef.current) {
      await previewSoundRef.current.stopAsync().catch(() => {});
      await previewSoundRef.current.unloadAsync().catch(() => {});
      previewSoundRef.current = null;
      setPlayingVoiceId(null);
      return;
    }

    setLoadingVoiceId(voice.id);

    try {
      if (previewSoundRef.current) {
        await previewSoundRef.current.stopAsync().catch(() => {});
        await previewSoundRef.current.unloadAsync().catch(() => {});
        previewSoundRef.current = null;
      }

      const response = await previewVoice({
        provider: voice.provider,
        voice_id: voice.id,
        text: PREVIEW_TEXT,
        language: voice.languages?.[0],
        voice_description: voice.description,
      });

      const audioUri = `data:${response.mime_type};base64,${response.audio_base64}`;

      const { sound } = await Audio.Sound.createAsync(
        { uri: audioUri },
        { shouldPlay: true },
      );

      previewSoundRef.current = sound;
      setPlayingVoiceId(voice.id);
      addRecent(voice.id);

      sound.setOnPlaybackStatusUpdate((status: AVPlaybackStatus) => {
        if (!status.isLoaded) return;
        if (status.didJustFinish) {
          sound.unloadAsync().catch(() => {});
          if (previewSoundRef.current === sound) {
            previewSoundRef.current = null;
          }
          setPlayingVoiceId((current) => (current === voice.id ? null : current));
        }
      });
    } catch (error: any) {
      Alert.alert('Preview failed', error?.message ?? 'Could not generate voice preview.');
      setPlayingVoiceId((current) => (current === voice.id ? null : current));
    } finally {
      setLoadingVoiceId((current) => (current === voice.id ? null : current));
    }
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* ─── Header ─── */}
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.text }]}>Voices</Text>
        <View style={styles.headerActions}>
          <TouchableOpacity
            onPress={() => setSearchVisible(!searchVisible)}
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons
              name={searchVisible ? 'close' : 'search'}
              size={20}
              color={colors.text}
            />
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.iconBtn, { backgroundColor: colors.card }]}
          >
            <MaterialIcons name="tune" size={20} color={colors.text} />
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
            placeholder="Search voices..."
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
              {t === 'Favorites' && favoriteIds.size > 0
                ? ` (${favoriteIds.size})`
                : ''}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.scrollContent}
      >
        {/* ─── Explore tab: sections ─── */}
        {activeTab === 'Explore' && !searchQuery && (
          <>
            {/* Trending Voices */}
            <View style={styles.section}>
              <Text style={[styles.sectionTitle, { color: colors.text }]}>
                Trending voices
              </Text>
              {trendingVoices.length > 0 ? (
                trendingVoices.map((voice, idx) => (
                  <VoiceRow
                    key={voice.id}
                    voice={voice}
                    gradientIdx={idx}
                    colors={colors}
                    isFav={isFavorite(voice.id)}
                    onToggleFav={() => toggleFavorite(voice.id)}
                    onPreview={() => handlePreviewVoice(voice)}
                    isPreviewLoading={loadingVoiceId === voice.id}
                    isPreviewPlaying={playingVoiceId === voice.id}
                  />
                ))
              ) : isLoading ? (
                <Text style={{ color: colors.textMuted }}>Loading...</Text>
              ) : (
                <Text style={{ color: colors.textMuted }}>No voices available</Text>
              )}
            </View>

            {/* Voice Collections */}
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

            {/* Best for your language */}
            {languageVoices.length > 0 && (
              <View style={styles.section}>
                <Text style={[styles.sectionTitle, { color: colors.text }]}>
                  Best for your language 🇺🇸
                </Text>
                {languageVoices.map((voice, idx) => (
                  <VoiceRow
                    key={voice.id}
                    voice={voice}
                    gradientIdx={idx + 4}
                    colors={colors}
                    isFav={isFavorite(voice.id)}
                    onToggleFav={() => toggleFavorite(voice.id)}
                    onPreview={() => handlePreviewVoice(voice)}
                    isPreviewLoading={loadingVoiceId === voice.id}
                    isPreviewPlaying={playingVoiceId === voice.id}
                  />
                ))}
              </View>
            )}
          </>
        )}

        {/* ─── Flat list for Favorites, Recents, or search results ─── */}
        {(activeTab !== 'Explore' || searchQuery) && (
          <View style={styles.section}>
            {activeTab === 'Favorites' && filteredVoices.length === 0 && !searchQuery && (
              <View style={styles.emptyState}>
                <MaterialIcons name="favorite-border" size={48} color={colors.textMuted} />
                <Text style={[styles.emptyTitle, { color: colors.text }]}>
                  No favorites yet
                </Text>
                <Text style={[styles.emptySubtitle, { color: colors.textSecondary }]}>
                  Tap the heart icon on any voice to save it here
                </Text>
              </View>
            )}
            {activeTab === 'Recents' && filteredVoices.length === 0 && !searchQuery && (
              <View style={styles.emptyState}>
                <MaterialIcons name="schedule" size={48} color={colors.textMuted} />
                <Text style={[styles.emptyTitle, { color: colors.text }]}>
                  No recent voices
                </Text>
                <Text style={[styles.emptySubtitle, { color: colors.textSecondary }]}>
                  Voices you use will appear here
                </Text>
              </View>
            )}
            {searchQuery && filteredVoices.length === 0 && (
              <View style={styles.emptyState}>
                <MaterialIcons name="search-off" size={48} color={colors.textMuted} />
                <Text style={[styles.emptyTitle, { color: colors.text }]}>
                  No matches
                </Text>
                <Text style={[styles.emptySubtitle, { color: colors.textSecondary }]}>
                  Try a different search term
                </Text>
              </View>
            )}
            {filteredVoices.map((voice, idx) => (
              <VoiceRow
                key={voice.id}
                voice={voice}
                gradientIdx={idx}
                colors={colors}
                isFav={isFavorite(voice.id)}
                onToggleFav={() => toggleFavorite(voice.id)}
                onPreview={() => handlePreviewVoice(voice)}
                isPreviewLoading={loadingVoiceId === voice.id}
                isPreviewPlaying={playingVoiceId === voice.id}
              />
            ))}
          </View>
        )}

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
  isFav,
  onToggleFav,
  onPreview,
  isPreviewLoading,
  isPreviewPlaying,
}: {
  voice: Voice;
  gradientIdx: number;
  colors: typeof Colors.dark;
  isFav: boolean;
  onToggleFav: () => void;
  onPreview: () => void;
  isPreviewLoading: boolean;
  isPreviewPlaying: boolean;
}) {
  const gc = GRADIENT_COLORS[gradientIdx % GRADIENT_COLORS.length];

  return (
    <View style={styles.voiceRow}>
      <TouchableOpacity onPress={onPreview} disabled={isPreviewLoading}>
        <View style={[styles.voiceAvatar, { backgroundColor: gc }]}>
          {isPreviewLoading ? (
            <ActivityIndicator size="small" color="#FFF" />
          ) : (
            <MaterialIcons name={isPreviewPlaying ? 'stop' : 'play-arrow'} size={24} color="#FFF" />
          )}
        </View>
      </TouchableOpacity>
      <View style={{ flex: 1 }}>
        <Text style={[styles.voiceName, { color: colors.text }]}>
          {voice.name}
        </Text>
        <Text
          style={[styles.voiceDesc, { color: colors.textMuted }]}
          numberOfLines={2}
        >
          {voice.description || `${voice.gender ?? 'AI'} Voice · ${voice.provider}`}
        </Text>
        {voice.languages && voice.languages.length > 0 && (
          <Text style={[styles.voiceLangs, { color: colors.textMuted }]}>
            {voice.languages.join(', ')}
          </Text>
        )}
      </View>
      <TouchableOpacity onPress={onToggleFav}>
        <MaterialIcons
          name={isFav ? 'favorite' : 'favorite-border'}
          size={22}
          color={isFav ? '#EF4444' : colors.textMuted}
        />
      </TouchableOpacity>
    </View>
  );
}

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
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    marginHorizontal: 24,
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
  voiceLangs: { fontSize: 11, marginTop: 2, fontStyle: 'italic' },
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
  // Empty states
  emptyState: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 60,
    gap: 8,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '700',
    marginTop: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    textAlign: 'center',
    maxWidth: 260,
  },
});
