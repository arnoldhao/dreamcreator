import * as React from "react";
import { Events, System } from "@wailsio/runtime";
import {
  BarChart3,
  Bot,
  Brain,
  Bug,
  Cog,
  Hammer,
  Info,
  Plug2,
  Server,
  Sparkles,
} from "lucide-react";

import { AboutSection } from "@/features/settings/about";
import { GeneralSection } from "@/features/settings/general";
import { ProviderSection } from "@/features/settings/provider";
import { DebugSection } from "@/features/settings/debug";
import { UsageSection } from "@/features/settings/usage";
import { MemorySection } from "@/features/settings/memory";
import { IntegrationSection } from "@/features/settings/integration";
import type { IntegrationTabId } from "@/features/settings/integration";
import { GatewaySection } from "@/features/settings/gateway";
import { CallsSkillsTab, CallsToolsTab } from "@/features/settings/calls";
import type { MemoryTabId } from "@/features/settings/memory";
import {
  consumePendingSettingsSection,
  listenPendingSettingsSection,
  isSettingsSection,
  type SettingsSectionId,
} from "./sectionStorage";
import {
  useOpenLogDirectory,
  useSelectDownloadDirectory,
  useSettings,
  useTestProxy,
  useUpdateSettings,
} from "@/shared/query/settings";
import { useI18n } from "@/shared/i18n";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import { useDebugMode } from "@/shared/store/debug";
import { AppShell } from "@/components/layout/AppShell";
import { LoadingState } from "@/shared/ui/LoadingState";
import type { MemorySettings, ProxySettings } from "@/shared/contracts/settings";
import { messageBus } from "@/shared/message";
import { mergeIncomingProxyDraft } from "./proxyDraft";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/shared/ui/sidebar";

const defaultMemorySettings: MemorySettings = {
  enabled: true,
  embeddingProviderId: "",
  embeddingModel: "",
  llmProviderId: "",
  llmModel: "",
  recallTopK: 5,
  vectorWeight: 0.7,
  textWeight: 0.3,
  recencyWeight: 0.15,
  recencyHalfLifeDays: 14,
  minScore: 0.35,
  autoRecall: true,
  autoCapture: true,
  sessionLifecycle: true,
  captureMaxEntries: 3,
};

function resolveMemorySettings(memory: MemorySettings | undefined): MemorySettings {
  return {
    ...defaultMemorySettings,
    ...(memory ?? {}),
  };
}

interface SettingsSidebarProps {
  activeSection: SettingsSectionId;
  onSectionChange: (section: SettingsSectionId) => void;
  className?: string;
  style?: React.CSSProperties;
}

