import * as React from "react";
import { ChevronsUpDown } from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import { Separator } from "@/shared/ui/separator";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Button } from "@/shared/ui/button";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useEnabledProvidersWithModels } from "@/shared/query/providers";
import {
  useAssistantProfileOptions,
  useRefreshAssistantUserLocale,
  useUpdateAssistant,
} from "@/shared/query/assistant";
import { useGatewayTools } from "@/shared/query/tools";
import { useMemorySummary } from "@/shared/query/memory";
import {
  useAssistantWorkspaceDirectory,
  useOpenAssistantWorkspaceDirectory,
} from "@/shared/query/workspace";
import { parseModelMeta } from "@/shared/utils/modelMeta";
import type {
  Assistant,
  AssistantIdentity,
  AssistantModel,
  AssistantUser,
  UserExtraField,
} from "@/shared/store/assistant";

import { AssistantEmojiPicker } from "./AssistantEmojiPicker";
import { Assistant3DAvatar } from "./Assistant3DAvatar";
import { AssistantIdentityPanel } from "./assistant-parameter-panels/AssistantIdentityPanel";
import { AssistantMemoryPanel } from "./assistant-parameter-panels/AssistantMemoryPanel";
import { AssistantModelPanel, type ResolvedModelSpec } from "./assistant-parameter-panels/AssistantModelPanel";
import { AssistantSkillsPanel } from "./assistant-parameter-panels/AssistantSkillsPanel";
import { AssistantSoulPanel } from "./assistant-parameter-panels/AssistantSoulPanel";
import { AssistantToolsPanel } from "./assistant-parameter-panels/AssistantToolsPanel";
import { AssistantUserPanel } from "./assistant-parameter-panels/AssistantUserPanel";
import { AssistantWorkspacePanel } from "./assistant-parameter-panels/AssistantWorkspacePanel";
import {
  buildModelRef,
  isImageModel,
  isEmbeddingModel,
  modelRefEquals,
  resolveUsableModelRef,
  type AssistantModelOption,
} from "./assistant-parameter-panels/model-utils";

interface AssistantParametersPanelProps {
  assistant: Assistant;
  assistants: Assistant[];
  onSelectAssistant: (id: string) => void;
  initialTab?: AssistantParameterTab;
  onTabChange?: (tab: AssistantParameterTab) => void;
  sidebarHeader?: React.ReactNode;
}

export type AssistantParameterTab =
  | "models"
  | "identity"
  | "soul"
  | "user"
  | "tools"
  | "skills"
  | "memory"
  | "workspace";

const PARAMETER_TABS: Array<{ id: AssistantParameterTab; labelKey: string; label: string }> = [
  { id: "models", labelKey: "settings.gateway.parameterTab.models", label: "Models" },
  { id: "identity", labelKey: "settings.gateway.parameterTab.identity", label: "Identity" },
  { id: "soul", labelKey: "settings.gateway.parameterTab.soul", label: "Soul" },
  { id: "user", labelKey: "settings.gateway.parameterTab.user", label: "User" },
  { id: "tools", labelKey: "settings.gateway.parameterTab.tools", label: "Tools" },
  { id: "skills", labelKey: "settings.gateway.parameterTab.skills", label: "Skills" },
  { id: "memory", labelKey: "settings.gateway.parameterTab.memory", label: "Memory" },
  { id: "workspace", labelKey: "settings.gateway.parameterTab.workspace", label: "Workspace" },
];

const TOOL_CATEGORY_ORDER = [
  "general",
  "fs",
  "runtime",
  "web",
  "ui",
  "media",
  "messaging",
  "automation",
  "sessions",
  "external_tools",
  "skills",
  "nodes",
  "voice",
  "memory",
  "library",
  "other",
] as const;

const resolveToolCategoryId = (value?: string) => {
  const normalized = value?.trim().toLowerCase();
  return normalized || "general";
};

const normalizeExtraFields = (extra?: UserExtraField[]) => {
  if (!extra) {
    return undefined;
  }
  const cleaned = extra
    .map((item) => ({
      key: item.key?.trim() ?? "",
      value: item.value?.trim() ?? "",
    }))
    .filter((item) => item.key);
  return cleaned.length > 0 ? cleaned : undefined;
};

