import { createTamagui } from '@tamagui/core'
import { Text } from '@tamagui/react-native-web'

// Re-export Text for use in components
export { Text }

const config = createTamagui({
  themes: {
    light: {
      background: '#fff',
      color: '#000',
      primary: '#007aff',
      secondary: '#5856d6',
      success: '#34c759',
      warning: '#ff9500',
      error: '#ff3b30',
    },
    dark: {
      background: '#000',
      color: '#fff',
      primary: '#0a84ff',
      secondary: '#5e5ce6',
      success: '#30d158',
      warning: '#ff9f0a',
      error: '#ff453a',
    },
  },
  tokens: {
    space: {
      sm: 8,
      md: 16,
      lg: 24,
      xl: 32,
    },
    size: {
      sm: 32,
      md: 48,
      lg: 64,
      xl: 96,
    },
    radius: {
      sm: 4,
      md: 8,
      lg: 16,
    },
  },
})

export type AppConfig = typeof config

declare module '@tamagui/core' {
  interface TamaguiCustomConfig extends AppConfig {}
}

export default config
