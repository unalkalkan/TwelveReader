/**
 * Pre-login server selection screen.
 * Shown before any app content. User selects official or custom server,
 * validates it via /api/v1/server-info, then proceeds.
 */

import React, { useState, useCallback } from 'react';
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
import { useRouter } from 'expo-router';
import AsyncStorage from '@react-native-async-storage/async-storage';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import { OFFICIAL_SERVER_URL, useServerConfig } from '../src/store/serverConfigStore';
import { validateServerUrl, setApiBase } from '../src/api/client';

export default function ServerSelectScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { setServerUrl } = useServerConfig();

  const [customUrl, setCustomUrl] = useState('');
  const [useCustom, setUseCustom] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [serverInfo, setServerInfo] = useState<{ version: string; environment: string } | null>(null);

  const currentUrl = useCustom ? customUrl.trim() : OFFICIAL_SERVER_URL;

  const handleConnect = useCallback(async () => {
    if (!currentUrl) {
      setError('Please enter a server URL.');
      return;
    }

    // Basic URL validation
    let normalized: string;
    try {
      const urlObj = new URL(currentUrl.startsWith('http') ? currentUrl : `https://${currentUrl}`);
      normalized = urlObj.origin;
    } catch {
      setError('Invalid URL format. Use http(s)://host:port');
      return;
    }

    setLoading(true);
    setError(null);
    setServerInfo(null);

    try {
      const info = await validateServerUrl(normalized);
      // Set API base immediately for the current session
      setApiBase(normalized + '/api/v1');
      setServerInfo({ version: info.version, environment: info.environment });

      // Persist through provider (clears validation flag on URL change)
      setServerUrl(normalized);

      // Mark server as validated
      await AsyncStorage.setItem('twelvereader_server_validated', 'true');

      // Navigate to main app
      router.replace('/(tabs)');
    } catch (err: any) {
      setError(err.message || 'Failed to connect to server.');
    } finally {
      setLoading(false);
    }
  }, [currentUrl, router, setServerUrl]);

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={[styles.container, { backgroundColor: colors.background }]}
    >
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Logo / Header */}
        <View style={styles.header}>
          <MaterialIcons name="headset" size={48} color={colors.accent} />
          <Text style={[styles.title, { color: colors.text }]}>TwelveReader</Text>
          <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
            Select your audiobook server
          </Text>
        </View>

        {/* Official server option */}
        <TouchableOpacity
          style={[
            styles.serverOption,
            {
              backgroundColor: colors.surface,
              borderColor: !useCustom ? colors.accent : colors.border,
              borderWidth: !useCustom ? 2 : 1,
            },
          ]}
          onPress={() => { setUseCustom(false); setError(null); }}
        >
          <MaterialIcons
            name="verified"
            size={24}
            color={!useCustom ? colors.accent : colors.textMuted}
          />
          <View style={styles.serverOptionText}>
            <Text style={[styles.serverOptionTitle, { color: colors.text }]}>
              Official Server
            </Text>
            <Text style={[styles.serverOptionUrl, { color: colors.textSecondary }]}>
              {OFFICIAL_SERVER_URL}
            </Text>
          </View>
          <MaterialIcons
            name={useCustom ? 'radio-button-unchecked' : 'radio-button-checked'}
            size={24}
            color={!useCustom ? colors.accent : colors.textMuted}
          />
        </TouchableOpacity>

        {/* Custom server option */}
        <TouchableOpacity
          style={[
            styles.serverOption,
            {
              backgroundColor: colors.surface,
              borderColor: useCustom ? colors.accent : colors.border,
              borderWidth: useCustom ? 2 : 1,
            },
          ]}
          onPress={() => { setUseCustom(true); setError(null); }}
        >
          <MaterialIcons
            name="dns"
            size={24}
            color={useCustom ? colors.accent : colors.textMuted}
          />
          <View style={styles.serverOptionText}>
            <Text style={[styles.serverOptionTitle, { color: colors.text }]}>
              Self-Hosted Server
            </Text>
            <Text style={[styles.serverOptionUrl, { color: colors.textSecondary }]}>
              Enter your server URL below
            </Text>
          </View>
          <MaterialIcons
            name={useCustom ? 'radio-button-checked' : 'radio-button-unchecked'}
            size={24}
            color={useCustom ? colors.accent : colors.textMuted}
          />
        </TouchableOpacity>

        {/* Custom URL input (only when custom selected) */}
        {useCustom && (
          <View style={styles.inputWrapper}>
            <Text style={[styles.inputLabel, { color: colors.textSecondary }]}>
              Server URL
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
              placeholder="http://localhost:8080 or https://myserver.com"
              placeholderTextColor={colors.textMuted}
              value={customUrl}
              onChangeText={(text) => { setCustomUrl(text); setError(null); }}
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
              returnKeyType="done"
              onSubmitEditing={handleConnect}
            />
          </View>
        )}

        {/* Error message */}
        {error && (
          <View style={[styles.errorBox, { borderColor: '#EF4444', backgroundColor: 'rgba(239,68,68,0.1)' }]}>
            <MaterialIcons name="error-outline" size={20} color="#EF4444" />
            <Text style={[styles.errorText, { color: '#EF4444' }]}>{error}</Text>
          </View>
        )}

        {/* Server info (shown after validation) */}
        {serverInfo && (
          <View style={[styles.infoBox, { borderColor: colors.accent, backgroundColor: 'rgba(59,130,246,0.1)' }]}>
            <MaterialIcons name="check-circle" size={20} color={colors.accent} />
            <View>
              <Text style={[styles.infoText, { color: colors.text }]}>
                Connected to server v{serverInfo.version} ({serverInfo.environment})
              </Text>
            </View>
          </View>
        )}

        {/* Connect button */}
        <TouchableOpacity
          style={[
            styles.connectButton,
            {
              backgroundColor: colors.accent,
              opacity: loading ? 0.7 : 1,
            },
          ]}
          onPress={handleConnect}
          disabled={loading}
        >
          {loading ? (
            <ActivityIndicator color="#FFFFFF" />
          ) : (
            <>
              <MaterialIcons name="login" size={20} color="#FFFFFF" />
              <Text style={styles.connectButtonText}>Connect</Text>
            </>
          )}
        </TouchableOpacity>

        {/* Footer note */}
        <Text style={[styles.footer, { color: colors.textMuted }]}>
          You can change your server later from Settings.
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
  serverOption: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
    borderRadius: 12,
    marginBottom: 12,
  },
  serverOptionText: {
    flex: 1,
    marginLeft: 12,
  },
  serverOptionTitle: {
    fontSize: 16,
    fontWeight: '600',
  },
  serverOptionUrl: {
    fontSize: 13,
    marginTop: 2,
  },
  inputWrapper: {
    marginBottom: 16,
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
    flex: 1,
    marginLeft: 8,
    fontSize: 14,
  },
  connectButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 14,
    borderRadius: 10,
    marginTop: 24,
    marginBottom: 8,
  },
  connectButtonText: {
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
