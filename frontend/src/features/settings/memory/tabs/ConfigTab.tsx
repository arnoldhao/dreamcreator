import * as React from "react";
import { ArrowUpRight } from "lucide-react";
import { Events } from "@wailsio/runtime";

import { setPendingGatewayTarget } from "@/app/settings/sectionStorage";
import { useAssistants } from "@/shared/query/assistant";
import { useEnabledProvidersWithModels, useProviders } from "@/shared/query/providers";
import type { MemorySettings, UpdateMemorySettingsRequest } from "@/shared/contracts/settings";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import {
  SETTINGS_CONTROL_WIDTH_CLASS,
  SETTINGS_WIDE_CONTROL_WIDTH_CLASS,
  SettingsCompactListCard,
  SettingsCompactRow,
  SettingsCompactSeparator,
} from "@/shared/ui/settings-layout";
import { Switch } from "@/shared/ui/switch";

interface ConfigTabProps {
  t: (key: string) => string;
  memory: MemorySettings;
  onPatch: (patch: UpdateMemorySettingsRequest) => void;
}

export function ConfigTab({ t, memory, onPatch }: ConfigTabProps) {
  const assistantsQuery = useAssistants(true);
  const providersCatalogQuery = useProviders();
  const providersQuery = useEnabledProvidersWithModels();
  const assistants = assistantsQuery.data ?? [];
  const providerCatalog = providersCatalogQuery.data ?? [];
  const providerEntries = providersQuery.data ?? [];
  const defaultAssistant = assistants.find((item) => item.isDefault) ?? assistants[0];

  const extractionModelRef = pickModelRef(
    defaultAssistant?.model?.agent?.primary,
    defaultAssistant?.model?.agent?.fallbacks
  );
  const embeddingInherit = defaultAssistant?.model?.embedding?.inherit ?? true;
  const embeddingModelRef = embeddingInherit
    ? extractionModelRef
    : pickModelRef(defaultAssistant?.model?.embedding?.primary, defaultAssistant?.model?.embedding?.fallbacks);

  const extractionModel = formatModelRef(extractionModelRef, providerEntries, providerCatalog);
  const embeddingModel = formatModelRef(embeddingModelRef, providerEntries, providerCatalog);

  const extractionValue = extractionModel || t("settings.memory.config.model.unconfigured");
  const embeddingBase = embeddingModel || t("settings.memory.config.model.unconfigured");
  const embeddingValue = embeddingInherit
    ? `${embeddingBase} · ${t("settings.memory.config.model.inherited")}`
    : embeddingBase;

  const fieldClassName = `${SETTINGS_CONTROL_WIDTH_CLASS} text-right`;
  const modelFieldClassName = `${SETTINGS_WIDE_CONTROL_WIDTH_CLASS} flex min-w-0 items-center justify-end gap-2`;

  const openAssistantModels = React.useCallback(() => {
    setPendingGatewayTarget({ view: "assistant", panelTab: "parameters", parameterTab: "models" });
    Events.Emit("settings:navigate", "gateway");
  }, []);

  const didForceEnableRef = React.useRef(false);
  React.useEffect(() => {
    if (memory.enabled || didForceEnableRef.current) {
      return;
    }
    didForceEnableRef.current = true;
    onPatch({ enabled: true });
  }, [memory.enabled, onPatch]);

  return (
    <div className="space-y-3">
      <SettingsCompactListCard>
        <SettingsCompactRow label={t("settings.memory.summary.enabled")}>
          <Switch checked disabled />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.config.model.extract")} contentClassName="min-w-0">
          <div className={modelFieldClassName}>
            <span className="min-w-0 flex-1 truncate text-right text-foreground">{extractionValue}</span>
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className="h-7 w-7 shrink-0"
              onClick={openAssistantModels}
              aria-label={t("settings.memory.config.model.openAssistant")}
              title={t("settings.memory.config.model.openAssistant")}
            >
              <ArrowUpRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.config.model.embedding")} contentClassName="min-w-0">
          <div className={modelFieldClassName}>
            <span className="min-w-0 flex-1 truncate text-right text-foreground">{embeddingValue}</span>
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className="h-7 w-7 shrink-0"
              onClick={openAssistantModels}
              aria-label={t("settings.memory.config.model.openAssistant")}
              title={t("settings.memory.config.model.openAssistant")}
            >
              <ArrowUpRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        </SettingsCompactRow>
      </SettingsCompactListCard>

      <SettingsCompactListCard>
        <SettingsCompactRow label={t("settings.memory.retrieval.autoRecall")}>
          <Switch checked={memory.autoRecall} onCheckedChange={(checked) => onPatch({ autoRecall: checked })} />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.recallTopK")}>
          <Input
            id="memory-recall-topk"
            type="number"
            min={1}
            max={50}
            value={memory.recallTopK}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ recallTopK: parsed });
              }
            }}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.minScore")}>
          <Input
            id="memory-min-score"
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={memory.minScore}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ minScore: parsed });
              }
            }}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.vectorWeight")}>
          <Input
            id="memory-vector-weight"
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={memory.vectorWeight}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ vectorWeight: parsed });
              }
            }}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.textWeight")}>
          <Input
            id="memory-text-weight"
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={memory.textWeight}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ textWeight: parsed });
              }
            }}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.recencyWeight")}>
          <Input
            id="memory-recency-weight"
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={memory.recencyWeight}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ recencyWeight: parsed });
              }
            }}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.retrieval.recencyHalfLifeDays")}>
          <Input
            id="memory-recency-half-life"
            type="number"
            min={1}
            max={365}
            step={1}
            value={memory.recencyHalfLifeDays}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ recencyHalfLifeDays: parsed });
              }
            }}
          />
        </SettingsCompactRow>
      </SettingsCompactListCard>

      <SettingsCompactListCard>
        <SettingsCompactRow label={t("settings.memory.lifecycle.autoCapture")}>
          <Switch checked={memory.autoCapture} onCheckedChange={(checked) => onPatch({ autoCapture: checked })} />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.lifecycle.sessionLifecycle")}>
          <Switch
            checked={memory.sessionLifecycle}
            onCheckedChange={(checked) => onPatch({ sessionLifecycle: checked })}
          />
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.memory.lifecycle.captureMaxEntries")}>
          <Input
            id="memory-capture-max-entries"
            type="number"
            min={1}
            max={20}
            value={memory.captureMaxEntries}
            size="compact"
            className={fieldClassName}
            onChange={(event) => {
              const parsed = Number(event.target.value);
              if (!Number.isNaN(parsed)) {
                onPatch({ captureMaxEntries: parsed });
              }
            }}
          />
        </SettingsCompactRow>
      </SettingsCompactListCard>
    </div>
  );
}