function SettingsSidebar({ activeSection, onSectionChange, className, style }: SettingsSidebarProps) {
  const { t } = useI18n();
  const { enabled: debugEnabled } = useDebugMode();
  const { enabled: assistantUiEnabled } = useAssistantUiMode();

  const menuItems: { id: SettingsSectionId; icon: React.ComponentType<React.SVGProps<SVGSVGElement>>; label: string }[] = [
    { id: "gateway", icon: Bot, label: t("app.settings.title.gateway") },
    { id: "general", icon: Cog, label: t("app.settings.title.general") },
    { id: "provider", icon: Server, label: t("app.settings.title.provider") },
    { id: "integration", icon: Plug2, label: t("app.settings.title.integration") },
    { id: "usage", icon: BarChart3, label: t("app.settings.title.usage") },
  ];
  if (assistantUiEnabled) {
    menuItems.splice(3, 0,
      { id: "tools", icon: Hammer, label: t("app.settings.title.tools") },
      { id: "skills", icon: Sparkles, label: t("app.settings.title.skills") },
      { id: "memory", icon: Brain, label: t("app.settings.title.memory") }
    );
  }
  if (debugEnabled) {
    menuItems.push({ id: "debug", icon: Bug, label: t("app.settings.title.debug") });
  }
  menuItems.push({ id: "about", icon: Info, label: t("app.settings.title.about") });

  return (
    <Sidebar
      variant="floating"
      collapsible="offcanvas"
      side="left"
      className={className}
      style={{ "--app-sidebar-padding-right": "0px", ...style } as React.CSSProperties}
    >
      {/* 同步主 sidebar 的 1px 上移，抵消浮动容器边框带来的垂直偏差 */}
      <SidebarHeader
        className="-mt-px h-[var(--app-sidebar-title-height)] justify-center gap-0"
        style={
          {
            "--wails-draggable": "drag",
            paddingTop: "var(--app-sidebar-padding)",
            paddingBottom: "var(--app-sidebar-padding)",
            paddingRight: "var(--app-sidebar-padding)",
            paddingLeft: "var(--app-sidebar-padding)",
          } as React.CSSProperties
        }
      >
        <div className="flex-1" />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu className="gap-1">
            {menuItems.map((item) => (
              <SidebarMenuItem key={item.id}>
                <SidebarMenuButton
                  isActive={activeSection === item.id}
                  onClick={() => onSectionChange(item.id)}
                >
                  <item.icon className="text-muted-foreground" />
                  <span className="truncate">{item.label}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}

export function SettingsApp() {
  const { data: settings, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();
  const testProxy = useTestProxy();
  const openLogDirectory = useOpenLogDirectory();
  const selectDownloadDirectory = useSelectDownloadDirectory();
  const { t } = useI18n();
  const { enabled: debugEnabled } = useDebugMode();
  const { enabled: assistantUiEnabled } = useAssistantUiMode();
  const isWindows = System.IsWindows();

  const [activeSection, setActiveSection] = React.useState<SettingsSectionId>("gateway");
  const [memoryTab, setMemoryTab] = React.useState<MemoryTabId>("summary");
  const [integrationTab, setIntegrationTab] = React.useState<IntegrationTabId>(
    assistantUiEnabled ? "channel" : "connector"
  );
  const [proxyDraft, setProxyDraft] = React.useState<ProxySettings | null>(null);
  const lastSavedProxyRef = React.useRef<ProxySettings | null>(null);
  const resolveVisibleIntegrationTab = React.useCallback(
    (tab: IntegrationTabId): IntegrationTabId => {
      if (!assistantUiEnabled && tab === "channel") {
        return "connector";
      }
      return tab;
    },
    [assistantUiEnabled]
  );
  const resolveVisibleSection = React.useCallback(
    (section: SettingsSectionId): SettingsSectionId => {
      if (!assistantUiEnabled && (section === "tools" || section === "skills" || section === "memory")) {
        return "about";
      }
      if (!debugEnabled && section === "debug") {
        return "about";
      }
      return section;
    },
    [assistantUiEnabled, debugEnabled]
  );

  React.useEffect(() => {
    setIntegrationTab((current) => resolveVisibleIntegrationTab(current));
  }, [resolveVisibleIntegrationTab]);

  React.useEffect(() => {
    const pending = consumePendingSettingsSection();
    if (pending) {
      if (pending === "connectors") {
        setIntegrationTab("connector");
        setActiveSection("integration");
      } else if (pending === "external-tools") {
        setIntegrationTab("external-tool");
        setActiveSection("integration");
      } else {
        setActiveSection(resolveVisibleSection(pending));
      }
    }
    const unsubscribe = listenPendingSettingsSection((section) => {
      if (section === "connectors") {
        setIntegrationTab("connector");
        setActiveSection("integration");
        return;
      }
      if (section === "external-tools") {
        setIntegrationTab("external-tool");
        setActiveSection("integration");
        return;
      }
      setActiveSection(resolveVisibleSection(section));
    });
    return () => unsubscribe();
  }, [resolveVisibleSection]);

  React.useEffect(() => {
    const offNavigate = Events.On("settings:navigate", (event: any) => {
      const target = event?.data ?? event;
      if (target === "appearance") {
        setActiveSection("general");
        return;
      }
      if (isSettingsSection(target)) {
        if (target === "connectors") {
          setIntegrationTab("connector");
          setActiveSection("integration");
          return;
        }
        if (target === "external-tools") {
          setIntegrationTab("external-tool");
          setActiveSection("integration");
          return;
        }
        setActiveSection(resolveVisibleSection(target));
        return;
      }
      if (target === "about") {
        setActiveSection("about");
      }
    });
    return () => {
      offNavigate();
    };
  }, [resolveVisibleSection]);

  React.useEffect(() => {
    if (!settings?.proxy) {
      return;
    }
    const previousSavedProxy = lastSavedProxyRef.current;
    setProxyDraft((current) => mergeIncomingProxyDraft(current, previousSavedProxy, settings.proxy));
    lastSavedProxyRef.current = settings.proxy;
  }, [settings?.proxy]);

  const effectiveActiveSection = resolveVisibleSection(activeSection);

  React.useEffect(() => {
    if (effectiveActiveSection !== activeSection) {
      setActiveSection(effectiveActiveSection);
    }
  }, [activeSection, effectiveActiveSection]);

  const resolvedSection =
    effectiveActiveSection === "connectors" || effectiveActiveSection === "external-tools"
      ? "integration"
      : effectiveActiveSection;
  const currentTitle = t(`app.settings.title.${resolvedSection}`);
  const CurrentIcon =
    resolvedSection === "gateway"
      ? Bot
      : resolvedSection === "general"
        ? Cog
      : resolvedSection === "provider"
          ? Server
          : resolvedSection === "tools"
            ? Hammer
            : resolvedSection === "skills"
              ? Sparkles
            : resolvedSection === "memory"
              ? Brain
              : resolvedSection === "integration"
                ? Plug2
                : resolvedSection === "usage"
                  ? BarChart3
                  : resolvedSection === "debug"
                    ? Info
                    : Info;

  const handleFontFamilyChange = (fontFamily: string) => {
    updateSettings.mutate({ fontFamily });
  };

  const handleFontSizeChange = (fontSize: number) => {
    updateSettings.mutate({ fontSize });
  };

  const handleAppearanceChange = (appearance: "light" | "dark" | "auto") => {
    updateSettings.mutate({ appearance });
  };

  const handleThemeColorChange = (themeColor: string) => {
    updateSettings.mutate({ themeColor });
  };

  const handleColorSchemeChange = (colorScheme: "default" | "contrast" | "slate" | "warm") => {
    updateSettings.mutate({ colorScheme });
  };

  const handleLanguageChange = (language: string) => {
    updateSettings.mutate({ language });
  };

  const handleLogLevelChange = (logLevel: string) => {
    updateSettings.mutate({ logLevel });
  };

  const handleAutoStartChange = (autoStart: boolean) => {
    updateSettings.mutate({ autoStart });
  };

  const handleMinimizeToTrayOnStartChange = (minimizeToTrayOnStart: boolean) => {
    updateSettings.mutate({ minimizeToTrayOnStart });
  };

  const handleOpenLogDirectory = () => {
    openLogDirectory.mutate();
  };

  const handleSelectDownloadDirectory = async () => {
    try {
      const title = t("settings.general.download.dialogTitle");
      const selected = await selectDownloadDirectory.mutateAsync(title);
      if (selected && selected.trim()) {
        updateSettings.mutate({ downloadDirectory: selected });
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      if (message.toLowerCase().includes("cancel")) {
        return;
      }
      messageBus.publishToast({
        title: t("settings.general.download.selectError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleProxyChange = (next: ProxySettings) => {
    setProxyDraft(next);
  };

  const handleProxyTest = async (next: ProxySettings) => {
    try {
      const result = await testProxy.mutateAsync(next);
      setProxyDraft(result);
      if (result.testSuccess) {
        handleProxySave(result);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      setProxyDraft({
        ...next,
        testSuccess: false,
        testMessage: message,
        testedAt: "",
      });
    }
  };

  const handleProxySave = (next: ProxySettings) => {
    updateSettings.mutate(
      { proxy: next },
      {
        onError: (error) => {
          const message = error instanceof Error ? error.message : String(error);
          setProxyDraft({
            ...next,
            testSuccess: false,
            testMessage: message,
            testedAt: "",
          });
        },
        onSuccess: (data) => {
          setProxyDraft(data.proxy);
        },
      }
    );
  };

  if (isLoading || !settings) {
    return <LoadingState message={t("app.settings.loading")} />;
  }

  const isDownloadDirectoryBusy = updateSettings.isPending || selectDownloadDirectory.isPending;
  const isToolsActive = resolvedSection === "tools";
  const isSkillsActive = resolvedSection === "skills";
  const isIntegrationActive = resolvedSection === "integration";
  const isDebugActive = resolvedSection === "debug";
  const isMemoryEntriesActive = resolvedSection === "memory" && memoryTab === "entries";
  const isFluidCardSection =
    resolvedSection === "general" ||
    resolvedSection === "usage" ||
    resolvedSection === "about";
  const resolvedIntegrationTab =
    activeSection === "connectors"
      ? "connector"
      : activeSection === "external-tools"
        ? "external-tool"
        : resolveVisibleIntegrationTab(integrationTab);

  return (
    <AppShell
      activeWindow="settings"
      title={currentTitle}
      titleActions={<CurrentIcon className="h-4 w-4 text-muted-foreground" />}
      sidebarWidth="14rem"
      showTitleSeparator={false}
      boldTitle
      hideTitleBar={!isWindows}
      contentScrollable={!isMemoryEntriesActive}
      contentClassName="px-5 pb-5 pt-5"
      sidebar={
        <SettingsSidebar
          activeSection={resolvedSection}
          onSectionChange={(section) => {
            if (section === "connectors") {
              setIntegrationTab("connector");
              setActiveSection("integration");
              return;
            }
            if (section === "external-tools") {
              setIntegrationTab("external-tool");
              setActiveSection("integration");
              return;
            }
            setActiveSection(resolveVisibleSection(section));
          }}
          className="[&_div[data-sidebar=sidebar]]:!rounded-[var(--app-sidebar-radius)]"
        />
      }
    >
      <div
        className={
          resolvedSection === "gateway" ||
          isToolsActive ||
          isSkillsActive ||
          resolvedSection === "provider" ||
          isIntegrationActive
            ? "flex min-h-0 flex-1 flex-col overflow-hidden"
            : isDebugActive
              ? "flex min-h-0 flex-1 flex-col overflow-hidden"
              : resolvedSection === "memory"
                ? isMemoryEntriesActive
                  ? "flex min-h-0 h-full w-full flex-1 flex-col overflow-hidden"
                  : "w-full"
                : isFluidCardSection
                  ? "w-full space-y-6"
                  : "mx-auto w-full max-w-3xl space-y-6"
        }
      >
        {resolvedSection === "gateway" ? <GatewaySection /> : null}

        {isToolsActive ? (
          <CallsToolsTab />
        ) : null}

        {isSkillsActive ? (
          <CallsSkillsTab />
        ) : null}

        {resolvedSection === "general" ? (
          <GeneralSection
            language={settings.language}
            logLevel={settings.logLevel}
            appearance={settings.appearance}
            fontFamily={settings.fontFamily}
            fontSize={settings.fontSize}
            themeColor={settings.themeColor}
            colorScheme={settings.colorScheme}
            systemThemeColor={settings.systemThemeColor}
            proxy={settings.proxy}
            proxyDraft={proxyDraft ?? settings.proxy}
            menuBarVisibility={settings.menuBarVisibility}
            autoStart={settings.autoStart}
            minimizeToTrayOnStart={settings.minimizeToTrayOnStart}
            downloadDirectory={settings.downloadDirectory}
            proxyIsTesting={testProxy.isPending}
            proxyIsSaving={updateSettings.isPending}
            onLanguageChange={handleLanguageChange}
            onLogLevelChange={handleLogLevelChange}
            onAppearanceChange={handleAppearanceChange}
            onFontFamilyChange={handleFontFamilyChange}
            onFontSizeChange={handleFontSizeChange}
            onThemeColorChange={handleThemeColorChange}
            onColorSchemeChange={handleColorSchemeChange}
            onOpenLogDirectory={handleOpenLogDirectory}
            onProxyChange={handleProxyChange}
            onProxyTest={handleProxyTest}
            onProxySave={handleProxySave}
            onMenuBarVisibilityChange={(value) => updateSettings.mutate({ menuBarVisibility: value })}
            onAutoStartChange={handleAutoStartChange}
            onMinimizeToTrayOnStartChange={handleMinimizeToTrayOnStartChange}
            onSelectDownloadDirectory={handleSelectDownloadDirectory}
            downloadDirectoryIsBusy={isDownloadDirectoryBusy}
          />
        ) : null}

        {resolvedSection === "provider" ? (
          <ProviderSection />
        ) : null}

        {resolvedSection === "memory" ? (
          <MemorySection tab={memoryTab} onTabChange={setMemoryTab} memory={resolveMemorySettings(settings.memory)} />
        ) : null}

        {isIntegrationActive ? (
          <IntegrationSection
            tab={resolvedIntegrationTab}
            onTabChange={(tab) => setIntegrationTab(resolveVisibleIntegrationTab(tab))}
            showChannelTab={assistantUiEnabled}
          />
        ) : null}

        {resolvedSection === "usage" ? (
          <UsageSection />
        ) : null}

        {resolvedSection === "about" ? (
          <AboutSection />
        ) : null}

        {resolvedSection === "debug" && debugEnabled ? (
          <DebugSection />
        ) : null}
      </div>
    </AppShell>
  );
}
