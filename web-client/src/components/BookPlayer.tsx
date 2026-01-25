import { useState, useRef, useEffect } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { useBookSegments } from '../api/hooks'
import { getAudioUrl } from '../api/client'

const Container = styled(YStack, {
  padding: 24,
  gap: 16,
  borderWidth: 1,
  borderRadius: 8,
  borderColor: '#e0e0e0',
  backgroundColor: '$background',
})

const SegmentText = styled(Text, {
  padding: 12,
  borderRadius: 8,
  fontSize: 16,
  lineHeight: 24,
})

export function BookPlayer({ bookId }: { bookId: string }) {
  const { data: segments, isLoading, error } = useBookSegments(bookId)
  const [currentSegmentIndex, setCurrentSegmentIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const audioRef = useRef<HTMLAudioElement>(null)

  const currentSegment = segments?.[currentSegmentIndex]

  useEffect(() => {
    if (audioRef.current && currentSegment) {
      audioRef.current.src = getAudioUrl(bookId, currentSegment.id)
      if (isPlaying) {
        audioRef.current.play()
      }
    }
  }, [currentSegmentIndex, currentSegment, bookId, isPlaying])

  const handlePlayPause = () => {
    if (!audioRef.current) return

    if (isPlaying) {
      audioRef.current.pause()
      setIsPlaying(false)
    } else {
      audioRef.current.play()
      setIsPlaying(true)
    }
  }

  const handleNext = () => {
    if (segments && currentSegmentIndex < segments.length - 1) {
      setCurrentSegmentIndex(currentSegmentIndex + 1)
    }
  }

  const handlePrevious = () => {
    if (currentSegmentIndex > 0) {
      setCurrentSegmentIndex(currentSegmentIndex - 1)
    }
  }

  const handleEnded = () => {
    if (segments && currentSegmentIndex < segments.length - 1) {
      setCurrentSegmentIndex(currentSegmentIndex + 1)
    } else {
      setIsPlaying(false)
    }
  }

  if (isLoading) {
    return (
      <Container>
        <Text>Loading segments...</Text>
      </Container>
    )
  }

  if (error) {
    return (
      <Container>
        <Text color="$error">Error loading segments: {error.message}</Text>
      </Container>
    )
  }

  if (!segments || segments.length === 0) {
    return (
      <Container>
        <Text>No segments available for playback</Text>
      </Container>
    )
  }

  return (
    <Container>
      <audio
        ref={audioRef}
        onEnded={handleEnded}
        onError={() => setIsPlaying(false)}
      />

      <Text fontSize={20} fontWeight="bold">
        Book Player
      </Text>

      <Text fontSize={14} color="#8e8e93">
        Segment {currentSegmentIndex + 1} of {segments.length}
      </Text>

      {currentSegment && (
        <SegmentText backgroundColor={isPlaying ? '#f0f8ff' : '#f5f5f5'}>
          {currentSegment.text}
        </SegmentText>
      )}

      {currentSegment && (
        <YStack gap={4}>
          <Text fontSize={12} color="#8e8e93">
            Speaker: {currentSegment.person}
          </Text>
          <Text fontSize={12} color="#8e8e93">
            Voice: {currentSegment.voice_description}
          </Text>
        </YStack>
      )}

      <XStack gap={12} justifyContent="center">
        <Button
          onPress={handlePrevious}
          disabled={currentSegmentIndex === 0}
          backgroundColor="$secondary"
          color="white"
        >
          Previous
        </Button>
        <Button
          onPress={handlePlayPause}
          backgroundColor="$primary"
          color="white"
        >
          {isPlaying ? 'Pause' : 'Play'}
        </Button>
        <Button
          onPress={handleNext}
          disabled={currentSegmentIndex === segments.length - 1}
          backgroundColor="$secondary"
          color="white"
        >
          Next
        </Button>
      </XStack>
    </Container>
  )
}
