import React from 'react';
import { View, TouchableOpacity, StyleSheet, Platform } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { Tabs, useRouter } from 'expo-router';

import Colors from '../../constants/Colors';
import { useColorScheme } from '../../src/hooks/useColorScheme';
import { MiniPlayer } from '../../src/components/MiniPlayer';

export default function TabLayout() {
  const theme = useColorScheme();
  const colors = Colors[theme];
  const router = useRouter();

  return (
    <View style={{ flex: 1, backgroundColor: colors.background }}>
      {/* Persistent mini-player above tabs */}
      <View style={styles.miniPlayerWrapper}>
        <MiniPlayer
          title="Perfume"
          author="Patrick Süskind"
        />
      </View>

      <Tabs
        screenOptions={{
          headerShown: false,
          tabBarActiveTintColor: colors.text,
          tabBarInactiveTintColor: colors.tabIconDefault,
          tabBarStyle: {
            backgroundColor: colors.background,
            borderTopColor: colors.border,
            paddingBottom: Platform.OS === 'ios' ? 28 : 8,
            paddingTop: 8,
            height: Platform.OS === 'ios' ? 88 : 64,
          },
          tabBarLabelStyle: {
            fontSize: 10,
            fontWeight: '500',
            textTransform: 'uppercase',
            letterSpacing: 0.5,
          },
        }}
      >
        <Tabs.Screen
          name="index"
          options={{
            title: 'Home',
            tabBarIcon: ({ color, size }) => (
              <MaterialIcons name="home" size={size} color={color} />
            ),
          }}
        />
        <Tabs.Screen
          name="explore"
          options={{
            title: 'Explore',
            tabBarIcon: ({ color, size }) => (
              <MaterialIcons name="explore" size={size} color={color} />
            ),
          }}
        />
        <Tabs.Screen
          name="add"
          options={{
            title: '',
            tabBarIcon: () => (
              <View
                style={[
                  styles.addButton,
                  {
                    backgroundColor:
                      theme === 'dark' ? '#FFFFFF' : '#1E1E1E',
                  },
                ]}
              >
                <MaterialIcons
                  name="add"
                  size={28}
                  color={theme === 'dark' ? '#000000' : '#FFFFFF'}
                />
              </View>
            ),
          }}
        />
        <Tabs.Screen
          name="library"
          options={{
            title: 'Library',
            tabBarIcon: ({ color, size }) => (
              <MaterialIcons name="headset" size={size} color={color} />
            ),
          }}
        />
        <Tabs.Screen
          name="voices"
          options={{
            title: 'Voices',
            tabBarIcon: ({ color, size }) => (
              <MaterialIcons
                name="record-voice-over"
                size={size}
                color={color}
              />
            ),
          }}
        />
      </Tabs>
    </View>
  );
}

const styles = StyleSheet.create({
  miniPlayerWrapper: {
    position: 'absolute',
    bottom: 100,
    left: 16,
    right: 16,
    zIndex: 40,
  },
  addButton: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 24,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 8,
  },
});
