import { MaterialIcons } from '@expo/vector-icons';
import {
  DarkTheme,
  DefaultTheme,
  ThemeProvider,
} from '@react-navigation/native';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useFonts } from 'expo-font';
import { Stack, useRouter, usePathname } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { useEffect, useState } from 'react';
import 'react-native-reanimated';

import Colors from '../constants/Colors';
import { useColorScheme } from '../src/hooks/useColorScheme';
import { PlaybackProvider } from '../src/store/playbackStore';
import { FavoritesProvider } from '../src/store/favoritesStore';
import { ServerConfigProvider, useServerConfig } from '../src/store/serverConfigStore';
import { AuthProvider, useAuth } from '../src/store/authStore';

export { ErrorBoundary } from 'expo-router';

export const unstable_settings = {
  initialRouteName: 'server-select',
};

SplashScreen.preventAutoHideAsync();

const queryClient = new QueryClient();

/**
 * Inner layout that has access to both color scheme and navigation.
 */
function AppContent() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme];
  const pathname = usePathname();
  const router = useRouter();
  const { isAuthenticated, initialized: authInitialized } = useAuth();

  // Redirect logic:
  // 1. If on server-select and already validated → go to login or tabs
  // 2. If on login and already authenticated → go to tabs
  // 3. If on tabs and NOT authenticated → go to login
  useEffect(() => {
    (async () => {
      try {
        const storedUrl = await AsyncStorage.getItem('twelvereader_server_url');
        const validatedFlag = await AsyncStorage.getItem('twelvereader_server_validated');

        // On server-select: if validated, go to login (or tabs if authenticated)
        if (pathname === '/server-select' && storedUrl && validatedFlag) {
          if (authInitialized && isAuthenticated) {
            router.replace('/(tabs)');
          } else {
            router.replace('/login');
          }
          return;
        }

        // On login: if already authenticated, go to tabs
        if (pathname === '/login' && authInitialized && isAuthenticated) {
          router.replace('/(tabs)');
          return;
        }

        // On main tabs: if NOT authenticated and auth has initialized, go to login
        if (
          pathname?.startsWith('/(tabs)') &&
          authInitialized &&
          !isAuthenticated
        ) {
          router.replace('/login');
        }
      } catch {
        // ignore
      }
    })();
  }, [pathname, router, isAuthenticated, authInitialized]);

  const navTheme = colorScheme === 'dark' ? {
    ...DarkTheme,
    colors: {
      ...DarkTheme.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.accent,
      text: colors.text,
    },
  } : {
    ...DefaultTheme,
    colors: {
      ...DefaultTheme.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.accent,
      text: colors.text,
    },
  };

  return (
    <ThemeProvider value={navTheme}>
      <Stack>
        {/* Server selection — shown first on fresh install or after logout */}
        <Stack.Screen
          name="server-select"
          options={{
            headerShown: false,
            presentation: 'fullScreenModal',
          }}
        />
        {/* Login — magic link authentication */}
        <Stack.Screen
          name="login"
          options={{
            headerShown: false,
            presentation: 'fullScreenModal',
          }}
        />
        {/* Main app tabs */}
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        {/* Full-screen player */}
        <Stack.Screen
          name="player"
          options={{
            headerShown: false,
            presentation: 'fullScreenModal',
            animation: 'slide_from_bottom',
          }}
        />
        {/* Generic modal */}
        <Stack.Screen name="modal" options={{ presentation: 'modal' }} />
      </Stack>
    </ThemeProvider>
  );
}

export default function RootLayout() {
  const [loaded, error] = useFonts({
    SpaceMono: require('../assets/fonts/SpaceMono-Regular.ttf'),
    ...MaterialIcons.font,
  });

  useEffect(() => {
    if (error) throw error;
  }, [error]);

  useEffect(() => {
    if (loaded) SplashScreen.hideAsync();
  }, [loaded]);

  if (!loaded) return null;

  return (
    <QueryClientProvider client={queryClient}>
      <ServerConfigProvider>
        <ApiBaseSyncer />
        <AuthProvider>
          <PlaybackProvider>
            <FavoritesProvider>
              <AppContent />
            </FavoritesProvider>
          </PlaybackProvider>
        </AuthProvider>
      </ServerConfigProvider>
    </QueryClientProvider>
  );
}

/**
 * Sync the persisted server URL into the API client's mutable base on mount / change.
 * Skips the initial render until ServerConfigProvider has finished loading from storage.
 */
function ApiBaseSyncer() {
  const { serverUrl, initialized } = useServerConfig();

  useEffect(() => {
    // Skip until provider has loaded persisted config
    if (!initialized) return;

    (async () => {
      const { setApiBase } = await import('../src/api/client');
      setApiBase(serverUrl.replace(/\/+$/, '') + '/api/v1');
    })();
  }, [serverUrl, initialized]);

  return null; // Pure side-effect component, renders nothing
}
