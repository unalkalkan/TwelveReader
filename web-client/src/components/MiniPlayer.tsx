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
import { usePlayback } from '../store/playbackStore';
import { useBook } from '../api/hooks';

export function MiniPlayer() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { state, togglePlayback, seekToSegment } = usePlayback();
  const { data: book } = useBook(state.currentBookId ?? undefined);

  // Don't render if no active book
  if (!state.currentBookId) return null;

  const title = book?.title ?? 'Loading...';
  const author = book?.author ?? '';

  return (
    <TouchableOpacity
      activeOpacity={0.9}
      onPress={() =>
        router.push(`/player?bookId=${state.currentBookId}`)
      }
      style={[styles.container, { backgroundColor: colors.miniPlayerBg }]}
    >
      <Image
        source={require('../../assets/images/icon.png')}
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
        <TouchableOpacity onPress={togglePlayback} hitSlop={8}>
          <MaterialIcons
            name={state.isPlaying ? 'pause' : 'play-arrow'}
            size={28}
            color={colors.text}
          />
        </TouchableOpacity>
        <TouchableOpacity
          hitSlop={8}
          style={{ marginLeft: 16 }}
          onPress={() => {
            if (state.currentSegmentIndex > 0) {
              seekToSegment(state.currentSegmentIndex - 1);
            }
          }}
        >
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
