import React from 'react';
import {
  View,
  Text,
  Image,
  TouchableOpacity,
  StyleSheet,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { useColorScheme } from '../hooks/useColorScheme';
import Colors from '../../constants/Colors';

interface MiniPlayerProps {
  bookId?: string;
  title?: string;
  author?: string;
  coverUrl?: string;
  isPlaying?: boolean;
  onPlayPause?: () => void;
}

export function MiniPlayer({
  bookId,
  title = 'Perfume',
  author = 'Patrick Süskind',
  coverUrl,
  isPlaying = false,
  onPlayPause,
}: MiniPlayerProps) {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();

  if (!bookId && !title) return null;

  return (
    <TouchableOpacity
      activeOpacity={0.9}
      onPress={() => bookId && router.push(`/player?bookId=${bookId}`)}
      style={[styles.container, { backgroundColor: colors.miniPlayerBg }]}
    >
      <Image
        source={
          coverUrl
            ? { uri: coverUrl }
            : require('../../assets/images/icon.png')
        }
        style={styles.cover}
      />
      <View style={styles.info}>
        <Text
          style={[styles.author, { color: colors.textMuted }]}
          numberOfLines={1}
        >
          {author?.toUpperCase()}
        </Text>
        <Text
          style={[styles.title, { color: colors.text }]}
          numberOfLines={1}
        >
          {title}
        </Text>
      </View>
      <View style={styles.controls}>
        <TouchableOpacity onPress={onPlayPause} hitSlop={8}>
          <MaterialIcons
            name={isPlaying ? 'pause' : 'play-arrow'}
            size={28}
            color={colors.text}
          />
        </TouchableOpacity>
        <TouchableOpacity hitSlop={8} style={{ marginLeft: 16 }}>
          <MaterialIcons name="replay-30" size={24} color={colors.text} />
        </TouchableOpacity>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    borderRadius: 16,
    gap: 12,
    // shadow
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.25,
    shadowRadius: 24,
    elevation: 12,
  },
  cover: {
    width: 40,
    height: 40,
    borderRadius: 8,
  },
  info: {
    flex: 1,
    minWidth: 0,
  },
  author: {
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 1,
    marginBottom: 1,
  },
  title: {
    fontSize: 14,
    fontWeight: '700',
  },
  controls: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 8,
  },
});
