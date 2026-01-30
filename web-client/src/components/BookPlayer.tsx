import { useState, useRef, useEffect } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { useBookSegments, usePersonas } from '../api/hooks'
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

const StatusBadge = styled(XStack, {
  paddingHorizontal: 8,
  paddingVertical: 4,
  borderRadius: 12,
  gap: 4,
  alignItems: 'center',
})

interface SegmentStatus {
  id: string
  status: 'ready' | 'synthesizing' | 'waiting' | 'error'
}

export function BookPlayer({ bookId }: { bookId: string }) {
  const { data: segments, isLoading, error, isRefetching } = useBookSegments(bookId)
  const { data: personas } = usePersonas(bookId)
  const [currentSegmentIndex, setCurrentSegmentIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [audioError, setAudioError] = useState(false)
  const audioRef = useRef<HTMLAudioElement>(null)

  const currentSegment = segments?.[currentSegmentIndex]

  // Determine segment status based on persona mapping
  const getSegmentStatus = (segment: typeof currentSegment): SegmentStatus['status'] => {
    if (!segment || !personas) return 'waiting'
    
    // Check if persona is mapped
    const isMapped = personas.mapped[segment.person] !== undefined
    
    if (!isMapped) {
      return 'waiting' // Waiting for voice mapping
    }
    
    // Check if segment has unmapped persona
    if (personas.unmapped.includes(segment.person)) {
      return 'waiting'
    }
    
    // Check if segment has been synthesized (has voice_id from TTS)
    if (!segment.voice_id) {
      return 'synthesizing' // Mapped but not yet synthesized
    }
    
    // If audio fails to load, mark as error
    if (audioError) {
      return 'error'
    }
    
    // Segment is mapped and synthesized - ready to play
    return 'ready'
  }

  const currentStatus = getSegmentStatus(currentSegment)

  useEffect(() => {
    if (audioRef.current && currentSegment && currentStatus === 'ready') {
      const newSrc = getAudioUrl(bookId, currentSegment.id)
      
      // Only update src if it has changed to avoid interrupting playback
      if (audioRef.current.src !== newSrc) {
        setAudioError(false)
        audioRef.current.src = newSrc
        
        // Restore playback state if audio was playing
        if (isPlaying) {
          audioRef.current.play().catch(() => {
            setAudioError(true)
            setIsPlaying(false)
          })
        }
      }
    } else if (currentStatus !== 'ready') {
      // Pause playback if segment not ready
      setIsPlaying(false)
    }
  }, [currentSegmentIndex, currentSegment, bookId, isPlaying, currentStatus])

  const handlePlayPause = () => {
    if (!audioRef.current || currentStatus !== 'ready') return

    if (isPlaying) {
      audioRef.current.pause()
      setIsPlaying(false)
    } else {
      audioRef.current.play().catch(() => {
        setAudioError(true)
        setIsPlaying(false)
      })
      setIsPlaying(true)
    }
  }

  const findNextReadySegment = (startIndex: number): number | null => {
    if (!segments) return null
    
    for (let i = startIndex; i < segments.length; i++) {
      const seg = segments[i]
      if (personas?.mapped[seg.person] !== undefined) {
        return i
      }
    }
    return null
  }

  const handleNext = () => {
    if (!segments) return
    
    // Reset audio error when changing segments
    setAudioError(false)
    
    // Try to find next ready segment
    const nextReady = findNextReadySegment(currentSegmentIndex + 1)
    
    if (nextReady !== null) {
      setCurrentSegmentIndex(nextReady)
    } else if (currentSegmentIndex < segments.length - 1) {
      // Just move to next even if not ready
      setCurrentSegmentIndex(currentSegmentIndex + 1)
    }
  }

  const handlePrevious = () => {
    if (currentSegmentIndex > 0) {
      // Reset audio error when changing segments
      setAudioError(false)
      setCurrentSegmentIndex(currentSegmentIndex - 1)
    }
  }

  const handleEnded = () => {
    if (segments && currentSegmentIndex < segments.length - 1) {
      handleNext()
    } else {
      setIsPlaying(false)
    }
  }

  const getStatusBadge = () => {
    switch (currentStatus) {
      case 'ready':
        return (
          <StatusBadge backgroundColor="#4CAF50">
            <Text fontSize={12} color="white">✅ Ready</Text>
          </StatusBadge>
        )
      case 'synthesizing':
        return (
          <StatusBadge backgroundColor="#2196F3">
            <Text fontSize={12} color="white">🎙️ Synthesizing</Text>
          </StatusBadge>
        )
      case 'waiting':
        return (
          <StatusBadge backgroundColor="#FF9800">
            <Text fontSize={12} color="white">⏸️ Waiting for mapping</Text>
          </StatusBadge>
        )
      case 'error':
        return (
          <StatusBadge backgroundColor="#F44336">
            <Text fontSize={12} color="white">❌ Error</Text>
          </StatusBadge>
        )
    }
  }

  if (isLoading || isRefetching) {
    return (
      <Container>
        <Text>Loading segments...</Text>
      </Container>
    )
  }

  // Only show error if we've exhausted all retries
  if (error && !isRefetching) {
    return (
      <Container>
        <YStack gap={12}>
          <Text color="$error">Error loading segments: {error.message}</Text>
          <Text fontSize={12} color="#666">
            The system is still processing your book. Please wait a moment and the segments should appear.
          </Text>
        </YStack>
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

  const readySegments = segments.filter(
    (seg) => seg.voice_id !== undefined && seg.voice_id !== ''
  ).length

  return (
    <Container>
      <audio
        ref={audioRef}
        onEnded={handleEnded}
        onError={() => {
          setAudioError(true)
          setIsPlaying(false)
        }}
        aria-label="Book segment audio player"
      />

      <XStack justifyContent="space-between" alignItems="center">
        <Text fontSize={20} fontWeight="bold">
          Book Player
        </Text>
        {getStatusBadge()}
      </XStack>

      <YStack gap={4}>
        <Text fontSize={14} color="#8e8e93">
          Segment {currentSegmentIndex + 1} of {segments.length}
        </Text>
        <Text fontSize={12} color="#666">
          {readySegments} / {segments.length} ready for playback
        </Text>
      </YStack>

      {currentSegment && (
        <SegmentText 
          backgroundColor={isPlaying ? '#f0f8ff' : currentStatus === 'waiting' ? '#fff3e0' : '#f5f5f5'}
        >
          {currentSegment.text}
        </SegmentText>
      )}

      {currentSegment && (
        <YStack gap={4}>
          <Text fontSize={12} color="#8e8e93">
            Speaker: {currentSegment.person}
          </Text>
          {personas?.mapped[currentSegment.person] && (
            <Text fontSize={12} color="#8e8e93">
              Voice: {currentSegment.voice_description}
            </Text>
          )}
        </YStack>
      )}

      {currentStatus === 'waiting' && (
        <YStack padding={12} backgroundColor="#FFF3E0" borderRadius={8}>
          <Text fontSize={14} color="#FF6F00">
            ⏸️ This segment requires voice mapping for "{currentSegment?.person}"
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
          disabled={currentStatus !== 'ready'}
          backgroundColor="$primary"
          color="white"
        >
          {isPlaying ? 'Pause' : currentStatus !== 'ready' ? 'Not Ready' : 'Play'}
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