export function AssistantParametersPanel({
  assistant,
  assistants,
  onSelectAssistant,
  initialTab,
  onTabChange,
  sidebarHeader,
}: AssistantParametersPanelProps) {
  const { t, supportedLanguages, language: appLanguage } = useI18n();
  const updateAssistant = useUpdateAssistant();
  const refreshAssistantUserLocale = useRefreshAssistantUserLocale();
  const assistantProfileOptions = useAssistantProfileOptions();
  const { data: providers = [] } = useEnabledProvidersWithModels();
  const { data: gatewayTools = [], isLoading: gatewayToolsLoading } = useGatewayTools();
  const memorySummary = useMemorySummary(assistant.id);
  const workspaceDirectory = useAssistantWorkspaceDirectory(assistant.id);
  const openWorkspaceDirectory = useOpenAssistantWorkspaceDirectory();

  const [draft, setDraft] = React.useState<Assistant>(assistant);
  const [activeTab, setActiveTab] = React.useState<AssistantParameterTab>(initialTab ?? "models");
  const [activeModelTab, setActiveModelTab] = React.useState<"agent" | "embedding" | "image">("agent");

  React.useEffect(() => {
    setDraft(assistant);
  }, [assistant.id, assistant.updatedAt]);

  React.useEffect(() => {
    if (initialTab) {
      setActiveTab(initialTab);
    }
  }, [initialTab]);

  const modelOptions = React.useMemo<AssistantModelOption[]>(() => {
    return providers.flatMap((provider) => {
      const providerName = provider.provider.name || provider.provider.id;
      return provider.models
        .filter((model) => model.showInUi !== false)
        .map((model) => {
        const meta = parseModelMeta(model);
        const display = (model.displayName || model.name || "").trim() || model.name;
        return {
          value: model.id,
          label: `${providerName} / ${display}`,
          providerId: provider.provider.id,
          modelName: model.name,
          modelRef: buildModelRef(provider.provider.id, model.name),
          model,
          meta,
        };
      });
    });
  }, [providers]);

  const agentModelOptions = modelOptions;
  const imageModelOptions = React.useMemo(() => {
    const filtered = modelOptions.filter((option) => isImageModel(option.model, option.meta));
    return filtered.length > 0 ? filtered : modelOptions;
  }, [modelOptions]);
  const embeddingModelOptions = React.useMemo(() => {
    const filtered = modelOptions.filter((option) => isEmbeddingModel(option.model, option.meta));
    return filtered.length > 0 ? filtered : modelOptions;
  }, [modelOptions]);

  const appLanguageLabel =
    supportedLanguages.find((option) => option.value === appLanguage)?.label ?? appLanguage;

  const resolvedTimezone = React.useMemo(() => {
    if (typeof Intl === "undefined" || !Intl.DateTimeFormat) {
      return "";
    }
    return Intl.DateTimeFormat().resolvedOptions().timeZone ?? "";
  }, []);

  const timezoneOptions = React.useMemo(() => {
    const supportedValuesOf = (Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supportedValuesOf !== "function") {
      return [];
    }
    try {
      return supportedValuesOf("timeZone").slice().sort((a, b) => a.localeCompare(b));
    } catch {
      return [];
    }
  }, []);

  const regionOptions = React.useMemo(() => {
    const supportedValuesOf = (Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] })
      .supportedValuesOf;
    if (typeof supportedValuesOf !== "function") {
      return [];
    }
    try {
      const displayNames =
        typeof Intl.DisplayNames === "function"
          ? new Intl.DisplayNames([appLanguage], { type: "region" })
          : null;
      return supportedValuesOf("region")
        .map((code) => ({
          value: code,
          label: displayNames?.of(code) ?? code,
        }))
        .sort((a, b) => a.label.localeCompare(b.label));
    } catch {
      return [];
    }
  }, [appLanguage]);


  const commitUpdate = React.useCallback(
    async (
      payload: Partial<
        Pick<Assistant, "identity" | "user" | "model" | "tools" | "skills" | "call" | "memory" | "enabled" | "isDefault">
      >
    ) => {
      try {
        await updateAssistant.mutateAsync({ id: assistant.id, ...payload });
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        messageBus.publishToast({
          title: t("settings.gateway.updateError"),
          description: message,
          intent: "warning",
        });
      }
    },
    [assistant.id, t, updateAssistant]
  );

  const handleOpenWorkspace = React.useCallback(async () => {
    try {
      await openWorkspaceDirectory.mutateAsync(assistant.id);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.workspace.openError"),
        description: message,
        intent: "warning",
      });
    }
  }, [assistant.id, openWorkspaceDirectory, t]);

  const updateIdentity = (next: AssistantIdentity, commit = false) => {
    setDraft((prev) => ({ ...prev, identity: next }));
    if (commit) {
      void commitUpdate({ identity: next });
    }
  };

  const updateUser = (next: AssistantUser, commit = false) => {
    setDraft((prev) => ({ ...prev, user: next }));
    if (commit) {
      const normalized = { ...next, extra: normalizeExtraFields(next.extra) };
      setDraft((prev) => ({ ...prev, user: normalized }));
      void commitUpdate({ user: normalized });
    }
  };

  const updateModel = (next: AssistantModel, commit = false) => {
    setDraft((prev) => ({ ...prev, model: next }));
    if (commit) {
      void commitUpdate({ model: next });
    }
  };

  React.useEffect(() => {
    if (agentModelOptions.length === 0) {
      return;
    }
    const currentPrimary = draft.model?.agent?.primary?.trim() ?? "";
    const nextPrimary = resolveUsableModelRef(currentPrimary, agentModelOptions);
    if (!nextPrimary || modelRefEquals(currentPrimary, nextPrimary)) {
      return;
    }
    const nextModel: AssistantModel = {
      ...(draft.model ?? { agent: {}, embedding: {}, image: {} }),
      agent: {
        ...(draft.model?.agent ?? {}),
        primary: nextPrimary,
      },
    };
    setDraft((prev) => ({ ...prev, model: nextModel }));
    void commitUpdate({ model: nextModel });
  }, [agentModelOptions, commitUpdate, draft.model]);

  const updateTools = (next: Assistant["tools"], commit = false) => {
    setDraft((prev) => ({ ...prev, tools: next }));
    if (commit) {
      void commitUpdate({ tools: next });
    }
  };

  const updateSkills = (next: Assistant["skills"], commit = false) => {
    setDraft((prev) => ({ ...prev, skills: next }));
    if (commit) {
      void commitUpdate({ skills: next });
    }
  };

  const updateMemory = (enabled: boolean) => {
    const next = { enabled };
    setDraft((prev) => ({ ...prev, memory: next }));
    void commitUpdate({ memory: next });
  };

  const identity = draft.identity ?? {};
  const soul = identity.soul ?? {};
  const user = draft.user ?? {};
  const userExtra = user.extra ?? [];
  const model = draft.model ?? { agent: {}, image: {}, embedding: {} };
  const tools = draft.tools ?? { items: [] };
  const skills = draft.skills ?? { mode: "on", maxSkillsInPrompt: 150, maxPromptChars: 30000 };
  const memory = draft.memory;
  const profileRoles = assistantProfileOptions.data?.roles ?? [];
  const profileVibes = assistantProfileOptions.data?.vibes ?? [];
  const defaultRole = assistantProfileOptions.data?.defaultRole?.trim() ?? "";
  const defaultVibe = assistantProfileOptions.data?.defaultVibe?.trim() ?? "";
  const roleOptions = React.useMemo(() => {
    const next = [...profileRoles];
    const ensureOption = (value: string) => {
      const trimmed = value.trim();
      if (!trimmed || next.some((item) => item === trimmed)) {
        return;
      }
      next.unshift(trimmed);
    };
    ensureOption(defaultRole);
    const current = identity.role?.trim() ?? "";
    ensureOption(current);
    return next;
  }, [defaultRole, identity.role, profileRoles]);
  const vibeOptions = React.useMemo(() => {
    const next = [...profileVibes];
    const ensureOption = (value: string) => {
      const trimmed = value.trim();
      if (!trimmed || next.some((item) => item === trimmed)) {
        return;
      }
      next.unshift(trimmed);
    };
    ensureOption(defaultVibe);
    const current = soul.vibe?.trim() ?? "";
    ensureOption(current);
    return next;
  }, [defaultVibe, profileVibes, soul.vibe]);

  const toolEntries = React.useMemo(() => {
    const items = tools.items ?? [];
    const lookup = new Map(
      items
        .filter((item) => item?.id)
        .map((item) => [String(item.id), Boolean(item.enabled)])
    );
    return (gatewayTools ?? [])
      .filter((spec) => spec && typeof spec.id === "string" && spec.id.trim())
      .map((spec) => {
        const specId = spec.id.trim();
        const defaultEnabled = items.length === 0 ? true : Boolean(lookup.get(specId));
        const locked = spec.enabled === false;
        const enabled = locked ? false : defaultEnabled;
        return {
          spec,
          enabled,
          locked,
        };
      });
  }, [gatewayTools, tools.items]);

  const toolGroups = React.useMemo(() => {
    const groups = new Map<string, typeof toolEntries>();
    toolEntries.forEach((entry) => {
      const categoryId = resolveToolCategoryId(entry.spec.category);
      const bucket = groups.get(categoryId) ?? [];
      bucket.push(entry);
      groups.set(categoryId, bucket);
    });
    const orderMap = new Map<string, number>();
    TOOL_CATEGORY_ORDER.forEach((value, index) => {
      orderMap.set(value, index);
    });
    const rows = Array.from(groups.entries()).map(([categoryId, items]) => ({
      categoryId,
      categoryLabel: t(`settings.tools.category.${categoryId}`),
      items,
    }));
    rows.sort((left, right) => {
      const leftOrder = orderMap.get(left.categoryId);
      const rightOrder = orderMap.get(right.categoryId);
      if (leftOrder !== undefined && rightOrder !== undefined) {
        return leftOrder - rightOrder;
      }
      if (leftOrder !== undefined) {
        return -1;
      }
      if (rightOrder !== undefined) {
        return 1;
      }
      return left.categoryLabel.localeCompare(right.categoryLabel);
    });
    return rows;
  }, [toolEntries, t]);

  const toolEntryById = React.useMemo(() => {
    const map = new Map<string, (typeof toolEntries)[number]>();
    toolEntries.forEach((entry) => {
      const key = entry.spec.id?.trim();
      if (key) {
        map.set(key, entry);
      }
    });
    return map;
  }, [toolEntries]);

  const handleToolToggle = (toolId: string, enabled: boolean) => {
    const nextItems = (gatewayTools ?? [])
      .filter((spec) => spec && typeof spec.id === "string" && spec.id.trim())
      .map((spec) => {
        const specId = spec.id.trim();
        const current = toolEntryById.get(specId);
        const resolved = specId === toolId ? enabled : current?.enabled ?? true;
        return { id: specId, enabled: spec.enabled === false ? false : resolved };
      });
    updateTools({ items: nextItems }, true);
  };

  const handleSkillsEnabledChange = (enabled: boolean) => {
    updateSkills({ ...skills, mode: enabled ? "on" : "off" }, true);
  };

  const handleSkillsLimitChange = (field: "maxSkillsInPrompt" | "maxPromptChars", value: number) => {
    updateSkills({ ...skills, [field]: Math.max(0, value || 0) }, true);
  };

  const fallbackTemperature = 0.7;
  const fallbackMaxTokens = 2048;
  const resolveModelConfig = (
    spec: AssistantModel["agent"] | undefined,
    target: "agent" | "embedding" | "image"
  ): ResolvedModelSpec => {
    const defaultInherit = target === "embedding" || target === "image";
    return {
      inherit: target === "agent" ? false : (spec?.inherit ?? defaultInherit),
      primary: spec?.primary ?? "",
      fallbacks: Array.isArray(spec?.fallbacks) ? spec?.fallbacks : [],
      stream: spec?.stream ?? true,
      temperature:
        typeof spec?.temperature === "number" && Number.isFinite(spec.temperature) && spec.temperature > 0
          ? spec.temperature
          : fallbackTemperature,
      maxTokens:
        typeof spec?.maxTokens === "number" && Number.isFinite(spec.maxTokens) && spec.maxTokens > 0
          ? spec.maxTokens
          : fallbackMaxTokens,
    };
  };
  const agentSpec = resolveModelConfig(model.agent, "agent");
  const embeddingSpec = resolveModelConfig(model.embedding, "embedding");
  const imageSpec = resolveModelConfig(model.image, "image");

  const renderPanel = () => {
    switch (activeTab) {
      case "models":
        return (
          <AssistantModelPanel
            t={t}
            activeModelTab={activeModelTab}
            onActiveModelTabChange={setActiveModelTab}
            model={model}
            agentSpec={agentSpec}
            embeddingSpec={embeddingSpec}
            imageSpec={imageSpec}
            agentModelOptions={agentModelOptions}
            embeddingModelOptions={embeddingModelOptions}
            imageModelOptions={imageModelOptions}
            onUpdateModel={updateModel}
          />
        );
      case "identity":
        return (
          <AssistantIdentityPanel
            t={t}
            assistant={assistant}
            identity={identity}
            roleOptions={roleOptions}
            defaultRole={defaultRole}
            onUpdateIdentity={updateIdentity}
          />
        );
      case "soul":
        return (
          <AssistantSoulPanel
            t={t}
            identity={identity}
            soul={soul}
            vibeOptions={vibeOptions}
            defaultVibe={defaultVibe}
            onUpdateIdentity={updateIdentity}
          />
        );
      case "user":
        return (
          <AssistantUserPanel
            t={t}
            user={user}
            userExtra={userExtra}
            supportedLanguages={supportedLanguages}
            appLanguageLabel={appLanguageLabel}
            resolvedTimezone={resolvedTimezone}
            timezoneOptions={timezoneOptions}
            regionOptions={regionOptions}
            onUpdateUser={updateUser}
            onRefreshLocale={() => {
              void refreshAssistantUserLocale.mutateAsync(assistant.id).catch(() => undefined);
            }}
          />
        );
      case "tools":
        return (
          <AssistantToolsPanel
            t={t}
            isLoading={gatewayToolsLoading}
            groups={toolGroups}
            onToggle={handleToolToggle}
          />
        );
      case "skills":
        return (
          <AssistantSkillsPanel
            t={t}
            skills={skills}
            onEnabledChange={handleSkillsEnabledChange}
            onLimitChange={handleSkillsLimitChange}
          />
        );
      case "memory":
        return (
          <AssistantMemoryPanel
            t={t}
            enabled={memory.enabled}
            summary={memorySummary.data}
            language={appLanguage}
            isLoading={memorySummary.isLoading || memorySummary.isFetching}
            onEnabledChange={updateMemory}
          />
        );
      case "workspace":
        return (
          <AssistantWorkspacePanel
            t={t}
            rootPath={workspaceDirectory.data?.rootPath?.trim() ?? ""}
            isLoading={workspaceDirectory.isLoading}
            isOpening={openWorkspaceDirectory.isPending}
            onOpenWorkspace={() => {
              void handleOpenWorkspace();
            }}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex min-h-0 min-w-0 flex-1">
        <Card className="flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden">
          <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
            <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
              {sidebarHeader ? (
                <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
                  {sidebarHeader}
                </div>
              ) : null}
              <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
                <SidebarMenu className="gap-1">
                  {PARAMETER_TABS.map((item) => (
                    <SidebarMenuItem key={item.id}>
                      <SidebarMenuButton
                        isActive={activeTab === item.id}
                        onClick={() => {
                          setActiveTab(item.id);
                          onTabChange?.(item.id);
                        }}
                      >
                        <span className="truncate">{t(item.labelKey)}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </div>
              <div className="px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
                <div className="flex flex-col items-center gap-3">
                  <div className="w-full max-w-[180px]">
                    <Assistant3DAvatar
                      assistant={assistant}
                      className="aspect-square w-full"
                      iconClassName="h-6 w-6"
                    />
                  </div>
                  <div className="flex w-full items-center gap-2">
                    <div className="flex min-w-0 flex-1 items-center gap-2">
                      <AssistantEmojiPicker assistant={assistant} emojiClassName="text-base" />
                      <span className="min-w-0 truncate text-left text-sm font-semibold uppercase tracking-wide text-foreground">
                        {assistant.identity?.name}
                      </span>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="outline"
                          size="compactIcon"
                          className="h-7 w-7 rounded-full"
                          aria-label={t("settings.gateway.action.switch")}
                        >
                          <ChevronsUpDown className="h-3.5 w-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-56">
                        {assistants.map((item) => {
                          const emoji = item.identity?.emoji?.trim() || "🙂";
                          return (
                            <DropdownMenuCheckboxItem
                              key={item.id}
                              checked={item.id === assistant.id}
                              onSelect={(event) => {
                                event.preventDefault();
                                if (item.id !== assistant.id) {
                                  onSelectAssistant(item.id);
                                }
                              }}
                              className="gap-2"
                            >
                              <span className="text-base">{emoji}</span>
                              <span className="min-w-0 flex-1 truncate">{item.identity?.name}</span>
                            </DropdownMenuCheckboxItem>
                          );
                        })}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  {!assistant.readiness?.ready && assistant.readiness?.missing?.length ? (
                    <Badge variant="subtle">
                      {t("settings.gateway.readiness.missing")}
                    </Badge>
                  ) : null}
                </div>
              </div>
            </div>

            <Separator orientation="vertical" className="self-stretch" />

            <div className="flex min-h-0 min-w-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-3 py-1.5">
                {renderPanel()}
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
