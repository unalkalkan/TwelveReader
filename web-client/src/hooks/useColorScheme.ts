/**
 * useColorScheme that works on all platforms.
 * Re-exports the RN hook directly.
 */
import { useColorScheme as useRNColorScheme } from 'react-native';

export function useColorScheme() {
  return useRNColorScheme() ?? 'dark';
}
