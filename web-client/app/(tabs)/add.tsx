import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Alert,
  Platform,
  ActivityIndicator,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { SafeAreaView } from 'react-native-safe-area-context';
import * as DocumentPicker from 'expo-document-picker';
import { useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { useUploadBook } from '../../src/api/hooks';

export default function AddScreen() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();
  const uploadMutation = useUploadBook();
  const [uploading, setUploading] = useState(false);

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
      setUploading(true);

      const book = await uploadMutation.mutateAsync({
        fileUri: file.uri,
        fileName: file.name,
        mimeType: file.mimeType ?? 'application/octet-stream',
      });

      setUploading(false);
      // Navigate to library or player after upload
      router.push(`/player?bookId=${book.id}`);
    } catch (err: any) {
      setUploading(false);
      Alert.alert('Upload failed', err.message || 'Something went wrong');
    }
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

        <View style={styles.grid}>
          {/* Upload file */}
          <TouchableOpacity
            activeOpacity={0.7}
            onPress={handleFilePick}
            disabled={uploading}
            style={[styles.card, { backgroundColor: colors.card }]}
          >
            {uploading ? (
              <ActivityIndicator size="large" color={colors.accent} />
            ) : (
              <>
                <MaterialIcons
                  name="file-upload"
                  size={40}
                  color={colors.textMuted}
                />
                <Text style={[styles.cardLabel, { color: colors.text }]}>
                  Upload a{'\n'}file
                </Text>
                <Text style={[styles.cardHint, { color: colors.textMuted }]}>
                  PDF, ePUB, TXT
                </Text>
              </>
            )}
          </TouchableOpacity>

          {/* Write text */}
          <TouchableOpacity
            activeOpacity={0.7}
            style={[styles.card, { backgroundColor: colors.card }]}
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
            style={[styles.card, { backgroundColor: colors.card }]}
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

        <View style={styles.infoSection}>
          <MaterialIcons name="info-outline" size={20} color={colors.textMuted} />
          <Text style={[styles.infoText, { color: colors.textSecondary }]}>
            The hybrid pipeline processes your book incrementally — you can
            start listening while synthesis continues in the background.
          </Text>
        </View>
      </ScrollView>
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
});
