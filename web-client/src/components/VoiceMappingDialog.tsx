import { useState, useEffect } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { usePersonas, useVoices, useSetVoiceMap } from '../api/hooks'

const Overlay = styled(YStack, {
  top: 0,
  right: 0,
  bottom: 0,
  width: 400,
  backgroundColor: 'white',
  boxShadow: '-4px 0 12px rgba(0,0,0,0.15)',
  padding: 24,
  gap: 20,
  zIndex: 1000,
  overflow: 'scroll',
})

const Header = styled(YStack, {
  gap: 8,
  paddingBottom: 16,
  borderBottomWidth: 1,
  borderBottomColor: '#e0e0e0',
})

const PersonaCard = styled(YStack, {
  gap: 12,
  padding: 16,
  borderRadius: 8,
  borderWidth: 1,
  borderColor: '#e0e0e0',
  backgroundColor: '#f9f9f9',
})

const PersonaListItem = styled(XStack, {
  gap: 8,
  padding: 12,
  borderRadius: 8,
  borderWidth: 1,
  borderColor: '#e0e0e0',
  backgroundColor: 'white',
  alignItems: 'center',
  cursor: 'pointer',
  hoverStyle: {
    backgroundColor: '#f5f5f5',
    borderColor: '#2196F3',
  },
})

const Modal = styled(YStack, {
  backgroundColor: 'white',
  borderRadius: 12,
  boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
  padding: 24,
  gap: 16,
  width: 400,
  zIndex: 2000,
})

const Backdrop = styled(YStack, {
  backgroundColor: 'rgba(0,0,0,0.5)',
  zIndex: 1999,
})

interface VoiceMappingDialogProps {
  bookId: string
  onComplete?: () => void
}

