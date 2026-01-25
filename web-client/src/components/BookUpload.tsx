import { useState } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { useUploadBook } from '../api/hooks'

const Container = styled(YStack, {
  padding: 24,
  gap: 16,
  borderWidth: 2,
  borderStyle: 'dashed',
  borderRadius: 8,
  backgroundColor: '$background',
})

export function BookUpload({ onSuccess }: { onSuccess?: (bookId: string) => void }) {
  const [file, setFile] = useState<File | null>(null)
  const [title, setTitle] = useState('')
  const [author, setAuthor] = useState('')
  const [language, setLanguage] = useState('en')
  const uploadMutation = useUploadBook()

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0])
      // Auto-populate title from filename if not set
      if (!title) {
        const name = e.target.files[0].name.replace(/\.[^/.]+$/, '')
        setTitle(name)
      }
    }
  }

  const handleUpload = async () => {
    if (!file) return

    try {
      const result = await uploadMutation.mutateAsync({
        file,
        metadata: {
          title: title || file.name,
          author,
          language,
        },
      })
      setFile(null)
      setTitle('')
      setAuthor('')
      onSuccess?.(result.id)
    } catch (error) {
      console.error('Upload failed:', error)
    }
  }

  return (
    <Container>
      <Text fontSize={20} fontWeight="bold">
        Upload a Book
      </Text>

      <YStack gap={12}>
        <YStack gap={4}>
          <Text fontSize={14}>Title</Text>
          <input
            type="text"
            value={title}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}
            placeholder="Book title"
            style={{
              padding: '8px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              fontSize: '14px',
              width: '100%',
            }}
          />
        </YStack>

        <YStack gap={4}>
          <Text fontSize={14}>Author</Text>
          <input
            type="text"
            value={author}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setAuthor(e.target.value)}
            placeholder="Author name"
            style={{
              padding: '8px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              fontSize: '14px',
              width: '100%',
            }}
          />
        </YStack>

        <YStack gap={4}>
          <Text fontSize={14}>Language</Text>
          <input
            type="text"
            value={language}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLanguage(e.target.value)}
            placeholder="en"
            style={{
              padding: '8px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              fontSize: '14px',
              width: '100%',
            }}
          />
        </YStack>

        <YStack gap={4}>
          <Text fontSize={14}>File</Text>
          <label
            style={{
              padding: '12px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              cursor: 'pointer',
              textAlign: 'center',
              backgroundColor: '#f5f5f5',
            }}
          >
            <input
              type="file"
              accept=".txt,.pdf,.epub"
              onChange={handleFileChange}
              style={{ display: 'none' }}
            />
            {file ? file.name : 'Choose a file (TXT, PDF, or ePUB)'}
          </label>
        </YStack>
      </YStack>

      <XStack gap={12}>
        <Button
          onPress={handleUpload}
          disabled={!file || uploadMutation.isPending}
          backgroundColor="$primary"
          color="white"
        >
          {uploadMutation.isPending ? 'Uploading...' : 'Upload Book'}
        </Button>
        {file && (
          <Button
            onPress={() => setFile(null)}
            backgroundColor="$secondary"
            color="white"
          >
            Clear
          </Button>
        )}
      </XStack>

      {uploadMutation.isError && (
        <Text color="$error" fontSize={14}>
          Error: {uploadMutation.error?.message}
        </Text>
      )}

      {uploadMutation.isSuccess && (
        <Text color="$success" fontSize={14}>
          Book uploaded successfully!
        </Text>
      )}
    </Container>
  )
}
