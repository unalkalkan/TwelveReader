import { useState, useEffect, useMemo } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { useVoices, useVoiceMap, useSetVoiceMap, useBookSegments } from '../api/hooks'
import type { Voice, PersonVoice, Segment } from '../types/api'

const Container = styled(YStack, {
  gap: 24,
})

const Section = styled(YStack, {
  gap: 12,
})

const PersonCard = styled(YStack, {
  padding: 16,
  borderWidth: 1,
  borderRadius: 8,
  borderColor: '#e0e0e0',
  backgroundColor: '#fafafa',
  gap: 12,
})

const VoiceSelect = styled(XStack, {
  gap: 8,
  flexWrap: 'wrap',
})

const VoiceOption = styled(YStack, {
  padding: 12,
  borderWidth: 2,
  borderRadius: 8,
  cursor: 'pointer',
  minWidth: 150,
  variants: {
    selected: {
      true: {
        borderColor: '#007aff',
        backgroundColor: '#e5f2ff',
      },
      false: {
        borderColor: '#e0e0e0',
        backgroundColor: 'white',
      },
    },
  } as const,
})

const Badge = styled(Text, {
  fontSize: 10,
  paddingHorizontal: 6,
  paddingVertical: 2,
  borderRadius: 4,
  backgroundColor: '#e0e0e0',
  color: '#666',
})

interface VoiceMapperProps {
  bookId: string
  onComplete?: () => void
}

export function VoiceMapper({ bookId, onComplete }: VoiceMapperProps) {
  const { data: voicesResponse, isLoading: voicesLoading, error: voicesError } = useVoices()
  const { data: existingVoiceMap, isLoading: voiceMapLoading } = useVoiceMap(bookId)
  const { data: segments, isLoading: segmentsLoading } = useBookSegments(bookId)
  const setVoiceMapMutation = useSetVoiceMap(bookId)

  // Extract unique persons from segments
  const persons = useMemo((): string[] => {
    if (!segments) return []
    const personSet = new Set<string>()
    segments.forEach((segment: Segment) => {
      if (segment.person) {
        personSet.add(segment.person)
      }
    })
    return Array.from(personSet).sort()
  }, [segments])

  // State for voice mappings
  const [mappings, setMappings] = useState<Record<string, string>>({})

  // Initialize mappings from existing voice map
  useEffect(() => {
    if (existingVoiceMap?.persons) {
      const initial: Record<string, string> = {}
      existingVoiceMap.persons.forEach((p: PersonVoice) => {
        initial[p.id] = p.provider_voice
      })
      setMappings(initial)
    }
  }, [existingVoiceMap])

  const handleVoiceSelect = (person: string, voiceId: string) => {
    setMappings((prev: Record<string, string>) => ({
      ...prev,
      [person]: voiceId,
    }))
  }

  const handleSave = async () => {
    const personVoices: PersonVoice[] = Object.entries(mappings).map(([id, provider_voice]) => ({
      id,
      provider_voice,
    }))

    try {
      await setVoiceMapMutation.mutateAsync({ persons: personVoices })
      onComplete?.()
    } catch (error) {
      console.error('Failed to save voice map:', error)
    }
  }

  const allPersonsMapped = persons.length > 0 && persons.every((p: string) => mappings[p])

  if (voicesLoading || voiceMapLoading || segmentsLoading) {
    return (
      <Container>
        <Text>Loading voices and persons...</Text>
      </Container>
    )
  }

  if (voicesError) {
    return (
      <Container>
        <Text color="$error">Error loading voices: {voicesError.message}</Text>
      </Container>
    )
  }

  if (!voicesResponse?.voices || voicesResponse.voices.length === 0) {
    return (
      <Container>
        <Text color="$error">No TTS voices available. Please configure a TTS provider.</Text>
      </Container>
    )
  }

  if (persons.length === 0) {
    return (
      <Container>
        <Text>No persons/characters detected in this book. The book may not have been segmented yet.</Text>
      </Container>
    )
  }

  // Group voices by provider
  const voicesByProvider = useMemo(() => {
    const grouped: Record<string, Voice[]> = {}
    voicesResponse.voices.forEach((voice: Voice) => {
      if (!grouped[voice.provider]) {
        grouped[voice.provider] = []
      }
      grouped[voice.provider].push(voice)
    })
    return grouped
  }, [voicesResponse.voices])

  return (
    <Container>
      <Section>
        <Text fontSize={20} fontWeight="bold">
          Map Voices to Characters
        </Text>
        <Text fontSize={14} color="#666">
          Select a voice for each character/narrator in your book. 
          Available voices: {voicesResponse.count}
        </Text>
      </Section>

      <Section>
        {persons.map((person: string) => (
          <PersonCard key={person}>
            <XStack justifyContent="space-between" alignItems="center">
              <Text fontSize={16} fontWeight="600">
                {person}
              </Text>
              {mappings[person] && (
                <Badge backgroundColor="#34c759" color="white">
                  ✓ Mapped
                </Badge>
              )}
            </XStack>

            <VoiceSelect>
              {(Object.entries(voicesByProvider) as [string, Voice[]][]).map(([provider, voices]) => (
                <YStack key={provider} gap={8}>
                  <Text fontSize={12} color="#666" fontWeight="500">
                    {provider}
                  </Text>
                  <XStack gap={8} flexWrap="wrap">
                    {voices.map((voice: Voice) => (
                      <VoiceOption
                        key={`${provider}-${voice.id}`}
                        selected={mappings[person] === voice.id}
                        onPress={() => handleVoiceSelect(person, voice.id)}
                        hoverStyle={{ backgroundColor: '#f0f0f0' }}
                      >
                        <Text fontSize={14} fontWeight="500">
                          {voice.name}
                        </Text>
                        <XStack gap={4} flexWrap="wrap">
                          {voice.gender && (
                            <Badge>{voice.gender}</Badge>
                          )}
                          {voice.accent && (
                            <Badge>{voice.accent}</Badge>
                          )}
                          {voice.languages?.slice(0, 2).map((lang: string) => (
                            <Badge key={lang}>{lang}</Badge>
                          ))}
                        </XStack>
                        {voice.description && (
                          <Text fontSize={11} color="#999" numberOfLines={2}>
                            {voice.description}
                          </Text>
                        )}
                      </VoiceOption>
                    ))}
                  </XStack>
                </YStack>
              ))}
            </VoiceSelect>
          </PersonCard>
        ))}
      </Section>

      <XStack gap={12} justifyContent="flex-end">
        <Button
          onPress={handleSave}
          disabled={!allPersonsMapped || setVoiceMapMutation.isPending}
          backgroundColor={allPersonsMapped ? '$primary' : '$secondary'}
          color="white"
        >
          {setVoiceMapMutation.isPending ? 'Saving...' : 'Save Voice Mappings'}
        </Button>
      </XStack>

      {!allPersonsMapped && (
        <Text fontSize={12} color="#ff9500" textAlign="right">
          Please assign a voice to all {persons.length} characters before saving.
        </Text>
      )}

      {setVoiceMapMutation.isError && (
        <Text fontSize={12} color="$error" textAlign="right">
          Error saving: {setVoiceMapMutation.error?.message}
        </Text>
      )}

      {setVoiceMapMutation.isSuccess && (
        <Text fontSize={12} color="#34c759" textAlign="right">
          Voice mappings saved successfully!
        </Text>
      )}
    </Container>
  )
}
