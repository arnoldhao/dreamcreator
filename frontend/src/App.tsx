import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Events, System } from '@wailsio/runtime';

import { SettingsApp } from "./app/settings";
import { MainApp } from "./app/main";
import { SETTINGS_QUERY_KEY, useSettings } from "./shared/query/settings";
import { useAssistantUiStorageSync } from "./shared/store/assistantUi";
import { useSettingsStore } from "./shared/store/settings";
import { detectBrowserLanguage, setLanguage } from "./shared/i18n";
import { hexToHsl, pickAccessibleForeground } from "./lib/color";
import { normalizeColorScheme } from "./lib/theme/color-schemes";

function isWailsRuntimeReady() {
  // guard for running outside Wails (e.g. plain browser preview)
  return typeof window !== 'undefined' && typeof (window as any)._wails?.dispatchWailsEvent === 'function';
}

function applyTheme(effectiveAppearance: string | undefined) {
  if (effectiveAppearance === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
}

function applyColorScheme(colorScheme: string | undefined) {
  document.documentElement.dataset.colorScheme = normalizeColorScheme(colorScheme);
}

function systemFontStack() {
  return [
    'system-ui',
    '-apple-system',
    'BlinkMacSystemFont',
    '"Segoe UI"',
    'Roboto',
    '"Noto Sans"',
    'Ubuntu',
    'Cantarell',
    '"Helvetica Neue"',
    'Arial',
    '"Apple Color Emoji"',
    '"Segoe UI Emoji"',
    '"Segoe UI Symbol"',
    'sans-serif',
  ].join(', ');
}

function quoteFontFamily(value: string) {
  const escaped = value.replace(/\\/g, "\\\\").replace(/\"/g, "\\\"");
  return `"${escaped}"`;
}

function buildFontStack(fontFamily: string | undefined) {
  const trimmed = (fontFamily ?? '').trim();
  if (!trimmed) {
    return systemFontStack();
  }
  return `${quoteFontFamily(trimmed)}, ${systemFontStack()}`;
}

function applyFont(fontFamily: string | undefined) {
  const stack = buildFontStack(fontFamily);
  document.documentElement.style.setProperty('--app-font-body', stack);
  document.documentElement.style.setProperty('--app-font-display', stack);
}

function applyFontSize(fontSize: number | undefined) {
  const safeSize = fontSize && fontSize > 0 ? fontSize : 15;
  document.documentElement.style.setProperty('--app-font-size', `${safeSize}px`);
}

function resolveThemeColor(themeColor: string | undefined, systemThemeColor: string | undefined) {
  const trimmed = (themeColor ?? '').trim();
  if (trimmed.toLowerCase() === 'system') {
    return (systemThemeColor ?? '').trim();
  }
  return trimmed;
}

function applyThemeColor(themeColor: string | undefined, systemThemeColor?: string) {
  const color = resolveThemeColor(themeColor, systemThemeColor);
  const hsl = hexToHsl(color);
  const fgHex = pickAccessibleForeground(color);
  const fgHsl = hexToHsl(fgHex ?? undefined);

  if (!hsl || !fgHsl) {
    document.documentElement.style.removeProperty('--primary');
    document.documentElement.style.removeProperty('--primary-foreground');
    document.documentElement.style.removeProperty('--ring');
    document.documentElement.style.removeProperty('--sidebar-primary');
    document.documentElement.style.removeProperty('--sidebar-primary-foreground');
    document.documentElement.style.removeProperty('--sidebar-ring');
    return;
  }

  // Primary palette
  document.documentElement.style.setProperty('--primary', hsl);
  document.documentElement.style.setProperty('--primary-foreground', fgHsl);
  document.documentElement.style.setProperty('--ring', hsl);
  // Sidebar primary to keep nav consistent
  document.documentElement.style.setProperty('--sidebar-primary', hsl);
  document.documentElement.style.setProperty('--sidebar-primary-foreground', fgHsl);
  document.documentElement.style.setProperty('--sidebar-ring', hsl);
}

function detectPlatform() {
  try {
    if (isWailsRuntimeReady()) {
      if (System.IsWindows()) {
        return 'windows';
      }
      if (System.IsMac()) {
        return 'macos';
      }
      if (System.IsLinux()) {
        return 'linux';
      }
    }
  } catch {
    // fall through to user agent detection for browser preview
  }

  const platform = typeof navigator === 'undefined'
    ? ''
    : `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
  if (platform.includes('win')) {
    return 'windows';
  }
  if (platform.includes('mac')) {
    return 'macos';
  }
  if (platform.includes('linux')) {
    return 'linux';
  }
  return 'unknown';
}

function applyPlatformChrome() {
  document.documentElement.dataset.platform = detectPlatform();
}

function App() {
  const queryClient = useQueryClient();
  const { data: settings } = useSettings();
  const setSettings = useSettingsStore((state) => state.setSettings);
  useAssistantUiStorageSync();

  const [windowType, setWindowType] = useState<string>('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setWindowType(params.get('window') || '');
  }, []);

  useEffect(() => {
    // Provide a best-effort initial language before settings arrive.
    setLanguage(detectBrowserLanguage());
  }, []);

  useEffect(() => {
    applyPlatformChrome();
  }, []);

  useEffect(() => {
    if (!settings) {
      return;
    }
    setSettings(settings);
    applyTheme(settings.effectiveAppearance);
    applyColorScheme(settings.colorScheme);
    applyFont(settings.fontFamily);
    applyFontSize(settings.fontSize);
    applyThemeColor(settings.themeColor, settings.systemThemeColor);
    setLanguage(settings.language);
  }, [settings]);

  useEffect(() => {
    if (!isWailsRuntimeReady()) {
      return;
    }

    const offSettingsUpdated = Events.On('settings:updated', (event: any) => {
      const payload = event?.data ?? event;
      if (!payload) {
        return;
      }

      queryClient.setQueryData(SETTINGS_QUERY_KEY, payload);
      setSettings(payload);
      applyTheme(payload.effectiveAppearance);
      applyColorScheme(payload.colorScheme);
      applyFont(payload.fontFamily);
      applyFontSize(payload.fontSize);
      applyThemeColor(payload.themeColor, payload.systemThemeColor);
      if (payload.language) {
        setLanguage(payload.language);
      }
    });

    const offThemeChanged = Events.On('theme:changed', (event: any) => {
      const appearance = event?.data ?? event;
      applyTheme(appearance);
    });

    return () => {
      offSettingsUpdated();
      offThemeChanged();
    };
  }, [queryClient]);

  if (windowType === 'settings') {
    return <SettingsApp />;
  }

  return <MainApp />;
}

export default App
