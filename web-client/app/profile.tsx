import React from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
  TouchableOpacity,
  Platform,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import { useUserProfile } from '../src/api/hooks';
import { useAuth } from '../src/store/authStore';

// ── Helpers ────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  } catch {
    return iso;
  }
}

function quotaPercent(used: number, limit: number): number {
  // -1 means unlimited
  if (limit <= 0) return 0;
  return Math.min(Math.round((used / limit) * 100), 100);
}

// ── Components ─────────────────────────────────────────────────────────

function ProfileSection({
  title,
  icon,
  children,
  colors,
}: {
  title: string;
  icon: keyof typeof MaterialIcons.glyphMap;
  children: React.ReactNode;
  colors: Record<string, string>;
}) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeader}>
        <MaterialIcons name={icon} size={20} color={colors.accent} />
        <Text style={[styles.sectionTitle, { color: colors.text }]}>{title}</Text>
      </View>
      {children}
    </View>
  );
}

function ProfileRow({
  label,
  value,
  colors,
}: {
  label: string;
  value: string;
  colors: Record<string, string>;
}) {
  return (
    <View style={styles.profileRow}>
      <Text style={[styles.profileLabel, { color: colors.textMuted }]}>{label}</Text>
      <Text style={[styles.profileValue, { color: colors.text }]} numberOfLines={1}>
        {value}
      </Text>
    </View>
  );
}

function UsageCard({
  label,
  value,
  icon,
  colors,
}: {
  label: string;
  value: string;
  icon: keyof typeof MaterialIcons.glyphMap;
  colors: Record<string, string>;
}) {
  return (
    <View style={[styles.usageCard, { backgroundColor: colors.card }]}>
      <MaterialIcons name={icon} size={24} color={colors.accent} />
      <Text style={[styles.usageValue, { color: colors.text }]}>{value}</Text>
      <Text style={[styles.usageLabel, { color: colors.textMuted }]}>{label}</Text>
    </View>
  );
}

function QuotaBar({
  label,
  used,
  limit,
  unit,
  colors,
}: {
  label: string;
  used: number;
  limit: number;
  unit: string;
  colors: Record<string, string>;
}) {
  const isUnlimited = limit <= 0;
  const pct = quotaPercent(used, limit);
  const displayLimit = isUnlimited ? 'Unlimited' : `${limit} ${unit}`;

  return (
    <View style={styles.quotaRow}>
      <Text style={[styles.quotaLabel, { color: colors.text }]}>{label}</Text>
      <Text style={[styles.quotaValue, { color: colors.textMuted }]}>
        {used} / {displayLimit} {unit}
      </Text>
      {!isUnlimited && (
        <>
          <View style={[styles.quotaBarBg, { backgroundColor: colors.border }]}>
            <View
              style={[
                styles.quotaBarFill,
                {
                  width: `${pct}%`,
                  backgroundColor: pct > 90 ? '#EF4444' : pct > 70 ? '#F59E0B' : colors.accent,
                },
              ]}
            />
          </View>
          <Text style={[styles.quotaPercent, { color: colors.textMuted }]}>{pct}%</Text>
        </>
      )}
    </View>
  );
}

// ── Main Screen ────────────────────────────────────────────────────────

