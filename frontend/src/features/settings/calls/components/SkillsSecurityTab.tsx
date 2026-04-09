import * as React from "react";
import { RotateCcw } from "lucide-react";

import { canAdoptIncomingDraftSnapshot, shouldSkipDraftPersist } from "@/app/settings/draftSync";
import { Skeleton } from "@/shared/ui/skeleton";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useSettings, useUpdateSettings } from "@/shared/query/settings";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";
import {
  SETTINGS_CONTROL_WIDTH_CLASS,
  SETTINGS_WIDE_CONTROL_WIDTH_CLASS,
  SettingsCompactListCard,
  SettingsCompactRow,
  SettingsCompactSeparator,
} from "@/shared/ui/settings-layout";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";

import { isRecord } from "../utils/calls-utils";
import {
  buildNextSettingsTools,
  parseSkillsSecurity,
  resolveSkillsSettingsState,
  toSecurityPayload,
  type SkillsActionMode,
  type SkillsSecurityConfig,
} from "../utils/skills-settings-utils";
import { SKILLS_SELECT_TEXT_CLASS } from "./skills-page-styles";

const actionGroups = [
  "read",
  "package_write",
  "deps_write",
  "config_write",
  "source_write",
] as const;

function cloneSecurity(config: SkillsSecurityConfig): SkillsSecurityConfig {
  return {
    actionModes: { ...config.actionModes },
    scannerMode: config.scannerMode,
    allowForceInstall: config.allowForceInstall,
    requireApproval: config.requireApproval,
    allowExternalToolsOnly: config.allowExternalToolsOnly,
    allowGenericInstaller: config.allowGenericInstaller,
    allowedBins: [...config.allowedBins],
  };
}

function toBinsInput(values: string[]): string {
  return values.join(", ");
}

function parseBinsInput(raw: string): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  raw
    .split(",")
    .map((item) => item.trim())
    .forEach((item) => {
      if (!item) {
        return;
      }
      const key = item.toLowerCase();
      if (seen.has(key)) {
        return;
      }
      seen.add(key);
      result.push(item);
    });
  return result;
}

