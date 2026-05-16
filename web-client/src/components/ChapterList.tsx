import React, { useMemo } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  FlatList,
  StyleSheet,
  Modal,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import Colors from '../../constants/Colors';
import { useColorScheme } from '../hooks/useColorScheme';
import type { Segment } from '../types/api';

interface ChapterListProps {
  visible: boolean;
  onClose: () => void;
  segments: Segment[];
  currentSegmentIndex: number;
  onSelectSegment: (index: number) => void;
}

interface Chapter {
  name: string;
  firstSegmentIndex: number;
  segmentCount: number;
}

export function ChapterList({
  visible,
  onClose,
  segments,
  currentSegmentIndex,
  onSelectSegment,
}: ChapterListProps) {
  const theme = useColorScheme();
  const colors = Colors[theme];

  const chapters = useMemo(() => {
    if (!segments?.length) return [];

    const chapterMap: Chapter[] = [];
    let currentChapter = '';

    segments.forEach((seg, idx) => {
      const chapterName =
        seg.toc_path?.length > 0
          ? seg.toc_path[seg.toc_path.length - 1]
          : seg.chapter || `Section ${chapterMap.length + 1}`;

      if (chapterName !== currentChapter) {
        currentChapter = chapterName;
        chapterMap.push({
          name: chapterName,
          firstSegmentIndex: idx,
          segmentCount: 1,
        });
      } else if (chapterMap.length > 0) {
        chapterMap[chapterMap.length - 1].segmentCount++;
      }
    });

    return chapterMap;
  }, [segments]);

  // Find which chapter is currently active
  const activeChapterIdx = useMemo(() => {
    for (let i = chapters.length - 1; i >= 0; i--) {
      if (currentSegmentIndex >= chapters[i].firstSegmentIndex) {
        return i;
      }
    }
    return 0;
  }, [chapters, currentSegmentIndex]);

  const renderChapter = ({
    item,
    index,
  }: {
    item: Chapter;
    index: number;
  }) => {
    const isActive = index === activeChapterIdx;

    return (
      <TouchableOpacity
        activeOpacity={0.7}
        onPress={() => {
          onSelectSegment(item.firstSegmentIndex);
          onClose();
        }}
        style={[
          styles.chapterRow,
          { borderBottomColor: colors.border },
          isActive && { backgroundColor: `${colors.accent}15` },
        ]}
      >
        <View style={styles.chapterNumber}>
          <Text
            style={[
              styles.chapterIdx,
              { color: isActive ? colors.accent : colors.textMuted },
            ]}
          >
            {index + 1}
          </Text>
        </View>
        <View style={styles.chapterInfo}>
          <Text
            style={[
              styles.chapterName,
              {
                color: isActive ? colors.accent : colors.text,
                fontWeight: isActive ? '700' : '500',
              },
            ]}
            numberOfLines={2}
          >
            {item.name}
          </Text>
          <Text style={[styles.chapterMeta, { color: colors.textMuted }]}>
            {item.segmentCount} segment{item.segmentCount !== 1 ? 's' : ''}
          </Text>
        </View>
        {isActive && (
          <MaterialIcons name="graphic-eq" size={20} color={colors.accent} />
        )}
      </TouchableOpacity>
    );
  };

  return (
    <Modal
      visible={visible}
      animationType="slide"
      transparent
      onRequestClose={onClose}
    >
      <View style={styles.overlay}>
        <TouchableOpacity
          style={styles.backdrop}
          activeOpacity={1}
          onPress={onClose}
        />
        <View
          style={[styles.sheet, { backgroundColor: colors.surface }]}
        >
          {/* Header */}
          <View style={styles.sheetHeader}>
            <View style={[styles.handle, { backgroundColor: colors.textMuted }]} />
            <View style={styles.sheetTitleRow}>
              <Text style={[styles.sheetTitle, { color: colors.text }]}>
                Chapters
              </Text>
              <TouchableOpacity onPress={onClose}>
                <MaterialIcons name="close" size={24} color={colors.textMuted} />
              </TouchableOpacity>
            </View>
          </View>

          {/* Chapter list */}
          <FlatList
            data={chapters}
            keyExtractor={(_, idx) => String(idx)}
            renderItem={renderChapter}
            contentContainerStyle={styles.listContent}
            showsVerticalScrollIndicator={false}
            initialScrollIndex={
              activeChapterIdx > 2 ? activeChapterIdx - 1 : 0
            }
            getItemLayout={(_, index) => ({
              length: 72,
              offset: 72 * index,
              index,
            })}
          />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  backdrop: {
    flex: 1,
  },
  sheet: {
    maxHeight: '70%',
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    paddingBottom: 32,
  },
  sheetHeader: {
    alignItems: 'center',
    paddingTop: 12,
    paddingHorizontal: 20,
    paddingBottom: 4,
  },
  handle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    marginBottom: 16,
    opacity: 0.4,
  },
  sheetTitleRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    width: '100%',
    marginBottom: 8,
  },
  sheetTitle: {
    fontSize: 20,
    fontWeight: '700',
  },
  listContent: {
    paddingHorizontal: 20,
  },
  chapterRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: 16,
    height: 72,
  },
  chapterNumber: {
    width: 32,
    alignItems: 'center',
  },
  chapterIdx: {
    fontSize: 16,
    fontWeight: '700',
  },
  chapterInfo: {
    flex: 1,
  },
  chapterName: {
    fontSize: 15,
    marginBottom: 2,
  },
  chapterMeta: {
    fontSize: 12,
  },
});
