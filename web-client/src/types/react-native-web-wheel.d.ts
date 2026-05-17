import type { ScrollViewProps } from 'react-native';

declare module 'react-native' {
  interface ScrollViewProps {
    onWheel?: (event: { nativeEvent?: { deltaY?: number } }) => void;
  }
}
