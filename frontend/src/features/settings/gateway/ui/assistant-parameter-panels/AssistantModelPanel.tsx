import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import type { AssistantModel } from "@/shared/store/assistant";

import {
  fieldClassName,
  fieldNumberClassName,
  modelCardClassName,
  modelRowLabelClassName,
  panelClassName,
  rowClassName,
} from "./constants";
import {
  MODEL_MISSING_VALUE_PREFIX,
  modelRefEquals,
  modelRefKey,
  resolveModelSelectValue,
  type AssistantModelOption,
} from "./model-utils";

type Translate = (key: string) => string;

export type ResolvedModelSpec = {
  inherit: boolean;
  primary: string;
  fallbacks: string[];
  stream: boolean;
  temperature: number;
  maxTokens: number;
};

interface AssistantModelPanelProps {
  t: Translate;
  activeModelTab: "agent" | "embedding" | "image";
  onActiveModelTabChange: (tab: "agent" | "embedding" | "image") => void;
  model: AssistantModel;
  agentSpec: ResolvedModelSpec;
  embeddingSpec: ResolvedModelSpec;
  imageSpec: ResolvedModelSpec;
  agentModelOptions: AssistantModelOption[];
  embeddingModelOptions: AssistantModelOption[];
  imageModelOptions: AssistantModelOption[];
  onUpdateModel: (next: AssistantModel, commit?: boolean) => void;
}

