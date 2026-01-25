import { styled } from '@tamagui/core'
import { YStack } from '@tamagui/stacks'
import { Text } from '../tamagui.config'
import { useBookStatus } from '../api/hooks'
import type { BookStatus } from '../types/api'

const Container = styled(YStack, {
  padding: 16,
  gap: 12,
  borderWidth: 1,
  borderRadius: 8,
  borderColor: '#e0e0e0',
  backgroundColor: '$background',
})

const ProgressBar = styled(YStack, {
  width: '100%',
  height: 8,
  backgroundColor: '#e0e0e0',
  borderRadius: 4,
  overflow: 'hidden',
})

const ProgressFill = styled(YStack, {
  height: '100%',
  backgroundColor: '$primary',
})

function getStatusColor(status: BookStatus): string {
  switch (status) {
    case 'uploaded':
    case 'parsing':
    case 'segmenting':
    case 'synthesizing':
      return '#ff9500'
    case 'voice_mapping':
    case 'ready':
      return '#007aff'
    case 'synthesized':
      return '#34c759'
    case 'error':
    case 'synthesis_error':
      return '#ff3b30'
    default:
      return '#8e8e93'
  }
}

function getStatusText(status: BookStatus): string {
  switch (status) {
    case 'uploaded':
      return 'Uploaded'
    case 'parsing':
      return 'Parsing book...'
    case 'segmenting':
      return 'Segmenting with LLM...'
    case 'voice_mapping':
      return 'Awaiting voice mapping'
    case 'ready':
      return 'Ready for synthesis'
    case 'synthesizing':
      return 'Synthesizing audio...'
    case 'synthesized':
      return 'Complete'
    case 'synthesis_error':
      return 'Synthesis failed'
    case 'error':
      return 'Error occurred'
    default:
      return status
  }
}

export function BookStatusCard({ bookId }: { bookId: string }) {
  const { data: status, isLoading, error } = useBookStatus(bookId)

  if (isLoading) {
    return (
      <Container>
        <Text>Loading status...</Text>
      </Container>
    )
  }

  if (error) {
    return (
      <Container>
        <Text color="$error">Error loading status: {error.message}</Text>
      </Container>
    )
  }

  if (!status) return null

  return (
    <Container>
      <YStack gap={4}>
        <Text fontSize={16} fontWeight="bold">
          Processing Status
        </Text>
        <Text fontSize={14} color={getStatusColor(status.status)}>
          {getStatusText(status.status)}
        </Text>
      </YStack>

      {status.progress > 0 && (
        <YStack gap={4}>
          <ProgressBar>
            <ProgressFill flexBasis={`${status.progress}%`} />
          </ProgressBar>
          <Text fontSize={12} color="#8e8e93">
            {status.progress.toFixed(1)}% complete
          </Text>
        </YStack>
      )}

      <YStack gap={2}>
        <Text fontSize={12}>
          Chapters: {status.parsed_chapters} / {status.total_chapters}
        </Text>
        <Text fontSize={12}>Segments: {status.total_segments}</Text>
      </YStack>
    </Container>
  )
}
