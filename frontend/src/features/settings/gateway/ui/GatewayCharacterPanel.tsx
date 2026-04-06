import * as React from "react";
import { Dialogs } from "@wailsio/runtime";
import { ChevronsUpDown, Minus, PencilLine, Plus, RotateCcw } from "lucide-react";

import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { Button } from "@/shared/ui/button";
import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import {
  useAssistant3DAvatarAssets,
  useAssistant3DMotionAssets,
  useDeleteAssistant3DAvatarAsset,
  useDeleteAssistant3DMotionAsset,
  useImportAssistant3DAvatarFromPath,
  useImportAssistant3DMotionFromPath,
  useUpdateAssistant,
  useUpdateAssistantAvatarAsset,
} from "@/shared/query/assistant";
import type { Assistant, Assistant3DAvatarAsset, AssistantAvatarAssetRef } from "@/shared/store/assistant";

import { Assistant3DAvatar } from "./Assistant3DAvatar";
import { AssistantEmojiPicker } from "./AssistantEmojiPicker";

const AVATAR_3D_EXTENSIONS = [".glb", ".vrm"];
const MOTION_EXTENSIONS = [".vrma"];

export type GatewayCharacterTab = "avatar" | "motion";

type RenameTarget = {
  kind: "3davatar" | "vrma";
  asset: Assistant3DAvatarAsset;
};

interface GatewayCharacterPanelProps {
  assistant: Assistant;
  assistants: Assistant[];
  currentAssistantId: string | null;
  activeTab: GatewayCharacterTab;
  onTabChange: (tab: GatewayCharacterTab) => void;
  onSelectAssistant: (id: string) => void;
}

const resolveExtensionAllowed = (path: string, allowed: string[]) => {
  const lower = path.trim().toLowerCase();
  return allowed.some((ext) => lower.endsWith(ext));
};

const stripExtension = (value: string) => value.replace(/\.[^/.]+$/, "");

const getFileName = (path: string) => {
  const trimmed = path.trim();
  if (!trimmed) {
    return "";
  }
  const parts = trimmed.split(/[\\/]/);
  return parts[parts.length - 1] || trimmed;
};

const isBuiltinAssetPath = (path: string) => {
  if (!path) {
    return false;
  }
  const normalized = path.replace(/\\/g, "/").toLowerCase();
  return normalized.includes("/dreamcreator/objects/builtin/");
};

const resolveAssetDisplayName = (
  asset: Assistant3DAvatarAsset,
  assistant: Assistant,
  kind: "3davatar" | "vrma"
) => {
  const name = asset.displayName?.trim();
  if (name) {
    return name;
  }
  if (kind === "3davatar" && assistant.avatar?.avatar3d?.path === asset.path) {
    const currentName = assistant.avatar?.avatar3d?.displayName?.trim();
    if (currentName) {
      return currentName;
    }
  }
  if (kind === "vrma" && assistant.avatar?.motion?.path === asset.path) {
    const currentName = assistant.avatar?.motion?.displayName?.trim();
    if (currentName) {
      return currentName;
    }
  }
  return stripExtension(asset.name);
};

const isBuiltinAsset = (asset: Assistant3DAvatarAsset | null) => {
  if (!asset) {
    return false;
  }
  if (asset.source === "builtin") {
    return true;
  }
  if ((asset.assetId ?? "").startsWith("builtin:")) {
    return true;
  }
  return isBuiltinAssetPath(asset.path);
};

const isUserAsset = (asset: Assistant3DAvatarAsset | null) => {
  if (!asset) {
    return false;
  }
  if (isBuiltinAsset(asset)) {
    return false;
  }
  if (asset.source) {
    return asset.source !== "builtin";
  }
  return true;
};

const findBuiltinAsset = (assets: Assistant3DAvatarAsset[]) =>
  assets.find((asset) => isBuiltinAsset(asset)) ?? null;