export function AssistantModelPanel({
  t,
  activeModelTab,
  onActiveModelTabChange,
  model,
  agentSpec,
  embeddingSpec,
  imageSpec,
  agentModelOptions,
  embeddingModelOptions,
  imageModelOptions,
  onUpdateModel,
}: AssistantModelPanelProps) {
  const resolveModelRefFromValue = (value: string, options: AssistantModelOption[]) => {
    if (!value) {
      return "";
    }
    if (value.startsWith(MODEL_MISSING_VALUE_PREFIX)) {
      return value.slice(MODEL_MISSING_VALUE_PREFIX.length).trim();
    }
    return options.find((option) => option.value === value)?.modelRef ?? "";
  };

  const renderConfigPanel = (
    target: "agent" | "embedding" | "image",
    spec: ResolvedModelSpec,
    options: AssistantModelOption[]
  ) => {
    const supportsInherit = target === "embedding" || target === "image";
    const isInherited = supportsInherit && spec.inherit;
    const normalizedPrimary = spec.primary.trim();
    const normalizedFallbacks = (spec.fallbacks ?? []).map((item) => item.trim()).filter(Boolean);
    const selectedPrimaryValue = resolveModelSelectValue(normalizedPrimary, options) || options[0]?.value || "";
    const selectedKeys = new Set(
      [modelRefKey(normalizedPrimary), ...normalizedFallbacks.map((item) => modelRefKey(item))].filter(Boolean)
    );
    const addableFallbackOptions = options.filter((option) => {
      const key = modelRefKey(option.modelRef);
      return key && !selectedKeys.has(key);
    });

    const commitSpec = (next: ResolvedModelSpec) => {
      onUpdateModel({ ...model, [target]: next }, true);
    };

    const handleInheritChange = (value: boolean) => {
      commitSpec({ ...spec, inherit: value });
    };

    const handlePrimaryChange = (value: string) => {
      const nextPrimary = resolveModelRefFromValue(value, options);
      commitSpec({ ...spec, primary: nextPrimary });
    };

    const handleAddFallback = () => {
      const option = addableFallbackOptions[0];
      if (!option) {
        return;
      }
      commitSpec({ ...spec, fallbacks: [...normalizedFallbacks, option.modelRef] });
    };

    const handleFallbackChange = (index: number, value: string) => {
      const nextRef = resolveModelRefFromValue(value, options);
      if (!nextRef) {
        return;
      }
      const nextFallbacks = [...normalizedFallbacks];
      nextFallbacks[index] = nextRef;
      const deduped = nextFallbacks.filter(
        (item, itemIndex) => item && nextFallbacks.findIndex((entry) => modelRefEquals(entry, item)) === itemIndex
      );
      commitSpec({ ...spec, fallbacks: deduped });
    };

    const handleFallbackRemove = (index: number) => {
      const nextFallbacks = normalizedFallbacks.filter((_, itemIndex) => itemIndex !== index);
      commitSpec({ ...spec, fallbacks: nextFallbacks });
    };

    const buildFallbackOptions = (currentRef: string, index: number) => {
      const excluded = new Set(
        [
          modelRefKey(normalizedPrimary),
          ...normalizedFallbacks
            .filter((_, itemIndex) => itemIndex !== index)
            .map((item) => modelRefKey(item)),
        ].filter(Boolean)
      );
      const rows = options
        .filter((option) => {
          const key = modelRefKey(option.modelRef);
          if (!key) {
            return false;
          }
          return !excluded.has(key) || modelRefEquals(option.modelRef, currentRef);
        })
        .map((option) => ({ value: option.value, label: option.label }));
      if (currentRef && !rows.some((row) => modelRefEquals(resolveModelRefFromValue(row.value, options), currentRef))) {
        rows.unshift({
          value: `${MODEL_MISSING_VALUE_PREFIX}${currentRef}`,
          label: currentRef,
        });
      }
      return rows;
    };

    return (
      <Card className={modelCardClassName}>
        <CardContent size="compact" className="space-y-3 p-2">
          {supportsInherit ? (
            <>
              <div className={rowClassName}>
                <div className={modelRowLabelClassName}>
                  {t("settings.gateway.model.inheritAgent")}
                </div>
                <Switch checked={isInherited} onCheckedChange={handleInheritChange} />
              </div>
              {isInherited ? (
                <div className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5 text-xs text-muted-foreground">
                  {t("settings.gateway.model.inheritAgentHint")}
                </div>
              ) : (
                <Separator />
              )}
            </>
          ) : null}

          {isInherited ? null : (
            <>
          <div className={rowClassName}>
            <div className={modelRowLabelClassName}>
              {t("settings.gateway.model.primary")}
            </div>
            <Select
              value={selectedPrimaryValue}
              className={fieldClassName}
              onChange={(event) => handlePrimaryChange(event.target.value)}
            >
              {options.length === 0 ? <option value="">{t("settings.gateway.model.selectPrimary")}</option> : null}
              {options.map((option) => (
                <option key={`${target}-${option.value}`} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          </div>

          <Separator />

          <div className="space-y-2">
            <div className="flex items-start justify-between gap-3">
              <div className={modelRowLabelClassName}>
                {t("settings.gateway.model.fallbacks")}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="compact"
                onClick={handleAddFallback}
                disabled={addableFallbackOptions.length === 0}
              >
                <Plus className="mr-1 h-4 w-4" />
                {t("settings.gateway.model.fallbackAdd")}
              </Button>
            </div>
            <div className="overflow-hidden rounded-md border border-border/60">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-full truncate">
                      {t("settings.gateway.model.fallbackModel")}
                    </TableHead>
                    <TableHead className="w-20 truncate text-right">
                      {t("settings.gateway.model.fallbackActions")}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {normalizedFallbacks.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={2} className="text-xs text-muted-foreground">
                        {t("settings.gateway.model.fallbackEmpty")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    normalizedFallbacks.map((fallbackRef, index) => {
                      const fallbackOptions = buildFallbackOptions(fallbackRef, index);
                      return (
                        <TableRow key={`${target}-fallback-${index}`}>
                          <TableCell>
                            <Select
                              value={resolveModelSelectValue(fallbackRef, options)}
                              className="w-full"
                              onChange={(event) => handleFallbackChange(index, event.target.value)}
                            >
                              {fallbackOptions.map((option) => (
                                <option key={`${target}-${index}-${option.value}`} value={option.value}>
                                  {option.label}
                                </option>
                              ))}
                            </Select>
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              type="button"
                              variant="ghost"
                              size="compactIcon"
                              onClick={() => handleFallbackRemove(index)}
                              aria-label={t("settings.gateway.model.fallbackRemove")}
                              title={t("settings.gateway.model.fallbackRemove")}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>
            </div>
          </div>

          <Separator />

          <div className={rowClassName}>
            <div className={modelRowLabelClassName}>
              {t("settings.gateway.model.stream")}
            </div>
            <Switch
              checked={spec.stream}
              onCheckedChange={(value) => onUpdateModel({ ...model, [target]: { ...spec, stream: value } }, true)}
            />
          </div>

          <Separator />

          <div className={rowClassName}>
            <div className={modelRowLabelClassName}>
              {t("settings.gateway.model.temperature")}
            </div>
            <Input
              type="number"
              value={Number.isFinite(spec.temperature) ? spec.temperature : 0}
              onChange={(event) => {
                const parsed = Number(event.target.value);
                if (!Number.isNaN(parsed)) {
                  onUpdateModel({ ...model, [target]: { ...spec, temperature: parsed } });
                }
              }}
              onBlur={() => onUpdateModel({ ...model, [target]: spec }, true)}
              size="compact"
              className={fieldNumberClassName}
            />
          </div>

          <Separator />

          <div className={rowClassName}>
            <div className={modelRowLabelClassName}>
              {t("settings.gateway.model.maxTokens")}
            </div>
            <Input
              type="number"
              value={Number.isFinite(spec.maxTokens) ? spec.maxTokens : 0}
              onChange={(event) => {
                const parsed = Number(event.target.value);
                if (!Number.isNaN(parsed)) {
                  onUpdateModel({ ...model, [target]: { ...spec, maxTokens: Math.trunc(parsed) } });
                }
              }}
              onBlur={() => onUpdateModel({ ...model, [target]: spec }, true)}
              size="compact"
              className={fieldNumberClassName}
            />
          </div>
            </>
          )}
        </CardContent>
      </Card>
    );
  };

  return (
    <div className={panelClassName}>
      <Tabs
        value={activeModelTab}
        onValueChange={(value) => onActiveModelTabChange(value as typeof activeModelTab)}
      >
        <div className="flex justify-center">
          <TabsList className="w-auto">
            <TabsTrigger value="agent">
              {t("settings.gateway.model.agentTitle")}
            </TabsTrigger>
            <TabsTrigger value="embedding">
              {t("settings.gateway.model.embeddingTitle")}
            </TabsTrigger>
            <TabsTrigger value="image">
              {t("settings.gateway.model.imageTitle")}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="agent" className="mt-3 space-y-2">
          {renderConfigPanel("agent", agentSpec, agentModelOptions)}
        </TabsContent>
        <TabsContent value="embedding" className="mt-3 space-y-2">
          {renderConfigPanel("embedding", embeddingSpec, embeddingModelOptions)}
        </TabsContent>
        <TabsContent value="image" className="mt-3 space-y-2">
          {renderConfigPanel("image", imageSpec, imageModelOptions)}
        </TabsContent>
      </Tabs>
    </div>
  );
}
