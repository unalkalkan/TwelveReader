import { StatusBar } from 'expo-status-bar';
import { Platform, StyleSheet, View, Text } from 'react-native';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import { useServerInfo } from '../src/api/hooks';

export default function ModalScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const { data: info } = useServerInfo();

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <Text style={[styles.title, { color: colors.text }]}>
        Twelve Reader
      </Text>
      <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
        Immersive Sync Reading Player
      </Text>
      {info && (
        <Text style={[styles.version, { color: colors.textMuted }]}>
          Server v{info.version} · Storage: {info.storage_adapter}
        </Text>
      )}
      <StatusBar style={Platform.OS === 'ios' ? 'light' : 'auto'} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    marginBottom: 16,
  },
  version: {
    fontSize: 13,
  },
});
