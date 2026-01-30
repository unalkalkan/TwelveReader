import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Text } from '../tamagui.config'
import { usePipelineStatus, useBookStatus } from '../api/hooks'
import type { StageProgress } from '../types/api'

const Container = styled(YStack, {
  gap: 24,
  padding: 16,
})

const StageContainer = styled(YStack, {
  gap: 8,
  padding: 16,
  borderRadius: 8,
  backgroundColor: '$backgroundHover',
})

const StatusBadge = styled(XStack, {
  paddingHorizontal: 12,
  paddingVertical: 4,
  borderRadius: 12,
  alignSelf: 'flex-start',
})

const ProgressBar = styled(YStack, {
  height: 24,
  backgroundColor: '#e0e0e0',
  borderRadius: 12,
  overflow: 'hidden',
})

const ProgressFill = styled(YStack, {
  height: '100%',
  backgroundColor: '$primary',
  borderRadius: 12,
  transition: 'width 0.3s ease',
})

interface UnifiedProgressViewProps {
  bookId: string
}

export function UnifiedProgressView({ bookId }: UnifiedProgressViewProps) {
  const { data: pipelineStatus, isLoading } = usePipelineStatus(bookId)
  const { data: bookStatus } = useBookStatus(bookId)

  if (isLoading) {
    return (
      <Container>
        <Text>Loading pipeline status...</Text>
      </Container>
    )
  }

  // Extract stages from pipeline status
  const segmentingStage = pipelineStatus?.stages.find((s) => s.stage === 'segmenting')
  const synthesizingStage = pipelineStatus?.stages.find((s) => s.stage === 'synthesizing')
  const readyStage = pipelineStatus?.stages.find((s) => s.stage === 'ready')

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return '#4CAF50'
      case 'in_progress':
        return '#2196F3'
      case 'waiting_for_mapping':
        return '#FF9800'
      case 'error':
        return '#F44336'
      default:
        return '#9E9E9E'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'completed':
        return 'Completed'
      case 'in_progress':
        return 'In Progress'
      case 'waiting_for_mapping':
        return 'Waiting for Mapping'
      case 'error':
        return 'Error'
      case 'pending':
        return 'Pending'
      default:
        return status
    }
  }

  const renderStage = (stage: StageProgress | undefined, title: string) => {
    if (!stage) return null

    const percentage = stage.total > 0 ? (stage.current / stage.total) * 100 : 0

    return (
      <StageContainer>
        <XStack justifyContent="space-between" alignItems="center">
          <Text fontSize={18} fontWeight="600">
            {title}
          </Text>
          <StatusBadge backgroundColor={getStatusColor(stage.status)}>
            <Text fontSize={12} color="white" fontWeight="500">
              {getStatusLabel(stage.status)}
            </Text>
          </StatusBadge>
        </XStack>

        <ProgressBar>
          <ProgressFill width={`${percentage}%`} />
        </ProgressBar>

        <XStack justifyContent="space-between">
          <Text fontSize={14} color="#666">
            {stage.message}
          </Text>
          <Text fontSize={14} fontWeight="500">
            {stage.current} / {stage.total > 0 ? stage.total : '?'}
          </Text>
        </XStack>

        {stage.total > 0 && (
          <Text fontSize={12} color="#999">
            {percentage.toFixed(1)}% complete
          </Text>
        )}
      </StageContainer>
    )
  }

  return (
    <Container>
      <YStack gap={12}>
        <Text fontSize={24} fontWeight="bold">
          Book Processing Progress
        </Text>
        <Text fontSize={14} color="#666">
          Current Status: {bookStatus?.status || 'unknown'}
        </Text>
      </YStack>

      {renderStage(segmentingStage, '📖 Segmenting')}
      {renderStage(synthesizingStage, '🎙️ Synthesizing')}
      {renderStage(readyStage, '✅ Ready for Playback')}

      {bookStatus?.status === 'voice_mapping' && (
        <StageContainer backgroundColor="#FFF3E0">
          <Text fontSize={16} fontWeight="600" color="#FF6F00">
            ⚠️ Voice Mapping Required
          </Text>
          <Text fontSize={14} color="#666">
            Please map voices to characters to continue synthesis.
          </Text>
        </StageContainer>
      )}

      {pipelineStatus?.updated_at && (
        <Text fontSize={12} color="#999">
          Last updated: {new Date(pipelineStatus.updated_at).toLocaleTimeString()}
        </Text>
      )}
    </Container>
  )
}
