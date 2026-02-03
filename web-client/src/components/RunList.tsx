import { useMemo } from 'react'
import { styled } from '@tamagui/core'
import { YStack, XStack } from '@tamagui/stacks'
import { Button } from '@tamagui/button'
import { Text } from '../tamagui.config'
import { useBooks } from '../api/hooks'
import type { BookStatus } from '../types/api'

const Card = styled(YStack, {
  padding: 16,
  borderRadius: 10,
  borderWidth: 1,
  borderColor: '#e0e0e0',
  backgroundColor: 'white',
  gap: 8,
})

const StatusBadge = styled(XStack, {
  paddingHorizontal: 8,
  paddingVertical: 4,
  borderRadius: 12,
  gap: 6,
  alignItems: 'center',
})

const statusColors: Record<BookStatus, string> = {
  uploaded: '#9E9E9E',
  parsing: '#9E9E9E',
  segmenting: '#2196F3',
  voice_mapping: '#FF9800',
  ready: '#4CAF50',
  synthesizing: '#3F51B5',
  synthesized: '#4CAF50',
  synthesis_error: '#F44336',
  error: '#F44336',
}

function formatDate(value: string) {
  const date = new Date(value)
  return isNaN(date.getTime()) ? value : date.toLocaleString()
}

function StatusLabel({ status }: { status: BookStatus }) {
  const color = statusColors[status] ?? '#9E9E9E'
  const label = status.replace('_', ' ')
  return (
    <StatusBadge backgroundColor={color}>
      <Text fontSize={12} color="white">
        {label}
      </Text>
    </StatusBadge>
  )
}

interface RunListProps {
  activeBookId?: string | null
  onSelect: (bookId: string) => void
}

export function RunList({ activeBookId, onSelect }: RunListProps) {
  const { data: books, isLoading, error, refetch, isRefetching } = useBooks()

  const sortedBooks = useMemo(() => {
    if (!books) return []
    return [...books].sort(
      (a, b) => new Date(b.uploaded_at).getTime() - new Date(a.uploaded_at).getTime()
    )
  }, [books])

  if (isLoading) {
    return (
      <YStack gap={8}>
        <Text fontSize={16} fontWeight="bold">
          Existing Runs
        </Text>
        <Card>
          <Text>Loading runs…</Text>
        </Card>
      </YStack>
    )
  }

  if (error) {
    return (
      <YStack gap={8}>
        <XStack justifyContent="space-between" alignItems="center">
          <Text fontSize={16} fontWeight="bold">
            Existing Runs
          </Text>
          <Button size="$2" onPress={() => refetch()}>
            Retry
          </Button>
        </XStack>
        <Card>
          <Text color="$error">Failed to load runs: {(error as Error).message}</Text>
        </Card>
      </YStack>
    )
  }

  return (
    <YStack gap={8}>
      <XStack justifyContent="space-between" alignItems="center">
        <Text fontSize={16} fontWeight="bold">
          Existing Runs
        </Text>
        <Button size="$2" onPress={() => refetch()} disabled={isRefetching}>
          {isRefetching ? 'Refreshing…' : 'Refresh'}
        </Button>
      </XStack>

      {sortedBooks.length === 0 && (
        <Card>
          <Text color="#666">No runs yet. Upload a book to start processing.</Text>
        </Card>
      )}

      <YStack gap={12}>
        {sortedBooks.map((book) => {
          const isActive = book.id === activeBookId
          return (
            <Card key={book.id} borderColor={isActive ? '#4CAF50' : '#e0e0e0'}>
              <XStack justifyContent="space-between" alignItems="center">
                <YStack gap={4}>
                  <Text fontSize={16} fontWeight="bold">
                    {book.title || 'Untitled'}
                  </Text>
                  <Text fontSize={12} color="#8e8e93">
                    {book.author ? `by ${book.author}` : 'Unknown author'} · Uploaded {formatDate(book.uploaded_at)}
                  </Text>
                </YStack>
                <StatusLabel status={book.status as BookStatus} />
              </XStack>

              <XStack gap={12} alignItems="center" justifyContent="space-between">
                <YStack gap={4}>
                  <Text fontSize={13} color="#666">
                    ID: {book.id}
                  </Text>
                  <Text fontSize={13} color="#666">
                    Segments: {book.total_segments ?? 0}
                  </Text>
                </YStack>
                <Button
                  onPress={() => onSelect(book.id)}
                  backgroundColor={isActive ? '$primary' : '$secondary'}
                  color="white"
                >
                  {isActive ? 'Rejoin' : 'Join'}
                </Button>
              </XStack>
            </Card>
          )
        })}
      </YStack>
    </YStack>
  )
}
