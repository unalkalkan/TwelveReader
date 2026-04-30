import React from 'react';
import { Linking, Platform, Pressable, StyleSheet, Text } from 'react-native';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../hooks/useColorScheme';

const ATTRIBUTION_URL = 'https://deerflow.tech';

export function AttributionBadge() {
  const theme = useColorScheme();
  const colors = Colors[theme];

  const handlePress = () => {
    if (Platform.OS === 'web') {
      window.open(ATTRIBUTION_URL, '_blank', 'noopener,noreferrer');
    } else {
      Linking.openURL(ATTRIBUTION_URL);
    }
  };

  return (
    <Pressable
      onPress={handlePress}
      style={({ pressed }) => [
        styles.container,
        {
          borderColor: colors.border,
          backgroundColor: colors.background,
          opacity: pressed ? 0.92 : 0.62,
        },
      ]}
      accessibilityRole="link"
      accessibilityLabel="Created By Deerflow"
    >
      <Text style={[styles.text, { color: colors.textMuted }]}>Created By Deerflow</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderWidth: 1,
    borderRadius: 10,
    alignSelf: 'center',
  },
  text: {
    fontSize: 10,
    fontWeight: '400',
    letterSpacing: 0.3,
  },
});