const ensureAssistantAsset = (
  assets: Assistant3DAvatarAsset[],
  ref: AssistantAvatarAssetRef | undefined,
  kind: "3davatar" | "vrma"
) => {
  const path = ref?.path?.trim();
  if (!path) {
    return assets;
  }
  if (assets.some((asset) => asset.path === path)) {
    return assets;
  }
  return [
    ...assets,
    {
      kind,
      path,
      name: getFileName(path),
      displayName: ref?.displayName,
      source: ref?.source,
      assetId: ref?.assetId,
    },
  ];
};

export function GatewayCharacterPanel({
  assistant,
  assistants,
  currentAssistantId,
  activeTab,
  onTabChange,
  onSelectAssistant,
}: GatewayCharacterPanelProps) {
  const { t } = useI18n();
  const updateAssistant = useUpdateAssistant();
  const updateAsset = useUpdateAssistantAvatarAsset();
  const importAvatar = useImportAssistant3DAvatarFromPath();
  const importMotion = useImportAssistant3DMotionFromPath();
  const deleteAvatar = useDeleteAssistant3DAvatarAsset();
  const deleteMotion = useDeleteAssistant3DMotionAsset();

  const avatarAssetsQuery = useAssistant3DAvatarAssets("3davatar", true);
  const motionAssetsQuery = useAssistant3DMotionAssets("vrma", true);
  const avatarAssets = React.useMemo(
    () =>
      ensureAssistantAsset(
        avatarAssetsQuery.data ?? [],
        assistant.avatar?.avatar3d,
        "3davatar"
      ),
    [
      avatarAssetsQuery.data,
      assistant.avatar?.avatar3d?.path,
      assistant.avatar?.avatar3d?.displayName,
      assistant.avatar?.avatar3d?.source,
      assistant.avatar?.avatar3d?.assetId,
    ]
  );
  const motionAssets = React.useMemo(
    () =>
      ensureAssistantAsset(
        motionAssetsQuery.data ?? [],
        assistant.avatar?.motion,
        "vrma"
      ),
    [
      motionAssetsQuery.data,
      assistant.avatar?.motion?.path,
      assistant.avatar?.motion?.displayName,
      assistant.avatar?.motion?.source,
      assistant.avatar?.motion?.assetId,
    ]
  );

  const [selectedAvatar, setSelectedAvatar] = React.useState(assistant.avatar?.avatar3d?.path ?? "");
  const [selectedMotion, setSelectedMotion] = React.useState(assistant.avatar?.motion?.path ?? "");
  const [renameTarget, setRenameTarget] = React.useState<RenameTarget | null>(null);
  const [renameValue, setRenameValue] = React.useState("");

  React.useEffect(() => {
    setSelectedAvatar(assistant.avatar?.avatar3d?.path ?? "");
    setSelectedMotion(assistant.avatar?.motion?.path ?? "");
  }, [assistant.id, assistant.updatedAt]);

  const selectedAvatarAsset =
    avatarAssets.find((asset) => asset.path === selectedAvatar) ?? null;
  const selectedMotionAsset =
    motionAssets.find((asset) => asset.path === selectedMotion) ?? null;

  const previewAvatar =
    (activeTab === "avatar" ? selectedAvatar : "") || assistant.avatar?.avatar3d?.path;
  const previewMotion =
    (activeTab === "motion" ? selectedMotion : "") || assistant.avatar?.motion?.path;

  const isBusy =
    updateAssistant.isPending ||
    updateAsset.isPending ||
    importAvatar.isPending ||
    importMotion.isPending ||
    deleteAvatar.isPending ||
    deleteMotion.isPending;

  const handleSelectAvatar = async (asset: Assistant3DAvatarAsset) => {
    if (!assistant) {
      return;
    }
    setSelectedAvatar(asset.path);
    try {
      await updateAssistant.mutateAsync({
        id: assistant.id,
        avatar: {
          ...(assistant.avatar ?? {}),
          avatar3d: {
            ...(assistant.avatar?.avatar3d ?? {}),
            path: asset.path,
            displayName: resolveAssetDisplayName(asset, assistant, "3davatar"),
            source: asset.source,
            assetId: asset.assetId,
          },
        },
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleSelectMotion = async (asset: Assistant3DAvatarAsset) => {
    if (!assistant) {
      return;
    }
    setSelectedMotion(asset.path);
    try {
      await updateAssistant.mutateAsync({
        id: assistant.id,
        avatar: {
          ...(assistant.avatar ?? {}),
          motion: {
            ...(assistant.avatar?.motion ?? {}),
            path: asset.path,
            displayName: resolveAssetDisplayName(asset, assistant, "vrma"),
            source: asset.source,
            assetId: asset.assetId,
          },
        },
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleRenameSave = async () => {
    if (!renameTarget) {
      return;
    }
    const trimmed = renameValue.trim();
    try {
      await updateAsset.mutateAsync({
        kind: renameTarget.kind,
        path: renameTarget.asset.path,
        displayName: trimmed,
      });
      if (renameTarget.kind === "3davatar" && assistant.avatar?.avatar3d?.path === renameTarget.asset.path) {
        await updateAssistant.mutateAsync({
          id: assistant.id,
          avatar: {
            ...(assistant.avatar ?? {}),
            avatar3d: {
              ...(assistant.avatar?.avatar3d ?? {}),
              displayName: trimmed,
            },
          },
        });
      }
      if (renameTarget.kind === "vrma" && assistant.avatar?.motion?.path === renameTarget.asset.path) {
        await updateAssistant.mutateAsync({
          id: assistant.id,
          avatar: {
            ...(assistant.avatar ?? {}),
            motion: {
              ...(assistant.avatar?.motion ?? {}),
              displayName: trimmed,
            },
          },
        });
      }
      setRenameTarget(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleImportAvatar = async () => {
    try {
      const selection = await Dialogs.OpenFile({
        Title: t("settings.gateway.change3davatar.pickTitle"),
        AllowsOtherFiletypes: true,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [
          {
            DisplayName: t("settings.gateway.change3davatar.filter3d"),
            Pattern: "*.glb;*.vrm;*.GLB;*.VRM",
          },
          { DisplayName: "GLB", Pattern: "*.glb;*.GLB" },
          { DisplayName: "VRM", Pattern: "*.vrm;*.VRM" },
          { DisplayName: t("settings.gateway.change3davatar.filterAll"), Pattern: "*.*" },
        ],
      });
      const path = Array.isArray(selection) ? selection[0] : selection;
      if (!path) {
        return;
      }
      if (!resolveExtensionAllowed(path, AVATAR_3D_EXTENSIONS)) {
        messageBus.publishToast({
          title: t("settings.gateway.change3davatar.invalidType"),
          description: t("settings.gateway.change3davatar.invalidHint"),
          intent: "warning",
        });
        return;
      }
      const asset = await importAvatar.mutateAsync({ kind: "3davatar", path });
      await handleSelectAvatar(asset);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.change3davatar.importError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleImportMotion = async () => {
    try {
      const selection = await Dialogs.OpenFile({
        Title: t("settings.gateway.change3dmotion.pickTitle"),
        AllowsOtherFiletypes: true,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [
          {
            DisplayName: t("settings.gateway.change3dmotion.filterMotion"),
            Pattern: "*.vrma;*.VRMA",
          },
          { DisplayName: "VRMA", Pattern: "*.vrma;*.VRMA" },
          { DisplayName: t("settings.gateway.change3dmotion.filterAll"), Pattern: "*.*" },
        ],
      });
      const path = Array.isArray(selection) ? selection[0] : selection;
      if (!path) {
        return;
      }
      if (!resolveExtensionAllowed(path, MOTION_EXTENSIONS)) {
        messageBus.publishToast({
          title: t("settings.gateway.change3dmotion.invalidType"),
          description: t("settings.gateway.change3dmotion.invalidHint"),
          intent: "warning",
        });
        return;
      }
      const asset = await importMotion.mutateAsync({ kind: "vrma", path });
      await handleSelectMotion(asset);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.change3dmotion.importError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleResetAvatar = async () => {
    const builtin = findBuiltinAsset(avatarAssets);
    if (!builtin) {
      messageBus.publishToast({
        title: t("settings.gateway.change3davatar.resetMissing"),
        description: t("settings.gateway.change3davatar.resetMissingHint"),
        intent: "warning",
      });
      return;
    }
    await handleSelectAvatar(builtin);
  };

  const handleResetMotion = async () => {
    const builtin = findBuiltinAsset(motionAssets);
    if (!builtin) {
      messageBus.publishToast({
        title: t("settings.gateway.change3dmotion.resetMissing"),
        description: t("settings.gateway.change3dmotion.resetMissingHint"),
        intent: "warning",
      });
      return;
    }
    await handleSelectMotion(builtin);
  };

  const handleDeleteAvatar = async () => {
    if (!selectedAvatarAsset || !isUserAsset(selectedAvatarAsset)) {
      return;
    }
    try {
      await deleteAvatar.mutateAsync({ kind: "3davatar", path: selectedAvatarAsset.path });
      if (assistant.avatar?.avatar3d?.path === selectedAvatarAsset.path) {
        await handleResetAvatar();
      }
      setSelectedAvatar("");
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.change3davatar.deleteError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const handleDeleteMotion = async () => {
    if (!selectedMotionAsset || !isUserAsset(selectedMotionAsset)) {
      return;
    }
    try {
      await deleteMotion.mutateAsync({ kind: "vrma", path: selectedMotionAsset.path });
      if (assistant.avatar?.motion?.path === selectedMotionAsset.path) {
        await handleResetMotion();
      }
      setSelectedMotion("");
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.change3dmotion.deleteError"),
        description: message,
        intent: "warning",
      });
    }
  };

  const activeAssets = activeTab === "avatar" ? avatarAssets : motionAssets;
  const currentPath =
    activeTab === "avatar" ? assistant.avatar?.avatar3d?.path : assistant.avatar?.motion?.path;
  const selectedPath = activeTab === "avatar" ? selectedAvatar : selectedMotion;
  const isEmpty = activeAssets.length === 0;
  const sidebarTabs = (
    <Tabs
      value={activeTab}
      onValueChange={(value) => {
        if (value === "avatar" || value === "motion") {
          onTabChange(value);
        }
      }}
      className="w-full"
    >
      <TabsList className="grid h-8 w-full grid-cols-2 p-0.5">
        <TabsTrigger
          value="avatar"
          className="h-full py-0 leading-none data-[state=active]:shadow-none"
        >
          {t("settings.gateway.characterTab.avatar")}
        </TabsTrigger>
        <TabsTrigger
          value="motion"
          className="h-full py-0 leading-none data-[state=active]:shadow-none"
        >
          {t("settings.gateway.characterTab.motion")}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );

  return (
    <div className="flex min-h-0 flex-1">
      <Card className="flex min-h-0 flex-1 overflow-hidden">
        <CardContent className="flex min-h-0 flex-1 p-0">
          <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
            <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
              {sidebarTabs}
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
              {isEmpty ? (
                <div className="py-6 text-center text-sm text-muted-foreground">
                  {activeTab === "avatar"
                    ? t("settings.gateway.change3davatar.empty")
                    : t("settings.gateway.change3dmotion.empty")}
                </div>
              ) : (
                <div className="space-y-2">
                  {activeAssets.map((asset) => {
                    const isSelected = asset.path === selectedPath;
                    const isCurrent = asset.path === currentPath;
                    const displayName =
                      resolveAssetDisplayName(
                        asset,
                        assistant,
                        activeTab === "avatar" ? "3davatar" : "vrma"
                      ) || asset.name;
                    const handleSelect = () => {
                      if (isBusy) {
                        return;
                      }
                      if (activeTab === "avatar") {
                        void handleSelectAvatar(asset);
                      } else {
                        void handleSelectMotion(asset);
                      }
                    };
                    return (
                      <div
                        key={asset.path}
                        role="button"
                        tabIndex={isBusy ? -1 : 0}
                        aria-disabled={isBusy}
                        onClick={handleSelect}
                        onKeyDown={(event) => {
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            handleSelect();
                          }
                        }}
                        className={`flex w-full items-center gap-2 rounded-md border px-2 py-1 text-left text-sm ${
                          isSelected ? "border-primary/60 bg-primary/10" : "border-border/60"
                        }`}
                      >
                        <span className="min-w-0 flex-1 truncate font-medium">{displayName}</span>
                        {isCurrent ? (
                          <Badge variant="subtle">{t("settings.gateway.selected")}</Badge>
                        ) : null}
                        <Button
                          type="button"
                          variant="ghost"
                          size="compactIcon"
                          onClick={(event) => {
                            event.stopPropagation();
                            setRenameTarget({ kind: activeTab === "avatar" ? "3davatar" : "vrma", asset });
                            setRenameValue(
                              resolveAssetDisplayName(
                                asset,
                                assistant,
                                activeTab === "avatar" ? "3davatar" : "vrma"
                              )
                            );
                          }}
                          aria-label={t("settings.gateway.change3davatar.edit")}
                        >
                          <PencilLine className="h-3 w-3" />
                        </Button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            <div className="flex h-7 shrink-0 items-center justify-between border-t px-[var(--app-sidebar-padding)]">
              <Button
                type="button"
                variant="ghost"
                size="compactIcon"
                disabled={isBusy}
                aria-label={
                  activeTab === "avatar"
                    ? t("settings.gateway.change3davatar.reset")
                    : t("settings.gateway.change3dmotion.reset")
                }
                onClick={() => void (activeTab === "avatar" ? handleResetAvatar() : handleResetMotion())}
              >
                <RotateCcw className="h-3 w-3 text-destructive" />
              </Button>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="compactIcon"
                  onClick={() => void (activeTab === "avatar" ? handleImportAvatar() : handleImportMotion())}
                  disabled={isBusy}
                  aria-label={
                    activeTab === "avatar"
                      ? t("settings.gateway.change3davatar.add")
                      : t("settings.gateway.change3dmotion.add")
                  }
                >
                  <Plus className="h-3 w-3" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="compactIcon"
                  onClick={() => void (activeTab === "avatar" ? handleDeleteAvatar() : handleDeleteMotion())}
                  disabled={
                    isBusy ||
                    (activeTab === "avatar"
                      ? !isUserAsset(selectedAvatarAsset)
                      : !isUserAsset(selectedMotionAsset))
                  }
                  aria-label={
                    activeTab === "avatar"
                      ? t("settings.gateway.change3davatar.delete")
                      : t("settings.gateway.change3dmotion.delete")
                  }
                >
                  <Minus className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>

          <Separator orientation="vertical" className="self-stretch" />

          <div className="flex min-h-0 min-w-0 flex-1 flex-col items-center justify-center gap-3 px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
            <div className="w-full max-w-[320px]">
              <Assistant3DAvatar
                assistant={assistant}
                avatarPathOverride={previewAvatar}
                motionPathOverride={previewMotion}
                className="aspect-square w-full"
                iconClassName="h-8 w-8"
              />
            </div>
            <div className="flex items-center gap-2">
              <AssistantEmojiPicker assistant={assistant} emojiClassName="text-2xl" />
              <span className="min-w-0 truncate text-sm font-semibold uppercase tracking-wide">
                {assistant.identity?.name}
              </span>
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
                        checked={item.id === currentAssistantId}
                        onSelect={(event) => {
                          event.preventDefault();
                          if (item.id !== currentAssistantId) {
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
            <div className="text-xs text-muted-foreground">
              {assistant.identity?.creature || t("settings.gateway.emptyDescription")}
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={Boolean(renameTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setRenameTarget(null);
          }
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-sm">{t("settings.gateway.change3davatar.edit")}</DialogTitle>
          </DialogHeader>
          <Input
            value={renameValue}
            onChange={(event) => setRenameValue(event.target.value)}
            placeholder={t("settings.gateway.change3davatar.displayNamePlaceholder")}
            size="compact"
          />
          <DialogFooter className="gap-2">
            <Button type="button" variant="outline" size="compact" onClick={() => setRenameTarget(null)}>
              {t("settings.gateway.changeName.cancel")}
            </Button>
            <Button
              type="button"
              size="compact"
              disabled={!renameValue.trim() || isBusy}
              onClick={() => void handleRenameSave()}
            >
              {t("settings.gateway.changeName.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
