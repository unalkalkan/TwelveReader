import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Alert,
  TextInput,
  Modal,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import * as DocumentPicker from 'expo-document-picker';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useUploadBookWithProgress } from '../../src/api/hooks';

type ActiveModal = null | 'metadata' | 'text' | 'link';

export default function AddScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const { mutateAsync, progress, isPending } = useUploadBookWithProgress();

  const [activeModal, setActiveModal] = useState<ActiveModal>(null);

  // Metadata form state
  const [title, setTitle] = useState('');
  const [author, setAuthor] = useState('');
  const [language, setLanguage] = useState('en');
  const [pendingFile, setPendingFile] = useState<{
    uri: string;
    name: string;
    mimeType: string;
  } | null>(null);

  // Text input state
  const [textContent, setTextContent] = useState('');
  const [textTitle, setTextTitle] = useState('');

  // Link input state
  const [linkUrl, setLinkUrl] = useState('');
  const [linkTitle, setLinkTitle] = useState('');

  const handleFilePick = async () => {
    try {
      const result = await DocumentPicker.getDocumentAsync({
        type: [
          'application/pdf',
          'application/epub+zip',
          'text/plain',
        ],
        copyToCacheDirectory: true,
      });

      if (result.canceled || !result.assets?.[0]) return;

      const file = result.assets[0];
      // Pre-fill title from filename
      const nameWithoutExt = file.name.replace(/\.[^/.]+$/, '');
      setTitle(nameWithoutExt);
      setAuthor('');
      setLanguage('en');
      setPendingFile({
        uri: file.uri,
        name: file.name,
        mimeType: file.mimeType ?? 'application/octet-stream',
      });
      setActiveModal('metadata');
    } catch (err: any) {
      Alert.alert('Error', err.message || 'Failed to pick file');
    }
  };

  const handleUpload = async () => {
    if (!pendingFile) return;
    setActiveModal(null);

    try {
      const book = await mutateAsync({
        fileSource: {
          uri: pendingFile.uri,
          name: pendingFile.name,
          type: pendingFile.mimeType,
        },
        metadata: {
          title: title.trim() || undefined,
          author: author.trim() || undefined,
          language: language.trim() || undefined,
        },
      });
      setPendingFile(null);
      router.push(`/player?bookId=${book.id}`);
    } catch (err: any) {
      Alert.alert('Upload failed', err.message || 'Something went wrong');
    }
  };

  const handleTextSubmit = async () => {
    if (!textContent.trim()) {
      Alert.alert('Error', 'Please enter some text');
      return;
    }

    if (Platform.OS !== 'web') {
      Alert.alert(
        'Not Available',
        'Typed text upload is currently web-only. Native temp-file support is coming soon.',
      );
      return;
    }

    setActiveModal(null);

    try {
      const blob = new Blob([textContent], { type: 'text/plain' });
      const fileName = `${textTitle.trim() || 'text-input'}.txt`;

      const book = await mutateAsync({
        fileSource: {
          blob,
          name: fileName,
          type: 'text/plain',
        },
        metadata: {
          title: textTitle.trim() || 'Text Input',
          author: author.trim() || undefined,
          language: 'en',
        },
      });
      setTextContent('');
      setTextTitle('');
      router.push(`/player?bookId=${book.id}`);
    } catch (err: any) {
      Alert.alert('Upload failed', err.message || 'Something went wrong');
    }
  };

  const handleLinkSubmit = async () => {
    if (!linkUrl.trim()) {
      Alert.alert('Error', 'Please enter a URL');
      return;
    }
    // For now, create a text file with the URL content placeholder
    Alert.alert(
      'Coming Soon',
      'URL import will be available in a future update. For now, please download the content and upload the file directly.',
    );
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={[styles.title, { color: colors.text }]}>
          Upload & Listen
        </Text>
        <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
          Upload a book and let AI create an immersive audiobook experience
        </Text>

        {/* Upload progress */}
        {isPending && (
          <View style={[styles.progressContainer, { backgroundColor: colors.card }]}>
            <View style={styles.progressHeader}>
              <MaterialIcons name="cloud-upload" size={20} color={colors.accent} />
              <Text style={[styles.progressLabel, { color: colors.text }]}>
                Uploading... {progress}%
              </Text>
            </View>
            <View style={[styles.progressTrack, { backgroundColor: colors.border }]}>
              <View
                style={[
                  styles.progressFill,
                  { width: `${progress}%`, backgroundColor: colors.accent },
                ]}
              />
            </View>
          </View>
        )}

        <View style={styles.grid}>
          {/* Upload file */}
          <TouchableOpacity
            activeOpacity={0.7}
            onPress={handleFilePick}
            disabled={isPending}
            style={[
              styles.card,
              { backgroundColor: colors.card },
              isPending && { opacity: 0.5 },
            ]}
          >
            <MaterialIcons
              name="file-upload"
              size={40}
              color={colors.accent}
            />
            <Text style={[styles.cardLabel, { color: colors.text }]}>
              Upload a{'\n'}file
            </Text>
            <Text style={[styles.cardHint, { color: colors.textMuted }]}>
              PDF, ePUB, TXT
            </Text>
          </TouchableOpacity>

          {/* Write text */}
          <TouchableOpacity
            activeOpacity={0.7}
            onPress={() => {
              setTextContent('');
              setTextTitle('');
              setActiveModal('text');
            }}
            disabled={isPending}
            style={[
              styles.card,
              { backgroundColor: colors.card },
              isPending && { opacity: 0.5 },
            ]}
          >
            <MaterialIcons
              name="text-fields"
              size={40}
              color={colors.textMuted}
            />
            <Text style={[styles.cardLabel, { color: colors.text }]}>
              Write{'\n'}text
            </Text>
            <Text style={[styles.cardHint, { color: colors.textMuted }]}>
              Paste or type
            </Text>
          </TouchableOpacity>

          {/* Scan text */}
          <TouchableOpacity
            activeOpacity={0.7}
            onPress={() =>
              Alert.alert('Coming Soon', 'Camera scanning will be available in a future update.')
            }
            style={[styles.card, { backgroundColor: colors.card }]}
          >
            <MaterialIcons
              name="document-scanner"
              size={40}
              color={colors.textMuted}
            />
            <Text style={[styles.cardLabel, { color: colors.text }]}>
              Scan{'\n'}text
            </Text>
            <Text style={[styles.cardHint, { color: colors.textMuted }]}>
              Use camera
            </Text>
          </TouchableOpacity>

          {/* Paste link */}
          <TouchableOpacity
            activeOpacity={0.7}
            onPress={() => {
              setLinkUrl('');
              setLinkTitle('');
              setActiveModal('link');
            }}
            disabled={isPending}
            style={[
              styles.card,
              { backgroundColor: colors.card },
              isPending && { opacity: 0.5 },
            ]}
          >
            <MaterialIcons name="link" size={40} color={colors.textMuted} />
            <Text style={[styles.cardLabel, { color: colors.text }]}>
              Paste a{'\n'}link
            </Text>
            <Text style={[styles.cardHint, { color: colors.textMuted }]}>
              From URL
            </Text>
          </TouchableOpacity>
        </View>

        <View style={[styles.infoSection, { backgroundColor: colors.card }]}>
          <MaterialIcons name="info-outline" size={20} color={colors.textMuted} />
          <Text style={[styles.infoText, { color: colors.textSecondary }]}>
            The hybrid pipeline processes your book incrementally — you can
            start listening while synthesis continues in the background.
          </Text>
        </View>
      </ScrollView>

      {/* ─── Metadata Modal ─── */}
      <Modal visible={activeModal === 'metadata'} animationType="slide" transparent>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.modalOverlay}
        >
          <TouchableOpacity
            style={styles.modalBackdrop}
            activeOpacity={1}
            onPress={() => setActiveModal(null)}
          />
          <View style={[styles.modalSheet, { backgroundColor: colors.surface }]}>
            <View style={[styles.modalHandle, { backgroundColor: colors.textMuted }]} />
            <Text style={[styles.modalTitle, { color: colors.text }]}>
              Book Details
            </Text>
            <Text style={[styles.modalSubtitle, { color: colors.textMuted }]}>
              {pendingFile?.name}
            </Text>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Title</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={title}
                onChangeText={setTitle}
                placeholder="Book title"
                placeholderTextColor={colors.textMuted}
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Author</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={author}
                onChangeText={setAuthor}
                placeholder="Author name"
                placeholderTextColor={colors.textMuted}
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Language</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={language}
                onChangeText={setLanguage}
                placeholder="en"
                placeholderTextColor={colors.textMuted}
              />
            </View>

            <TouchableOpacity
              onPress={handleUpload}
              style={[styles.submitBtn, { backgroundColor: colors.accent }]}
            >
              <MaterialIcons name="cloud-upload" size={20} color="#FFF" />
              <Text style={styles.submitBtnText}>Upload Book</Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </Modal>

      {/* ─── Text Input Modal ─── */}
      <Modal visible={activeModal === 'text'} animationType="slide" transparent>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.modalOverlay}
        >
          <TouchableOpacity
            style={styles.modalBackdrop}
            activeOpacity={1}
            onPress={() => setActiveModal(null)}
          />
          <View style={[styles.modalSheet, { backgroundColor: colors.surface, maxHeight: '85%' }]}>
            <View style={[styles.modalHandle, { backgroundColor: colors.textMuted }]} />
            <Text style={[styles.modalTitle, { color: colors.text }]}>
              Write or Paste Text
            </Text>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Title</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={textTitle}
                onChangeText={setTextTitle}
                placeholder="Give it a title"
                placeholderTextColor={colors.textMuted}
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Content</Text>
              <TextInput
                style={[
                  styles.formInput,
                  styles.textArea,
                  { color: colors.text, backgroundColor: colors.card, borderColor: colors.border },
                ]}
                value={textContent}
                onChangeText={setTextContent}
                placeholder="Paste or type your text here..."
                placeholderTextColor={colors.textMuted}
                multiline
                textAlignVertical="top"
              />
            </View>

            <TouchableOpacity
              onPress={handleTextSubmit}
              disabled={!textContent.trim()}
              style={[
                styles.submitBtn,
                { backgroundColor: textContent.trim() ? colors.accent : colors.card },
              ]}
            >
              <MaterialIcons name="cloud-upload" size={20} color={textContent.trim() ? '#FFF' : colors.textMuted} />
              <Text style={[styles.submitBtnText, !textContent.trim() && { color: colors.textMuted }]}>
                Convert to Audio
              </Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </Modal>

      {/* ─── Link Input Modal ─── */}
      <Modal visible={activeModal === 'link'} animationType="slide" transparent>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          style={styles.modalOverlay}
        >
          <TouchableOpacity
            style={styles.modalBackdrop}
            activeOpacity={1}
            onPress={() => setActiveModal(null)}
          />
          <View style={[styles.modalSheet, { backgroundColor: colors.surface }]}>
            <View style={[styles.modalHandle, { backgroundColor: colors.textMuted }]} />
            <Text style={[styles.modalTitle, { color: colors.text }]}>
              Paste a Link
            </Text>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>Title (optional)</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={linkTitle}
                onChangeText={setLinkTitle}
                placeholder="Title for this content"
                placeholderTextColor={colors.textMuted}
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={[styles.formLabel, { color: colors.textSecondary }]}>URL</Text>
              <TextInput
                style={[styles.formInput, { color: colors.text, backgroundColor: colors.card, borderColor: colors.border }]}
                value={linkUrl}
                onChangeText={setLinkUrl}
                placeholder="https://..."
                placeholderTextColor={colors.textMuted}
                keyboardType="url"
                autoCapitalize="none"
              />
            </View>

            <TouchableOpacity
              onPress={handleLinkSubmit}
              disabled={!linkUrl.trim()}
              style={[
                styles.submitBtn,
                { backgroundColor: linkUrl.trim() ? colors.accent : colors.card },
              ]}
            >
              <MaterialIcons name="link" size={20} color={linkUrl.trim() ? '#FFF' : colors.textMuted} />
              <Text style={[styles.submitBtnText, !linkUrl.trim() && { color: colors.textMuted }]}>
                Import from URL
              </Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </Modal>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1 },
  content: {
    padding: 24,
    paddingBottom: 200,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    lineHeight: 22,
    marginBottom: 32,
  },
  // Progress
  progressContainer: {
    padding: 16,
    borderRadius: 12,
    marginBottom: 24,
  },
  progressHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 12,
  },
  progressLabel: {
    fontSize: 14,
    fontWeight: '600',
  },
  progressTrack: {
    height: 6,
    borderRadius: 3,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    borderRadius: 3,
  },
  // Grid
  grid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 16,
  },
  card: {
    width: '47%',
    aspectRatio: 1,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    padding: 16,
  },
  cardLabel: {
    fontSize: 16,
    fontWeight: '600',
    textAlign: 'center',
    lineHeight: 20,
  },
  cardHint: {
    fontSize: 12,
    textAlign: 'center',
  },
  infoSection: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 12,
    marginTop: 32,
    padding: 16,
    borderRadius: 12,
  },
  infoText: {
    flex: 1,
    fontSize: 14,
    lineHeight: 20,
  },
  // Modal
  modalOverlay: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  modalBackdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
  },
  modalSheet: {
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 24,
    paddingTop: 12,
  },
  modalHandle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    alignSelf: 'center',
    marginBottom: 20,
    opacity: 0.4,
  },
  modalTitle: {
    fontSize: 22,
    fontWeight: '700',
    marginBottom: 4,
  },
  modalSubtitle: {
    fontSize: 13,
    marginBottom: 20,
  },
  // Form
  formGroup: {
    marginBottom: 16,
  },
  formLabel: {
    fontSize: 13,
    fontWeight: '600',
    marginBottom: 6,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  formInput: {
    fontSize: 16,
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderRadius: 12,
    borderWidth: 1,
  },
  textArea: {
    height: 200,
    textAlignVertical: 'top',
    paddingTop: 12,
  },
  submitBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    paddingVertical: 16,
    borderRadius: 28,
    marginTop: 8,
  },
  submitBtnText: {
    color: '#FFF',
    fontSize: 16,
    fontWeight: '700',
  },
});
