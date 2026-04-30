import React, { useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Modal,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../hooks/useColorScheme';
import { usePersonas, useSetVoiceMap, useVoices } from '../api/hooks';
import type { Voice } from '../types/api';

interface VoiceMappingModalProps {
  bookId: string | undefined;
  visible: boolean;
  onClose: () => void;
  initialMapping?: boolean;
}

function formatVoiceLabel(voice: Voice): string {
  const details = [voice.provider, voice.gender, voice.accent]
    .filter(Boolean)
    .join(' · ');
  return details ? `${voice.name} (${details})` : voice.name;
}

function chooseDefaultVoice(persona: string, voices: Voice[], index: number): string {
  if (!voices.length) return '';
  if (persona.toLowerCase().includes('narrator')) {
    const narratorVoice = voices.find((voice) =>
      `${voice.name} ${voice.description ?? ''}`.toLowerCase().includes('narr'),
    );
    if (narratorVoice) return narratorVoice.id;
  }
  return voices[index % voices.length]?.id ?? voices[0].id;
}

export function VoiceMappingModal({
  bookId,
  visible,
  onClose,
  initialMapping = true,
}: VoiceMappingModalProps) {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const { data: personas, isLoading: personasLoading, refetch: refetchPersonas } =
    usePersonas(visible ? bookId : undefined);
  const { data: voicesData, isLoading: voicesLoading, refetch: refetchVoices } =
    useVoices();
  const setVoiceMap = useSetVoiceMap(bookId ?? '');
  const [selected, setSelected] = useState<Record<string, string>>({});

  const voices = voicesData?.voices ?? [];
  const allPersonas = useMemo(() => {
    const names = new Set<string>();
    personas?.discovered?.forEach((persona) => names.add(persona));
    personas?.unmapped?.forEach((persona) => names.add(persona));
    Object.keys(personas?.mapped ?? {}).forEach((persona) => names.add(persona));
    return Array.from(names).sort((a, b) => {
      if (a.toLowerCase() === 'narrator') return -1;
      if (b.toLowerCase() === 'narrator') return 1;
      return a.localeCompare(b);
    });
  }, [personas]);

  useEffect(() => {
    if (!visible || !bookId) return;
    refetchPersonas();
    refetchVoices();
  }, [bookId, visible, refetchPersonas, refetchVoices]);

  useEffect(() => {
    if (!visible || !allPersonas.length || !voices.length) return;
    setSelected((current) => {
      const next: Record<string, string> = { ...current };
      allPersonas.forEach((persona, index) => {
        if (!next[persona]) {
          next[persona] =
            personas?.mapped?.[persona] ?? chooseDefaultVoice(persona, voices, index);
        }
      });
      return next;
    });
  }, [allPersonas, personas?.mapped, visible, voices]);

  const isLoading = personasLoading || voicesLoading;
  const canSubmit =
    !!bookId &&
    allPersonas.length > 0 &&
    allPersonas.every((persona) => Boolean(selected[persona])) &&
    !setVoiceMap.isPending;

  const submitMapping = async () => {
    if (!bookId || !canSubmit) return;
    try {
      await setVoiceMap.mutateAsync({
        voiceMap: {
          persons: allPersonas.map((persona) => ({
            id: persona,
            provider_voice: selected[persona],
          })),
        },
        options: initialMapping ? { initial: true } : { update: true },
      });
      Alert.alert('Voice mapping saved', 'TwelveReader will continue synthesis with these voices.');
      onClose();
    } catch (error: any) {
      Alert.alert('Mapping failed', error?.message ?? 'Could not save the voice map.');
    }
  };

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <View style={styles.backdrop}>
        <View style={[styles.sheet, { backgroundColor: colors.background, borderColor: colors.border }]}>
          <View style={styles.header}>
            <View>
              <Text style={[styles.eyebrow, { color: colors.accent }]}>VOICE MAPPING</Text>
              <Text style={[styles.title, { color: colors.text }]}>Assign voices to personas</Text>
            </View>
            <TouchableOpacity style={[styles.iconButton, { backgroundColor: colors.surface }]} onPress={onClose}>
              <MaterialIcons name="close" size={20} color={colors.textSecondary} />
            </TouchableOpacity>
          </View>

          <Text style={[styles.description, { color: colors.textSecondary }]}>
            Map every discovered speaker before synthesis continues. Defaults are preselected; adjust any persona that should use a different TTS voice.
          </Text>

          {isLoading ? (
            <View style={styles.centerState}>
              <ActivityIndicator color={colors.accent} />
              <Text style={{ color: colors.textMuted, marginTop: 12 }}>Loading personas and voices...</Text>
            </View>
          ) : !allPersonas.length ? (
            <View style={styles.centerState}>
              <MaterialIcons name="person-search" size={42} color={colors.textMuted} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>No personas discovered yet</Text>
              <Text style={[styles.emptyText, { color: colors.textSecondary }]}>Try again after segmentation has processed more text.</Text>
            </View>
          ) : !voices.length ? (
            <View style={styles.centerState}>
              <MaterialIcons name="record-voice-over" size={42} color={colors.textMuted} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>No voices available</Text>
              <Text style={[styles.emptyText, { color: colors.textSecondary }]}>Configure at least one TTS provider, then refresh voices.</Text>
            </View>
          ) : (
            <ScrollView style={styles.mappingList} showsVerticalScrollIndicator={false}>
              {allPersonas.map((persona, personaIndex) => (
                <View key={persona} style={[styles.personaCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
                  <View style={styles.personaHeader}>
                    <View style={[styles.personaAvatar, { backgroundColor: colors.card }]}>
                      <MaterialIcons name="person" size={18} color={colors.accent} />
                    </View>
                    <View style={{ flex: 1 }}>
                      <Text style={[styles.personaName, { color: colors.text }]}>{persona}</Text>
                      <Text style={[styles.personaMeta, { color: colors.textMuted }]}>
                        {personas?.mapped?.[persona] ? 'Already mapped' : 'Needs voice'}
                      </Text>
                    </View>
                  </View>
                  <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.voiceChoices}>
                    {voices.map((voice, voiceIndex) => {
                      const active = selected[persona] === voice.id;
                      return (
                        <TouchableOpacity
                          key={`${persona}-${voice.provider}-${voice.id}`}
                          onPress={() => setSelected((current) => ({ ...current, [persona]: voice.id }))}
                          style={[
                            styles.voiceChip,
                            {
                              backgroundColor: active ? colors.accent : colors.card,
                              borderColor: active ? colors.accent : colors.border,
                            },
                          ]}
                        >
                          <Text style={[styles.voiceChipText, { color: active ? '#FFFFFF' : colors.text }]} numberOfLines={1}>
                            {formatVoiceLabel(voice)}
                          </Text>
                          <Text style={[styles.voiceChipMeta, { color: active ? 'rgba(255,255,255,0.75)' : colors.textMuted }]}>
                            Option {voiceIndex + 1} for #{personaIndex + 1}
                          </Text>
                        </TouchableOpacity>
                      );
                    })}
                  </ScrollView>
                </View>
              ))}
            </ScrollView>
          )}

          <View style={styles.footer}>
            <TouchableOpacity style={[styles.secondaryButton, { borderColor: colors.border }]} onPress={onClose}>
              <Text style={[styles.secondaryButtonText, { color: colors.textSecondary }]}>Later</Text>
            </TouchableOpacity>
            <TouchableOpacity
              disabled={!canSubmit}
              style={[
                styles.primaryButton,
                { backgroundColor: canSubmit ? colors.accent : colors.card, opacity: canSubmit ? 1 : 0.6 },
              ]}
              onPress={submitMapping}
            >
              {setVoiceMap.isPending ? (
                <ActivityIndicator color="#FFFFFF" />
              ) : (
                <>
                  <MaterialIcons name="check" size={18} color="#FFFFFF" />
                  <Text style={styles.primaryButtonText}>Save mapping</Text>
                </>
              )}
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    justifyContent: 'flex-end',
    backgroundColor: 'rgba(2, 6, 23, 0.72)',
  },
  sheet: {
    maxHeight: '88%',
    borderTopLeftRadius: 18,
    borderTopRightRadius: 18,
    borderWidth: StyleSheet.hairlineWidth,
    paddingHorizontal: 20,
    paddingTop: 18,
    paddingBottom: 20,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
  },
  eyebrow: { fontSize: 11, fontWeight: '800', letterSpacing: 1.1 },
  title: { fontSize: 22, fontWeight: '800', marginTop: 4 },
  iconButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  description: { fontSize: 14, lineHeight: 20, marginTop: 14, marginBottom: 12 },
  centerState: { alignItems: 'center', justifyContent: 'center', paddingVertical: 48 },
  emptyTitle: { fontSize: 16, fontWeight: '700', marginTop: 12 },
  emptyText: { fontSize: 13, textAlign: 'center', marginTop: 6, lineHeight: 18 },
  mappingList: { marginTop: 4 },
  personaCard: {
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    padding: 14,
    marginBottom: 12,
  },
  personaHeader: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  personaAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: 'center',
    justifyContent: 'center',
  },
  personaName: { fontSize: 16, fontWeight: '800' },
  personaMeta: { fontSize: 12, marginTop: 2 },
  voiceChoices: { gap: 8, paddingTop: 12, paddingBottom: 2 },
  voiceChip: {
    width: 210,
    borderRadius: 10,
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  voiceChipText: { fontSize: 13, fontWeight: '700' },
  voiceChipMeta: { fontSize: 11, marginTop: 4 },
  footer: { flexDirection: 'row', gap: 12, marginTop: 16 },
  secondaryButton: {
    flex: 1,
    height: 48,
    borderWidth: 1,
    borderRadius: 10,
    alignItems: 'center',
    justifyContent: 'center',
  },
  secondaryButtonText: { fontSize: 14, fontWeight: '700' },
  primaryButton: {
    flex: 1.6,
    height: 48,
    borderRadius: 10,
    flexDirection: 'row',
    gap: 8,
    alignItems: 'center',
    justifyContent: 'center',
  },
  primaryButtonText: { color: '#FFFFFF', fontSize: 14, fontWeight: '800' },
});