export function VoiceMappingDialog({ bookId, onComplete }: VoiceMappingDialogProps) {
  const { data: personas, isLoading: personasLoading } = usePersonas(bookId)
  const { data: voices } = useVoices()
  const setVoiceMapMutation = useSetVoiceMap(bookId)

  const [selectedVoices, setSelectedVoices] = useState<Record<string, string>>({})
  const [isInitialMapping, setIsInitialMapping] = useState(true)
  const [selectedPersona, setSelectedPersona] = useState<string | null>(null)
  const [tempVoiceSelection, setTempVoiceSelection] = useState<string>('')

  // Determine if this is initial mapping or update
  useEffect(() => {
    if (personas) {
      // Initial mapping if no personas are mapped yet
      const hasExistingMappings = Object.keys(personas.mapped).length > 0
      setIsInitialMapping(!hasExistingMappings)
      
      // Pre-populate existing mappings
      setSelectedVoices(personas.mapped as Record<string, string>)
      
      // Debug: log persona state changes
      console.log('[VoiceMappingDialog] Personas updated:', {
        discovered: personas.discovered,
        mapped: Object.keys(personas.mapped),
        unmapped: personas.unmapped,
      })
    }
  }, [personas])

  // Auto-close dialog when all personas are mapped
  useEffect(() => {
    if (personas && personas.unmapped.length === 0 && personas.discovered.length > 0) {
      console.log('[VoiceMappingDialog] All personas mapped, closing dialog')
      if (onComplete) {
        onComplete()
      }
    }
  }, [personas, onComplete])

  if (!personas || personas.discovered.length === 0) {
    return null // Don't show dialog if no personas discovered
  }

  const handlePersonaClick = (persona: string) => {
    // If persona is already mapped, allow remapping
    setSelectedPersona(persona)
    setTempVoiceSelection(selectedVoices[persona] || '')
    console.log('[VoiceMappingDialog] Persona clicked:', persona)
  }

  const handleCloseModal = () => {
    setSelectedPersona(null)
    setTempVoiceSelection('')
  }

  const handleConfirmMapping = () => {
    if (!selectedPersona || !tempVoiceSelection) {
      alert('Please select a voice')
      return
    }

    console.log('[VoiceMappingDialog] Confirming mapping:', {
      persona: selectedPersona,
      voice: tempVoiceSelection,
      isInitial: isInitialMapping,
    })

    // Update local state optimistically
    const newMappings = { ...selectedVoices, [selectedPersona]: tempVoiceSelection }
    setSelectedVoices(newMappings)

    // Combine with existing mappings from backend
    const allMappings = { ...personas.mapped, [selectedPersona]: tempVoiceSelection }

    const voiceMap = {
      persons: Object.entries(allMappings).map(([id, provider_voice]) => ({
        id,
        provider_voice: provider_voice as string,
      })),
    }

    setVoiceMapMutation.mutate(
      {
        voiceMap,
        options: {
          initial: isInitialMapping,
          update: !isInitialMapping,
        },
      },
      {
        onSuccess: () => {
          console.log('[VoiceMappingDialog] Voice mapping successful')
          // Close the modal
          handleCloseModal()
          // After first mapping, all subsequent mappings are updates
          setIsInitialMapping(false)
        },
        onError: (error) => {
          console.error('[VoiceMappingDialog] Voice mapping failed:', error)
          // Revert optimistic update
          setSelectedVoices(personas.mapped as Record<string, string>)
        },
      }
    )
  }

  return (
    <>
      <Overlay style={{ position: 'fixed' } as React.CSSProperties}>
        <Header>
          <Text fontSize={20} fontWeight="bold">
            🎭 Voice Mapping
          </Text>
          <Text fontSize={14} color="#666">
            Click on a persona to assign a voice
          </Text>
          <Text fontSize={12} color="#999">
            {personas.unmapped.length} persona{personas.unmapped.length !== 1 ? 's' : ''} remaining
          </Text>
        </Header>

        {personas.pending_segments > 0 && (
          <PersonaCard style={{ backgroundColor: '#FFF3E0' } as React.CSSProperties}>
            <Text fontSize={14} color="#FF6F00" fontWeight="600">
              ⏸️ {personas.pending_segments} segment{personas.pending_segments !== 1 ? 's' : ''} waiting for voice mapping
            </Text>
            <Text fontSize={12} color="#666">
              Map voices to resume synthesis
            </Text>
          </PersonaCard>
        )}

        <YStack gap={12}>
          <Text fontSize={16} fontWeight="600">
            All Discovered Personas:
          </Text>
          {personas.discovered.map((persona) => {
            const isMapped = !!personas.mapped[persona]
            const voiceName = isMapped 
              ? voices?.voices.find(v => v.id === personas.mapped[persona])?.name 
              : null

            return (
              <PersonaListItem
                key={persona}
                onPress={() => handlePersonaClick(persona)}
                style={{
                  borderColor: isMapped ? '#4CAF50' : '#FF9800',
                  backgroundColor: isMapped ? '#F1F8F4' : 'white',
                } as React.CSSProperties}
              >
                <Text fontSize={20}>{isMapped ? '✅' : '⏳'}</Text>
                <YStack flex={1} gap={4}>
                  <Text fontSize={14} fontWeight="600">
                    {persona}
                  </Text>
                  {voiceName && (
                    <Text fontSize={12} color="#666">
                      {voiceName}
                    </Text>
                  )}
                </YStack>
                <Text fontSize={12} color="#2196F3" fontWeight="600">
                  {isMapped ? 'Change' : 'Set Voice'}
                </Text>
              </PersonaListItem>
            )
          })}
        </YStack>
      </Overlay>

      {/* Voice Selection Modal */}
      {selectedPersona && (
        <>
          <Backdrop 
            onPress={handleCloseModal}
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
            } as React.CSSProperties}
          />
          <Modal
            style={{
              position: 'fixed',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
            } as React.CSSProperties}
          >
            <YStack gap={16}>
              <YStack gap={8}>
                <Text fontSize={18} fontWeight="bold">
                  Select Voice for {selectedPersona}
                </Text>
                {personas.pending_segments > 0 && !personas.mapped[selectedPersona] && (
                  <Text fontSize={12} color="#FF6F00">
                    Mapping this voice will trigger synthesis
                  </Text>
                )}
              </YStack>

              <YStack gap={8}>
                <Text fontSize={14} fontWeight="500">
                  Choose a voice:
                </Text>
                <select
                  value={tempVoiceSelection}
                  onChange={(e) => setTempVoiceSelection(e.target.value)}
                  style={{
                    padding: '12px',
                    borderRadius: '8px',
                    border: '1px solid #e0e0e0',
                    fontSize: '14px',
                    backgroundColor: 'white',
                    width: '100%',
                  }}
                  autoFocus
                >
                  <option value="">-- Select a voice --</option>
                  {voices?.voices.map((voice) => (
                    <option key={voice.id} value={voice.id}>
                      {voice.name} {voice.gender ? `(${voice.gender})` : ''} - {voice.provider}
                    </option>
                  ))}
                </select>
              </YStack>

              <XStack gap={12} justifyContent="flex-end">
                <Button
                  onPress={handleCloseModal}
                  backgroundColor="$gray5"
                  color="$gray11"
                  disabled={setVoiceMapMutation.isPending}
                >
                  Cancel
                </Button>
                <Button
                  onPress={handleConfirmMapping}
                  backgroundColor="$primary"
                  color="white"
                  disabled={!tempVoiceSelection || setVoiceMapMutation.isPending || personasLoading}
                >
                  {setVoiceMapMutation.isPending || personasLoading ? 'Confirming...' : 'Confirm Mapping'}
                </Button>
              </XStack>

              {setVoiceMapMutation.isError && (
                <Text fontSize={12} color="$error">
                  Error: {setVoiceMapMutation.error.message}
                </Text>
              )}
            </YStack>
          </Modal>
        </>
      )}
    </>
  )
}
