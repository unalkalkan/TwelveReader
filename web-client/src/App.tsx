import { useEffect, useState } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from './tamagui.config'
import { BookUpload } from './components/BookUpload'
import { BookPlayer } from './components/BookPlayer'
import { UnifiedProgressView } from './components/UnifiedProgressView'
import { VoiceMappingDialog } from './components/VoiceMappingDialog'
import { RunList } from './components/RunList'
import { useServerInfo, usePersonas } from './api/hooks'
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

type View = 'upload' | 'processing' | 'player'

const loadFromStorage = <T,>(key: string): T | null => {
  if (typeof window === 'undefined') return null
  const value = window.localStorage.getItem(key)
  return value ? (value as T) : null
}

function App() {
  const [currentView, setCurrentView] = useState<View>(() => loadFromStorage<View>('tr:currentView') ?? 'upload')
  const [currentBookId, setCurrentBookId] = useState<string | null>(() => loadFromStorage<string>('tr:currentBookId'))
  const { data: serverInfo } = useServerInfo()
  const { data: personas } = usePersonas(currentBookId ?? undefined)

  const handleUploadSuccess = (bookId: string) => {
    setCurrentBookId(bookId)
    setCurrentView('processing')
  }

  const handleJoinRun = (bookId: string) => {
    setCurrentBookId(bookId)
    setCurrentView('processing')
  }

  // Persist selections so a page refresh keeps the active book/pipeline context
  useEffect(() => {
    if (typeof window === 'undefined') return
    if (currentBookId) {
      window.localStorage.setItem('tr:currentBookId', currentBookId)
    } else {
      window.localStorage.removeItem('tr:currentBookId')
    }
  }, [currentBookId])

  useEffect(() => {
    if (typeof window === 'undefined') return
    window.localStorage.setItem('tr:currentView', currentView)
  }, [currentView])

  const handleVoiceMappingComplete = () => {
    // Voice mapping complete - user can continue
    // Don't automatically switch views
  }

  // Show voice mapping dialog if there are unmapped personas
  const showVoiceDialog = currentBookId && personas && personas.unmapped.length > 0

  return (
    <Container>
      <Header>
        <Text fontSize={32} fontWeight="bold">
          Twelve Reader - Hybrid Pipeline
        </Text>
        <Text fontSize={16} color="#8e8e93">
          Upload, process incrementally, and play audiobooks with instant voice mapping
        </Text>
        {serverInfo && (
          <Text fontSize={12} color="#8e8e93">
            Server v{serverInfo.version} | Storage: {serverInfo.storage_adapter}
          </Text>
        )}
      </Header>

      <Content>
        <XStack gap={12} flexWrap="wrap">
          <Button
            onPress={() => setCurrentView('upload')}
            backgroundColor={currentView === 'upload' ? '$primary' : '$secondary'}
            color="white"
          >
            Upload Book
          </Button>
          <Button
            onPress={() => setCurrentView('processing')}
            disabled={!currentBookId}
            backgroundColor={currentView === 'processing' ? '$primary' : '$secondary'}
            color="white"
          >
            View Progress
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
          <RunList activeBookId={currentBookId} onSelect={handleJoinRun} />
        </Card>

        <Card>
          {currentView === 'upload' && (
            <YStack gap={16}>
              <Text fontSize={24} fontWeight="bold">
                Upload a Book
              </Text>
              <BookUpload onSuccess={handleUploadSuccess} />
            </YStack>
          )}

          {currentView === 'processing' && currentBookId && (
            <YStack gap={16}>
              <UnifiedProgressView bookId={currentBookId} />
            </YStack>
          )}

          {currentView === 'player' && currentBookId && (
            <YStack gap={16}>
              <BookPlayer bookId={currentBookId} />
            </YStack>
          )}
        </Card>

        <Card>
          <YStack gap={12}>
            <Text fontSize={20} fontWeight="bold">
              About Hybrid Pipeline
            </Text>
            <Text fontSize={14} lineHeight={20}>
              The hybrid pipeline enables instant playback by processing books incrementally:
            </Text>
            <YStack gap={8} paddingLeft={16}>
              <Text fontSize={14} lineHeight={20}>
                1. Upload your book (TXT, PDF, or ePUB)
              </Text>
              <Text fontSize={14} lineHeight={20}>
                2. LLM segments first 5 paragraphs → Pause for voice mapping
              </Text>
              <Text fontSize={14} lineHeight={20}>
                3. Continue segmentation + TTS synthesis in parallel
              </Text>
              <Text fontSize={14} lineHeight={20}>
                4. New personas discovered? Map voices incrementally
              </Text>
              <Text fontSize={14} lineHeight={20}>
                5. Start listening while synthesis continues!
              </Text>
            </YStack>
            <Text fontSize={14} lineHeight={20} color="#8e8e93">
              Tech Stack: React + TypeScript + Tamagui + TanStack Query + Go Backend
            </Text>
          </YStack>
        </Card>
      </Content>

      {/* Floating Voice Mapping Dialog - shows automatically when unmapped personas exist */}
      {showVoiceDialog && (
        <VoiceMappingDialog bookId={currentBookId} onComplete={handleVoiceMappingComplete} />
      )}
    </Container>
  )
}

export default App