export default function ProfileScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { logout } = useAuth();
  const { data: profile, isLoading, error } = useUserProfile();

  const handleLogout = async () => {
    await logout();
    // After logout, navigation will be handled by auth redirect in _layout.tsx
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity
          onPress={() => router.back()}
          style={[styles.iconBtn, { backgroundColor: colors.card }]}
        >
          <MaterialIcons name="arrow-back" size={20} color={colors.text} />
        </TouchableOpacity>
        <Text style={[styles.headerTitle, { color: colors.text }]}>Profile</Text>
        <View style={{ width: 40 }} />
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        {isLoading && (
          <View style={styles.loadingContainer}>
            <ActivityIndicator size="large" color={colors.accent} />
          </View>
        )}

        {error && (
          <View style={[styles.errorCard, { backgroundColor: colors.card }]}>
            <MaterialIcons name="error-outline" size={24} color="#EF4444" />
            <Text style={{ color: '#EF4444', marginTop: 8, textAlign: 'center' }}>
              Failed to load profile. Please try again later.
            </Text>
          </View>
        )}

        {!isLoading && !error && profile && (
          <>
            {/* ─── User Info ─── */}
            <ProfileSection title="Account" icon="person" colors={colors}>
              <View style={[styles.profileCard, { backgroundColor: colors.card }]}>
                <View style={[styles.avatarCircle, { backgroundColor: colors.accent }]}>
                  <Text style={styles.avatarText}>
                    {(profile.user.name || profile.user.email)?.[0]?.toUpperCase() ?? '?'}
                  </Text>
                </View>
                <ProfileRow
                  label="Email"
                  value={profile.user.email}
                  colors={colors}
                />
                {profile.user.name && (
                  <ProfileRow
                    label="Name"
                    value={profile.user.name}
                    colors={colors}
                  />
                )}
                <ProfileRow
                  label="Role"
                  value={profile.user.role_name || profile.user.role_id}
                  colors={colors}
                />
                <ProfileRow
                  label="Status"
                  value={profile.user.status}
                  colors={colors}
                />
                <ProfileRow
                  label="Member since"
                  value={formatDate(profile.user.created_at)}
                  colors={colors}
                />
              </View>
            </ProfileSection>

            {/* ─── Usage ─── */}
            <ProfileSection title="Usage" icon="bar-chart" colors={colors}>
              {!profile.metering_active && (
                <View style={[styles.meteringNotice, { backgroundColor: colors.card }]}>
                  <MaterialIcons name="info-outline" size={16} color={colors.accent} />
                  <Text style={{ color: colors.textMuted, fontSize: 12 }}>
                    Usage metering not yet active. Values below are placeholders.
                  </Text>
                </View>
              )}
              <View style={styles.usageGrid}>
                <UsageCard
                  label="Books"
                  value={String(profile.usage.books_uploaded)}
                  icon="book"
                  colors={colors}
                />
                <UsageCard
                  label="TTS (min)"
                  value={String(Math.round(profile.usage.tts_minutes))}
                  icon="record-voice-over"
                  colors={colors}
                />
                <UsageCard
                  label="Storage"
                  value={formatBytes(profile.usage.storage_bytes)}
                  icon="storage"
                  colors={colors}
                />
                <UsageCard
                  label="Segments"
                  value={String(profile.usage.total_segments)}
                  icon="article"
                  colors={colors}
                />
              </View>
            </ProfileSection>

            {/* ─── Quota ─── */}
            <ProfileSection title="Quota" icon="speed" colors={colors}>
              <View style={[styles.profileCard, { backgroundColor: colors.card }]}>
                <ProfileRow
                  label="Plan"
                  value={profile.quota.plan.charAt(0).toUpperCase() + profile.quota.plan.slice(1)}
                  colors={colors}
                />
                <QuotaBar
                  label="TTS Minutes"
                  used={Math.round(profile.quota.tts_minutes_used)}
                  limit={profile.quota.tts_minutes_limit}
                  unit="min"
                  colors={colors}
                />
                <QuotaBar
                  label="Storage"
                  used={parseFloat(profile.quota.storage_gb_used.toFixed(2))}
                  limit={Math.round(profile.quota.storage_gb_limit)}
                  unit="GB"
                  colors={colors}
                />
                <QuotaBar
                  label="Books"
                  used={profile.quota.books_used}
                  limit={profile.quota.books_limit}
                  unit=""
                  colors={colors}
                />
              </View>
            </ProfileSection>

            {/* ─── Actions ─── */}
            <View style={styles.actions}>
              <TouchableOpacity
                onPress={handleLogout}
                style={[styles.logoutBtn, { backgroundColor: colors.card, borderColor: colors.border }]}
              >
                <MaterialIcons name="logout" size={20} color="#EF4444" />
                <Text style={{ color: '#EF4444', fontWeight: '600' }}>Log out</Text>
              </TouchableOpacity>
            </View>
          </>
        )}

        {/* Bottom spacer */}
        <View style={{ height: 40 }} />
      </ScrollView>
    </SafeAreaView>
  );
}

// ── Styles ─────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  safe: { flex: 1 },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 24,
    paddingVertical: 12,
  },
  headerTitle: { fontSize: 18, fontWeight: '700' },
  iconBtn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scrollContent: { paddingHorizontal: 24 },
  section: { marginBottom: 24 },
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 12,
  },
  sectionTitle: { fontSize: 16, fontWeight: '700' },
  // Profile card
  profileCard: { borderRadius: 16, padding: 16, gap: 12 },
  avatarCircle: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
    alignSelf: 'center',
    marginBottom: 4,
  },
  avatarText: { color: '#FFF', fontSize: 22, fontWeight: '700' },
  profileRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 6,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(128,128,128,0.15)',
  },
  profileLabel: { fontSize: 14 },
  profileValue: { fontSize: 14, fontWeight: '500', textAlign: 'right', flex: 1, marginLeft: 16 },
  // Usage grid
  usageGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: 12, justifyContent: 'space-between' },
  usageCard: {
    flexBasis: '48%',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    gap: 4,
  },
  usageValue: { fontSize: 20, fontWeight: '700', marginTop: 4 },
  usageLabel: { fontSize: 12 },
  // Quota
  quotaRow: { marginBottom: 16 },
  quotaLabel: { fontSize: 14, fontWeight: '600', marginBottom: 4 },
  quotaValue: { fontSize: 12, marginBottom: 4 },
  quotaBarBg: { height: 6, borderRadius: 3, overflow: 'hidden' },
  quotaBarFill: { height: '100%', borderRadius: 3 },
  quotaPercent: { fontSize: 11, textAlign: 'right', marginTop: 2 },
  // Metering notice
  meteringNotice: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    borderRadius: 10,
    padding: 12,
    marginBottom: 12,
  },
  // Loading/error
  loadingContainer: { padding: 40, alignItems: 'center' },
  errorCard: { borderRadius: 12, padding: 20, alignItems: 'center', gap: 8 },
  // Actions
  actions: { marginBottom: 16 },
  logoutBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: 12,
    paddingVertical: 14,
    borderWidth: 1,
  },
});
