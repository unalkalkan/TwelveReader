import { useState } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from './tamagui.config'
import { BookUpload } from './components/BookUpload'
import { BookStatusCard } from './components/BookStatusCard'
import { BookPlayer } from './components/BookPlayer'
import { useServerInfo } from './api/hooks'
import './App.css'

const Container = styled(YStack, {
  minHeight: '100vh',
  backgroundColor: '#fafafa',
  padding: 24,
  gap: 24,
})

const Header = styled(YStack, {
  gap: 8,
  paddingBottom: 16,
  borderBottomWidth: 1,
  borderBottomColor: '#e0e0e0',
})

const Content = styled(YStack, {
  maxWidth: 1200,
  width: '100%',
  marginHorizontal: 'auto',
  gap: 24,
})

const Card = styled(YStack, {
  backgroundColor: 'white',
  padding: 24,
  borderRadius: 12,
  borderWidth: 1,
  borderColor: '#e0e0e0',
})

type View = 'upload' | 'status' | 'player'

function App() {
  const [currentView, setCurrentView] = useState<View>('upload')
  const [currentBookId, setCurrentBookId] = useState<string | null>(null)
  const { data: serverInfo } = useServerInfo()

  const handleUploadSuccess = (bookId: string) => {
    setCurrentBookId(bookId)
    setCurrentView('status')
  }

  return (
    <Container>
      <Header>
        <Text fontSize={32} fontWeight="bold">
          Twelve Reader - Web Client MVP
        </Text>
        <Text fontSize={16} color="#8e8e93">
          Upload, process, and play audiobooks with synchronized text
        </Text>
        {serverInfo && (
          <Text fontSize={12} color="#8e8e93">
            Server v{serverInfo.version} | Storage: {serverInfo.storage_adapter}
          </Text>
        )}
      </Header>

      <Content>
        <XStack gap={12}>
          <Button
            onPress={() => setCurrentView('upload')}
            backgroundColor={currentView === 'upload' ? '$primary' : '$secondary'}
            color="white"
          >
            Upload Book
          </Button>
          <Button
            onPress={() => setCurrentView('status')}
            disabled={!currentBookId}
            backgroundColor={currentView === 'status' ? '$primary' : '$secondary'}
            color="white"
          >
            View Status
          </Button>
          <Button
            onPress={() => setCurrentView('player')}
            disabled={!currentBookId}
            backgroundColor={currentView === 'player' ? '$primary' : '$secondary'}
            color="white"
          >
            Play Book
          </Button>
        </XStack>

        <Card>
          {currentView === 'upload' && (
            <BookUpload onSuccess={handleUploadSuccess} />
          )}

          {currentView === 'status' && currentBookId && (
            <YStack gap={16}>
              <Text fontSize={24} fontWeight="bold">
                Book Processing Status
              </Text>
              <BookStatusCard bookId={currentBookId} />
            </YStack>
          )}

          {currentView === 'player' && currentBookId && (
            <BookPlayer bookId={currentBookId} />
          )}
        </Card>

        <Card>
          <YStack gap={12}>
            <Text fontSize={20} fontWeight="bold">
              About Twelve Reader
            </Text>
            <Text fontSize={14} lineHeight={20}>
              Twelve Reader transforms static books into fully voiced, time-aligned
              experiences. Upload a book (TXT, PDF, or ePUB), let the LLM segment
              and annotate it, map voices to characters, and enjoy synchronized
              audio playback with text highlighting.
            </Text>
            <Text fontSize={14} lineHeight={20} color="#8e8e93">
              Tech Stack: React + TypeScript + Tamagui + TanStack Query + Zod
            </Text>
          </YStack>
        </Card>
      </Content>
    </Container>
  )
}

export default App

