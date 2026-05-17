import {
  getPreviewDirectionFromVerticalDelta,
  getPreviewIndex,
} from './src/playerPreview';

function assertEqual<T>(actual: T, expected: T, message: string) {
  if (actual !== expected) {
    throw new Error(`${message}: expected ${expected}, got ${actual}`);
  }
}

assertEqual(getPreviewIndex(5, 5, 10, 'previous'), 4, 'previous preview moves back');
assertEqual(getPreviewIndex(5, 5, 10, 'next'), 6, 'next preview moves forward');
assertEqual(getPreviewIndex(0, 0, 10, 'previous'), 0, 'previous preview clamps at first segment');
assertEqual(getPreviewIndex(9, 9, 10, 'next'), 9, 'next preview clamps at last segment');
assertEqual(getPreviewIndex(4, 5, 10, 'previous'), 3, 'repeated previous preview continues from displayed segment');
assertEqual(getPreviewIndex(6, 5, 10, 'next'), 7, 'repeated next preview continues from displayed segment');
assertEqual(getPreviewIndex(5, 5, 0, 'next'), 5, 'empty segment list leaves active index unchanged');

assertEqual(getPreviewDirectionFromVerticalDelta(48), 'next', 'downward threshold maps to next segment');
assertEqual(getPreviewDirectionFromVerticalDelta(-48), 'previous', 'upward threshold maps to previous segment');
assertEqual(getPreviewDirectionFromVerticalDelta(47), null, 'sub-threshold downward drag is ignored');
assertEqual(getPreviewDirectionFromVerticalDelta(-47), null, 'sub-threshold upward drag is ignored');

console.log('player preview helpers ok');
