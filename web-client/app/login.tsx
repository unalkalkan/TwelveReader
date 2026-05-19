/**
 * Login screen — magic link authentication.
 * Shown after server selection. User enters email, requests a magic link,
 * then either clicks it (deep link) or pastes the token manually.
 */

import React, { useState, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { useRouter, useLocalSearchParams, usePathname } from 'expo-router';
import AsyncStorage from '@react-native-async-storage/async-storage';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import { useAuth } from '../src/store/authStore';

export default function LoginScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const pathname = usePathname();
  const { initialized, isAuthenticated, loginRequestMagicLink, loginVerifyToken } = useAuth();

  // Deep link params: ?token=xxx from magic link
  const params = useLocalSearchParams();
  const deepLinkToken = params.token as string | undefined;

  const [email, setEmail] = useState('');
  const [manualToken, setManualToken] = useState('');
  const [mode, setMode] = useState<'request' | 'sent' | 'manual'>('request');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Auto-verify deep link token if present
  useEffect(() => {
    if (deepLinkToken && deepLinkToken.trim().length > 0) {
      handleVerifyToken(deepLinkToken);
    }
  }, [deepLinkToken]);

  // Redirect to main app if already authenticated
  useEffect(() => {
    if (initialized && isAuthenticated && pathname !== '/(tabs)') {
      router.replace('/(tabs)');
    }
  }, [initialized, isAuthenticated, pathname, router]);

  const handleRequestMagicLink = useCallback(async () => {
    if (!email.trim()) {
      setError('Please enter your email address.');
      return;
    }

    // Basic email validation
    if (!email.includes('@')) {
      setError('Please enter a valid email address.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await loginRequestMagicLink(email);
      setMode('sent');
    } catch (err: any) {
      setError(err.message || 'Failed to send magic link.');
    } finally {
      setLoading(false);
    }
  }, [email, loginRequestMagicLink]);

  const handleVerifyToken = useCallback(async (token: string) => {
    setLoading(true);
    setError(null);

    try {
      await loginVerifyToken(token.trim());

      // Mark server as validated + authenticated
      await AsyncStorage.setItem('twelvereader_server_validated', 'true');
      await AsyncStorage.setItem('twelvereader_authenticated', 'true');

      router.replace('/(tabs)');
    } catch (err: any) {
      setError(err.message || 'Invalid or expired token.');
      setMode('manual');
    } finally {
      setLoading(false);
    }
  }, [loginVerifyToken, router]);

  const handleManualTokenSubmit = useCallback(async () => {
    if (!manualToken.trim()) {
      setError('Please paste the magic link or token.');
      return;
    }

    // Extract token from full URL if user pasted the whole link
    let token = manualToken.trim();

    // If it looks like a full URL, extract the token query parameter
    try {
      if (token.includes('://') || token.startsWith('/')) {
        const urlMatch = token.match(/[?&]token=([^&]+)/);
        if (urlMatch) {
          token = decodeURIComponent(urlMatch[1]);
        } else {
          // Maybe it's a deep link: twelvereader://auth/verify?token=xxx
          const pathParts = token.split('?');
          if (pathParts.length > 1) {
            const searchParams = new URLSearchParams(pathParts[1]);
            token = searchParams.get('token') || token;
          }
        }
      }
    } catch {
      // ignore parsing errors, use raw input
    }

    await handleVerifyToken(token);
  }, [manualToken, handleVerifyToken]);

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={[styles.container, { backgroundColor: colors.background }]}
    >
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Logo / Header */}
        <View style={styles.header}>
          <MaterialIcons name="login" size={48} color={colors.accent} />
          <Text style={[styles.title, { color: colors.text }]}>Sign in</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            {mode === 'sent'
              ? 'Check your email for the magic link'
              : 'Enter your email to sign in'}
          </Text>
        </View>

        {/* Request magic link mode */}
        {(mode === 'request' || mode === 'manual') && (
          <>
            <View style={styles.inputWrapper}>
              <Text style={[styles.inputLabel, { color: colors.textSecondary }]}>
                Email address
              </Text>
              <TextInput
                style={[
                  styles.input,
                  {
                    backgroundColor: colors.surface,
                    color: colors.text,
                    borderColor: colors.border,
                  },
                ]}
                placeholder="you@example.com"
                placeholderTextColor={colors.textMuted}
                value={email}
                onChangeText={(text) => { setEmail(text); setError(null); }}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="email-address"
                returnKeyType="next"
                onSubmitEditing={() => {
                  if (mode === 'request') handleRequestMagicLink();
                }}
              />
            </View>

            {/* Manual token input */}
            {mode === 'manual' && (
              <View style={styles.inputWrapper}>
                <Text style={[styles.inputLabel, { color: colors.textSecondary }]}>
                  Magic link or token
                </Text>
                <TextInput
                  style={[
                    styles.input,
                    {
                      backgroundColor: colors.surface,
                      color: colors.text,
                      borderColor: colors.border,
                    },
                  ]}
                  placeholder="Paste the full link or just the token"
                  placeholderTextColor={colors.textMuted}
                  value={manualToken}
                  onChangeText={(text) => { setManualToken(text); setError(null); }}
                  autoCapitalize="none"
                  autoCorrect={false}
                  returnKeyType="done"
                  onSubmitEditing={handleManualTokenSubmit}
                />
              </View>
            )}

            {/* Switch to manual mode */}
            {mode === 'request' && (
              <TouchableOpacity onPress={() => setMode('manual')}>
                <Text style={[styles.switchLink, { color: colors.accent }]}>
                  Paste link manually instead
                </Text>
              </TouchableOpacity>
            )}
          </>
        )}

        {/* Email sent confirmation */}
        {mode === 'sent' && (
          <>
            <View style={[styles.infoBox, { borderColor: colors.accent, backgroundColor: 'rgba(59,130,246,0.1)' }]}>
              <MaterialIcons name="email" size={20} color={colors.accent} />
              <View style={{ flex: 1, marginLeft: 8 }}>
                <Text style={[styles.infoText, { color: colors.text }]}>
                  Magic link sent to
                </Text>
                <Text style={[styles.infoTextBold, { color: colors.text }]}>
                  {email}
                </Text>
              </View>
            </View>

            <TouchableOpacity onPress={() => setMode('manual')}>
              <Text style={[styles.switchLink, { color: colors.accent }]}>
                Didn't receive the email? Paste link manually
              </Text>
            </TouchableOpacity>
          </>
        )}

        {/* Error message */}
        {error && (
          <View style={[styles.errorBox, { borderColor: '#EF4444', backgroundColor: 'rgba(239,68,68,0.1)' }]}>
            <MaterialIcons name="error-outline" size={20} color="#EF4444" />
            <Text style={[styles.errorText, { color: '#EF4444' }]}>{error}</Text>
          </View>
        )}

        {/* Primary action button */}
        {(mode === 'request' || mode === 'manual') && (
          <TouchableOpacity
            style={[
              styles.actionButton,
              {
                backgroundColor: colors.accent,
                opacity: loading ? 0.7 : 1,
              },
            ]}
            onPress={mode === 'request' ? handleRequestMagicLink : handleManualTokenSubmit}
            disabled={loading}
          >
            {loading ? (
              <ActivityIndicator color="#FFFFFF" />
            ) : (
              <>
                <MaterialIcons
                  name={mode === 'request' ? 'send' : 'login'}
                  size={20}
                  color="#FFFFFF"
                />
                <Text style={styles.actionButtonText}>
                  {mode === 'request' ? 'Send magic link' : 'Sign in with token'}
                </Text>
              </>
            )}
          </TouchableOpacity>
        )}

        {/* Resend button in sent mode */}
        {mode === 'sent' && (
          <TouchableOpacity
            style={[
              styles.actionButton,
              {
                backgroundColor: colors.accent,
                opacity: loading ? 0.7 : 1,
              },
            ]}
            onPress={handleRequestMagicLink}
            disabled={loading}
          >
            {loading ? (
              <ActivityIndicator color="#FFFFFF" />
            ) : (
              <>
                <MaterialIcons name="restart-alt" size={20} color="#FFFFFF" />
                <Text style={styles.actionButtonText}>Resend magic link</Text>
              </>
            )}
          </TouchableOpacity>
        )}

        {/* Footer */}
        <Text style={[styles.footer, { color: colors.textMuted }]}>
          Open the email and click the link to sign in automatically.
        </Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    padding: 24,
  },
  header: {
    alignItems: 'center',
    marginTop: 60,
    marginBottom: 40,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    marginTop: 16,
    letterSpacing: -0.5,
  },
  subtitle: {
    fontSize: 16,
    marginTop: 8,
  },
  inputWrapper: {
    marginBottom: 12,
  },
  inputLabel: {
    fontSize: 14,
    fontWeight: '500',
    marginBottom: 8,
  },
  input: {
    borderWidth: 1,
    borderRadius: 8,
    padding: 12,
    fontSize: 15,
  },
  errorBox: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    borderRadius: 8,
    marginBottom: 16,
    borderWidth: 1,
  },
  errorText: {
    flex: 1,
    marginLeft: 8,
    fontSize: 14,
  },
  infoBox: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    borderRadius: 8,
    marginBottom: 16,
    borderWidth: 1,
  },
  infoText: {
    fontSize: 14,
  },
  infoTextBold: {
    fontSize: 14,
    fontWeight: '600',
    marginTop: 2,
  },
  switchLink: {
    textAlign: 'center',
    fontSize: 14,
    marginBottom: 16,
    textDecorationLine: 'underline',
  },
  actionButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 14,
    borderRadius: 10,
    marginTop: 24,
    marginBottom: 8,
  },
  actionButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
    marginLeft: 8,
  },
  footer: {
    textAlign: 'center',
    fontSize: 13,
    marginTop: 12,
  },
});