function pickModelRef(primary?: string, fallbacks?: string[]): string {
  const normalizedPrimary = (primary ?? "").trim();
  if (normalizedPrimary) {
    return normalizedPrimary;
  }
  for (const item of fallbacks ?? []) {
    const normalized = item.trim();
    if (normalized) {
      return normalized;
    }
  }
  return "";
}

function formatModelRef(
  modelRef: string,
  providers: Array<{
    provider: { id: string; name: string };
    models: Array<{ name: string; displayName?: string }>;
  }>,
  providerCatalog: Array<{ id: string; name: string }>
): string {
  const trimmed = modelRef.trim();
  if (!trimmed) {
    return "";
  }
  const parsed = parseModelRef(trimmed);
  if (!parsed.providerId || !parsed.modelName) {
    return trimmed;
  }
  const provider = providers.find((item) => item.provider.id === parsed.providerId);
  const providerNameFromCatalog = providerCatalog.find((item) => item.id === parsed.providerId)?.name?.trim();
  const providerName = provider?.provider.name?.trim() || providerNameFromCatalog || "";
  const model = provider?.models.find((item) => item.name === parsed.modelName);
  const modelName = model?.displayName?.trim() || parsed.modelName;
  if (!providerName) {
    return modelName;
  }
  return `${providerName} / ${modelName}`;
}

function parseModelRef(value: string): { providerId: string; modelName: string } {
  const trimmed = value.trim();
  if (!trimmed) {
    return { providerId: "", modelName: "" };
  }
  const slashIndex = trimmed.indexOf("/");
  if (slashIndex > 0) {
    return {
      providerId: trimmed.slice(0, slashIndex).trim(),
      modelName: trimmed.slice(slashIndex + 1).trim(),
    };
  }
  const colonIndex = trimmed.indexOf(":");
  if (colonIndex > 0) {
    return {
      providerId: trimmed.slice(0, colonIndex).trim(),
      modelName: trimmed.slice(colonIndex + 1).trim(),
    };
  }
  return { providerId: "", modelName: trimmed };
}
