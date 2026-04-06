import * as React from "react";

import { Minus, Plus, Star } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import { Separator } from "@/shared/ui/separator";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import {
  useCreateAssistant,
  useDeleteAssistant,
  useSetDefaultAssistant,
  useUpdateAssistant,
} from "@/shared/query/assistant";
import type { Assistant } from "@/shared/store/assistant";
import { cn } from "@/lib/utils";
import { Input } from "@/shared/ui/input";
interface AssistantManagerPanelProps {
  assistants: Assistant[];
  currentAssistantId: string | null;
  onSelectAssistant: (id: string) => void;
  onOpenAvatar: (id: string) => void;
  onOpenName: (id: string) => void;
  active: boolean;
  sidebarHeader?: React.ReactNode;
}

const buildUniqueName = (base: string, existing: string[]) => {
  if (!existing.includes(base)) {
    return base;
  }
  let index = 2;
  while (existing.includes(`${base} ${index}`)) {
    index += 1;
  }
  return `${base} ${index}`;
};

export function AssistantManagerPanel({
  assistants,
  currentAssistantId,
  onSelectAssistant,
  onOpenAvatar,
  onOpenName,
  active,
  sidebarHeader,
}: AssistantManagerPanelProps) {
  const { t } = useI18n();
  const createAssistant = useCreateAssistant();
  const setDefaultAssistant = useSetDefaultAssistant();
  const deleteAssistant = useDeleteAssistant();
  const updateAssistant = useUpdateAssistant();

  const [selectedId, setSelectedId] = React.useState<string | null>(currentAssistantId);
  const [draftDescription, setDraftDescription] = React.useState("");
  const [isEditingDescription, setIsEditingDescription] = React.useState(false);

  React.useEffect(() => {
    if (active) {
      setSelectedId(currentAssistantId);
    }
  }, [active, currentAssistantId]);

  const selectedAssistant = assistants.find((item) => item.id === selectedId) ?? null;
  React.useEffect(() => {
    setDraftDescription(selectedAssistant?.identity?.creature ?? "");
    setIsEditingDescription(false);
  }, [selectedAssistant?.id, selectedAssistant?.identity?.creature]);

  const handleSelect = (id: string) => {
    setSelectedId(id);
    onSelectAssistant(id);
  };

  const handleCreate = async () => {
    const baseName = t("settings.gateway.manager.newName");
    const name = buildUniqueName(
      baseName,
      assistants.map((item) => item.identity?.name ?? "")
    );
    const baseAssistant = assistants.find((item) => item.id === selectedId) ?? assistants[0];
    const emojiPool = ["🤖", "✨", "🌟", "🧠", "🦋", "🪄", "🌙", "☀️", "🪐", "🎧", "🎨", "📚"];
    const fallbackEmoji = emojiPool[Math.floor(Math.random() * emojiPool.length)] || "🙂";
    const emoji = baseAssistant?.identity?.emoji?.trim() || fallbackEmoji;
    try {
      const defaultCall = {
        tools: { mode: "auto" },
        skills: { mode: "auto" },
      };
      const created = await createAssistant.mutateAsync({
        identity: {
          ...(baseAssistant?.identity ?? {}),
          name,
          emoji,
        },
        avatar: {
          ...(baseAssistant?.avatar ?? {}),
        },
        user: baseAssistant?.user ?? {
          language: { mode: "auto" },
          timezone: { mode: "auto" },
          location: { mode: "auto" },
        },
        model: baseAssistant?.model ?? {
          agent: { primary: "", fallbacks: [], stream: true, temperature: 0.7, maxTokens: 2048 },
          image: { inherit: true, primary: "", fallbacks: [], stream: true, temperature: 0.7, maxTokens: 2048 },
          embedding: { inherit: true, primary: "", fallbacks: [], stream: true, temperature: 0.7, maxTokens: 2048 },
        },
        tools: baseAssistant?.tools ?? { items: [] },
        skills: baseAssistant?.skills ?? { mode: "on", maxSkillsInPrompt: 150, maxPromptChars: 30000 },
        call: defaultCall,
        memory: baseAssistant?.memory ?? { enabled: true },
        enabled: true,
        isDefault: false,
      });
      handleSelect(created.id);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.createError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleSetDefault = async (assistant: Assistant) => {
    try {
      await setDefaultAssistant.mutateAsync({ id: assistant.id });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.defaultError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleDelete = async () => {
    if (!selectedId) {
      return;
    }
    try {
      await deleteAssistant.mutateAsync({ id: selectedId });
      const remaining = assistants.filter((item) => item.id !== selectedId);
      const next = remaining[0]?.id ?? null;
      setSelectedId(next);
      if (next) {
        onSelectAssistant(next);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.deleteError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleDescriptionBlur = async () => {
    if (!selectedAssistant) {
      return;
    }
    const trimmed = draftDescription.trim();
    if (trimmed === (selectedAssistant.identity?.creature ?? "")) {
      setIsEditingDescription(false);
      return;
    }
    try {
      await updateAssistant.mutateAsync({
        id: selectedAssistant.id,
        identity: {
          ...(selectedAssistant.identity ?? {}),
          creature: trimmed,
        },
      });
      setIsEditingDescription(false);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };
  const handleDescriptionKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      event.currentTarget.blur();
    }
  };

  const isDefaultDisabled = !selectedAssistant || selectedAssistant.isDefault || setDefaultAssistant.isPending;
  const isDeleteDisabled =
    !selectedAssistant ||
    deleteAssistant.isPending ||
    assistants.length <= 1 ||
    selectedAssistant.isDefault ||
    selectedAssistant.builtin ||
    !selectedAssistant.deletable;
  const isCreateDisabled = createAssistant.isPending;

  return (
    <CardContent className="flex min-h-0 min-w-0 flex-1 p-0">
      <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
        {sidebarHeader ? (
          <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
            {sidebarHeader}
          </div>
        ) : null}
        <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
          {assistants.length === 0 ? (
            <div className="py-6 text-center text-sm text-muted-foreground">
              {t("settings.gateway.manager.empty")}
            </div>
          ) : (
            <div className="space-y-2">
              {assistants.map((assistant) => {
                const isSelected = assistant.id === selectedId;
                return (
                  <button
                    key={assistant.id}
                    type="button"
                    onClick={() => handleSelect(assistant.id)}
                    className={cn(
                      "flex w-full items-center justify-between gap-2 rounded-md border px-2 py-1 text-sm",
                      isSelected
                        ? "border-primary/60 bg-primary/10"
                        : "border-border/60 hover:border-border"
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate text-left font-medium">
                      {assistant.identity?.name}
                    </span>
                    {isSelected ? (
                      <Badge variant="subtle">
                        {t("settings.gateway.manager.currentSelection")}
                      </Badge>
                    ) : null}
                  </button>
                );
              })}
            </div>
          )}
        </div>
        <div className="flex h-7 shrink-0 items-center justify-between border-t px-[var(--app-sidebar-padding)]">
          <TooltipProvider delayDuration={0}>
            <div className="flex w-full items-center justify-between">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="compactIcon"
                    onClick={() => selectedAssistant && handleSetDefault(selectedAssistant)}
                    disabled={isDefaultDisabled}
                    aria-label={t("settings.gateway.manager.setDefault")}
                  >
                    <Star className="h-3 w-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("settings.gateway.manager.setDefault")}</TooltipContent>
              </Tooltip>
              <div className="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="compactIcon"
                      onClick={handleCreate}
                      aria-label={t("settings.gateway.manager.new")}
                      disabled={isCreateDisabled}
                    >
                      <Plus className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("settings.gateway.manager.new")}</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="compactIcon"
                      onClick={handleDelete}
                      aria-label={t("settings.gateway.manager.delete")}
                      disabled={isDeleteDisabled}
                    >
                      <Minus className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("settings.gateway.manager.delete")}</TooltipContent>
                </Tooltip>
              </div>
            </div>
          </TooltipProvider>
        </div>
      </div>

      <Separator orientation="vertical" className="self-stretch" />

      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-3 px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
        <Card className="shrink-0 shadow-none">
          <div className="flex items-center justify-between gap-2 border-b px-3 py-2 text-sm">
            <div className="min-w-0 flex-1 truncate font-medium">
              {selectedAssistant?.identity?.name ?? t("settings.gateway.manager.name")}
            </div>
            <Button
              type="button"
              size="compact"
              variant="ghost"
              onClick={() => selectedAssistant && onOpenName(selectedAssistant.id)}
              disabled={!selectedAssistant}
            >
              {t("settings.gateway.manager.edit")}
            </Button>
          </div>
          <div className="flex items-center justify-between gap-2 border-b px-3 py-2 text-sm">
            {isEditingDescription ? (
              <Input
                value={draftDescription}
                onChange={(event) => setDraftDescription(event.target.value)}
                onBlur={handleDescriptionBlur}
                onKeyDown={handleDescriptionKeyDown}
                size="compact"
                autoFocus
                placeholder={t("settings.gateway.manager.description")}
                className="h-7 w-48 text-xs"
                disabled={!selectedAssistant || updateAssistant.isPending}
              />
            ) : (
              <div className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                {draftDescription || t("settings.gateway.emptyDescription")}
              </div>
            )}
            <Button
              type="button"
              size="compact"
              variant="ghost"
              onClick={() => setIsEditingDescription(true)}
              disabled={!selectedAssistant}
            >
              {t("settings.gateway.manager.edit")}
            </Button>
          </div>
          <Separator />
          <div className="flex items-center justify-center gap-2 px-3 py-2">
            <Button
              type="button"
              size="compact"
              variant="outline"
              onClick={() => selectedAssistant && handleSetDefault(selectedAssistant)}
              disabled={isDefaultDisabled}
            >
              {t("settings.gateway.manager.setDefault")}
            </Button>
            <Button
              type="button"
              size="compact"
              variant="outline"
              onClick={() => selectedAssistant && onOpenAvatar(selectedAssistant.id)}
              disabled={!selectedAssistant}
            >
              {t("settings.gateway.manager.changeAvatar3d")}
            </Button>
          </div>
        </Card>
      </div>
    </CardContent>
  );
}
