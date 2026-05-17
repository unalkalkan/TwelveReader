export const READER_PREVIEW_DRAG_THRESHOLD_PX = 48;

type PreviewDirection = 'previous' | 'next';

export function getPreviewIndex(
  currentPreviewIndex: number,
  activeSegmentIndex: number,
  totalSegments: number,
  direction: PreviewDirection,
): number {
  if (totalSegments <= 0) return activeSegmentIndex;

  const boundedPreviewIndex = Math.min(
    Math.max(currentPreviewIndex, 0),
    totalSegments - 1,
  );

  if (direction === 'previous') {
    return Math.max(0, boundedPreviewIndex - 1);
  }

  return Math.min(totalSegments - 1, boundedPreviewIndex + 1);
}

export function getPreviewDirectionFromVerticalDelta(
  deltaY: number,
  thresholdPx = READER_PREVIEW_DRAG_THRESHOLD_PX,
): PreviewDirection | null {
  if (Math.abs(deltaY) < thresholdPx) return null;
  return deltaY > 0 ? 'next' : 'previous';
}
