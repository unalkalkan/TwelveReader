/**
 * Design tokens from Stitch "Immersive Sync Reading Player" designs.
 * Dark-mode first: background-dark #121212, surface #1E1E1E, accent-blue #3B82F6.
 */

const tintColorLight = '#3B82F6';
const tintColorDark = '#3B82F6';

const Colors = {
  light: {
    text: '#1E1E1E',
    textSecondary: '#6B7280',
    textMuted: '#9CA3AF',
    background: '#F8F9FA',
    surface: '#FFFFFF',
    card: '#E5E7EB',
    tint: tintColorLight,
    icon: '#6B7280',
    tabIconDefault: '#9CA3AF',
    tabIconSelected: tintColorLight,
    border: '#E5E7EB',
    accent: '#3B82F6',
    playerBg: 'rgba(243,244,246,0.95)',
    miniPlayerBg: 'rgba(241,245,249,0.95)',
  },
  dark: {
    text: '#F1F5F9',
    textSecondary: '#94A3B8',
    textMuted: '#64748B',
    background: '#121212',
    surface: '#1E1E1E',
    card: '#1E1E1E',
    tint: tintColorDark,
    icon: '#94A3B8',
    tabIconDefault: '#64748B',
    tabIconSelected: tintColorDark,
    border: 'rgba(255,255,255,0.05)',
    accent: '#3B82F6',
    playerBg: 'rgba(18,18,18,0.95)',
    miniPlayerBg: 'rgba(30,30,30,0.95)',
  },
};

export default Colors;