export function SkillsSecurityTab() {
  const { t } = useI18n();
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();

  const settingsToolsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.tools;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.tools]);
  const settingsSkillsRaw = React.useMemo(() => {
    const candidate = settingsQuery.data?.skills;
    return isRecord(candidate) ? (candidate as Record<string, unknown>) : undefined;
  }, [settingsQuery.data?.skills]);
  const { toolsConfig, skillsConfig } = React.useMemo(
    () => resolveSkillsSettingsState(settingsToolsRaw, settingsSkillsRaw),
    [settingsSkillsRaw, settingsToolsRaw]
  );
  const security = React.useMemo(() => parseSkillsSecurity(skillsConfig), [skillsConfig]);

  const [draft, setDraft] = React.useState<SkillsSecurityConfig>(() => cloneSecurity(security));
  const [allowedBinsInput, setAllowedBinsInput] = React.useState<string>("");
  const lastPersistedDraftSignatureRef = React.useRef("");
  const pendingDraftSignatureRef = React.useRef("");
  const currentSecuritySignature = React.useMemo(() => JSON.stringify(security), [security]);
  const previousSecuritySignatureRef = React.useRef(currentSecuritySignature);

  const normalizedDraft = React.useMemo(() => {
    const next = cloneSecurity(draft);
    next.allowedBins = parseBinsInput(allowedBinsInput);
    if (next.allowExternalToolsOnly) {
      next.allowGenericInstaller = false;
    }
    return next;
  }, [allowedBinsInput, draft]);
  const draftSecuritySignature = React.useMemo(() => JSON.stringify(normalizedDraft), [normalizedDraft]);

  React.useEffect(() => {
    const previousSecuritySignature = previousSecuritySignatureRef.current;
    previousSecuritySignatureRef.current = currentSecuritySignature;
    const canAdoptRemote = canAdoptIncomingDraftSnapshot({
      draftSignature: draftSecuritySignature,
      currentRemoteSignature: currentSecuritySignature,
      previousRemoteSignature: previousSecuritySignature,
      lastPersistedSignature: lastPersistedDraftSignatureRef.current,
    });
    if (!canAdoptRemote) {
      return;
    }
    const next = cloneSecurity(security);
    setDraft(next);
    setAllowedBinsInput(toBinsInput(next.allowedBins));
  }, [currentSecuritySignature, draftSecuritySignature, security]);
  const dirty = React.useMemo(() => {
    return currentSecuritySignature !== draftSecuritySignature;
  }, [currentSecuritySignature, draftSecuritySignature]);

  const updateActionMode = React.useCallback((actionGroup: string, mode: SkillsActionMode) => {
    setDraft((previous) => ({
      ...previous,
      actionModes: {
        ...previous.actionModes,
        [actionGroup]: mode,
      },
    }));
  }, []);

  const persistSecurity = React.useCallback(() => {
    const submittedSignature = draftSecuritySignature;
    pendingDraftSignatureRef.current = submittedSignature;
    const nextSkillsConfig: Record<string, unknown> = {
      ...skillsConfig,
      security: toSecurityPayload(normalizedDraft),
    };
    const nextToolsConfig: Record<string, unknown> = {
      ...toolsConfig,
    };
    const nextTools = buildNextSettingsTools(nextToolsConfig);
    updateSettings.mutate(
      {
        tools: nextTools,
        skills: nextSkillsConfig,
      },
      {
        onSuccess: () => {
          lastPersistedDraftSignatureRef.current = submittedSignature;
          if (pendingDraftSignatureRef.current === submittedSignature) {
            pendingDraftSignatureRef.current = "";
          }
        },
        onError: (error) => {
          if (pendingDraftSignatureRef.current === submittedSignature) {
            pendingDraftSignatureRef.current = "";
          }
          messageBus.publishToast({
            intent: "warning",
            title: t("settings.calls.skills.security.saveFailed"),
            description: error instanceof Error ? error.message : String(error ?? ""),
          });
        },
      }
    );
  }, [draftSecuritySignature, normalizedDraft, skillsConfig, t, toolsConfig, updateSettings]);

  React.useEffect(() => {
    if (!dirty) {
      return;
    }
    if (
      shouldSkipDraftPersist({
        draftSignature: draftSecuritySignature,
        lastPersistedSignature: lastPersistedDraftSignatureRef.current,
        pendingSubmittedSignature: pendingDraftSignatureRef.current,
      })
    ) {
      return;
    }
    const timer = window.setTimeout(() => {
      if (updateSettings.isPending) {
        return;
      }
      persistSecurity();
    }, 500);
    return () => window.clearTimeout(timer);
  }, [dirty, draftSecuritySignature, persistSecurity, updateSettings.isPending]);

  React.useEffect(() => {
    if (!dirty && currentSecuritySignature === draftSecuritySignature) {
      lastPersistedDraftSignatureRef.current = draftSecuritySignature;
      if (pendingDraftSignatureRef.current === draftSecuritySignature) {
        pendingDraftSignatureRef.current = "";
      }
    }
  }, [currentSecuritySignature, dirty, draftSecuritySignature]);

  if (settingsQuery.isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-52 w-full" />
      </div>
    );
  }

  if (settingsQuery.isError) {
    return (
      <SettingsCompactListCard>
        <SettingsCompactRow
          label={t("settings.calls.skills.security.loadFailed")}
          labelClassName="text-destructive"
        >
          <Button variant="outline" size="compact" onClick={() => void settingsQuery.refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            {t("common.retry")}
          </Button>
        </SettingsCompactRow>
      </SettingsCompactListCard>
    );
  }

  const isSaving = updateSettings.isPending;
  const selectClassName = `${SKILLS_SELECT_TEXT_CLASS} ${SETTINGS_CONTROL_WIDTH_CLASS}`;
  const wideFieldClassName = SETTINGS_WIDE_CONTROL_WIDTH_CLASS;

  return (
    <div className="flex min-h-0 flex-1 flex-col space-y-3">
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        <SettingsCompactListCard>
          <div className="overflow-x-auto">
            <Table className="min-w-[42rem]">
              <TableHeader className="[&_tr]:border-b [&_tr]:border-border/70">
                <TableRow className="bg-muted/20 hover:bg-muted/20">
                  <TableHead className="h-auto w-1/5 px-3 py-2.5 normal-case tracking-normal">
                    {t("settings.calls.skills.security.actionGroup")}
                  </TableHead>
                  <TableHead className="h-auto w-3/5 px-3 py-2.5 normal-case tracking-normal">
                    {t("settings.calls.skills.security.coverage")}
                  </TableHead>
                  <TableHead className="h-auto w-[var(--app-settings-control-width)] min-w-[var(--app-settings-control-width)] px-3 py-2.5 normal-case tracking-normal">
                    {t("settings.calls.skills.security.modeLabel")}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="[&_tr:last-child]:border-b-0">
                {actionGroups.map((groupID) => {
                  const mode = (draft.actionModes[groupID] ?? "ask") as SkillsActionMode;
                  return (
                    <TableRow key={groupID} className="hover:bg-transparent">
                      <TableCell className="w-1/5 px-3 py-2.5 font-medium text-foreground">
                        {t(`settings.calls.skills.security.groups.${groupID}.label`)}
                      </TableCell>
                      <TableCell className="w-3/5 px-3 py-2.5 leading-5 text-muted-foreground">
                        {t(`settings.calls.skills.security.groups.${groupID}.description`)}
                      </TableCell>
                      <TableCell className="w-[var(--app-settings-control-width)] min-w-[var(--app-settings-control-width)] px-3 py-2.5">
                        <Select
                          value={mode}
                          onChange={(event) => updateActionMode(groupID, event.target.value as SkillsActionMode)}
                          disabled={isSaving}
                          className={selectClassName}
                        >
                          <option value="allow">{t("settings.calls.skills.security.mode.allow")}</option>
                          <option value="ask">{t("settings.calls.skills.security.mode.ask")}</option>
                          <option value="deny">{t("settings.calls.skills.security.mode.deny")}</option>
                        </Select>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </SettingsCompactListCard>

        <SettingsCompactListCard>
          <SettingsCompactRow label={t("settings.calls.skills.security.scannerMode")}>
            <Select
              value={draft.scannerMode}
              onChange={(event) =>
                setDraft((previous) => ({
                  ...previous,
                  scannerMode: event.target.value as SkillsSecurityConfig["scannerMode"],
                }))
              }
              disabled={isSaving}
              className={selectClassName}
            >
              <option value="off">{t("settings.calls.skills.security.scanner.off")}</option>
              <option value="warn">{t("settings.calls.skills.security.scanner.warn")}</option>
              <option value="block">{t("settings.calls.skills.security.scanner.block")}</option>
            </Select>
          </SettingsCompactRow>
          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.calls.skills.security.allowedBins")}>
            <Input
              value={allowedBinsInput}
              onChange={(event) => setAllowedBinsInput(event.target.value)}
              placeholder={t("settings.calls.skills.security.allowedBinsPlaceholder")}
              disabled={isSaving}
              size="compact"
              className={wideFieldClassName}
            />
          </SettingsCompactRow>
          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.calls.skills.security.allowForceInstall")}>
            <Switch
              checked={draft.allowForceInstall}
              onCheckedChange={(checked) =>
                setDraft((previous) => ({
                  ...previous,
                  allowForceInstall: checked === true,
                }))
              }
              disabled={isSaving}
            />
          </SettingsCompactRow>
          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.calls.skills.security.requireApproval")}>
            <Switch
              checked={draft.requireApproval}
              onCheckedChange={(checked) =>
                setDraft((previous) => ({
                  ...previous,
                  requireApproval: checked === true,
                }))
              }
              disabled={isSaving}
            />
          </SettingsCompactRow>
          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.calls.skills.security.allowExternalToolsOnly")}>
            <Switch
              checked={draft.allowExternalToolsOnly}
              onCheckedChange={(checked) =>
                setDraft((previous) => ({
                  ...previous,
                  allowExternalToolsOnly: checked === true,
                  allowGenericInstaller: checked === true ? false : previous.allowGenericInstaller,
                }))
              }
              disabled={isSaving}
            />
          </SettingsCompactRow>
          <SettingsCompactSeparator />
          <SettingsCompactRow label={t("settings.calls.skills.security.allowGenericInstaller")}>
            <Switch
              checked={draft.allowGenericInstaller}
              onCheckedChange={(checked) =>
                setDraft((previous) => ({
                  ...previous,
                  allowGenericInstaller: checked === true,
                }))
              }
              disabled={isSaving || draft.allowExternalToolsOnly}
            />
          </SettingsCompactRow>
        </SettingsCompactListCard>
      </div>
    </div>
  );
}
