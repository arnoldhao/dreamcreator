import * as React from "react";
import { HelpCircle } from "lucide-react";

import type { TalkConfigEntity, TTSConfigEntity, TTSStatusEntity } from "@/entities/voice";
import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import {
  controlClassName,
  panelClassName,
  parseVoiceAliases,
  rowClassName,
  rowLabelClassName,
} from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

const renderRowLabel = (label: string, description?: string) => (
  <div className="flex min-w-0 flex-1 items-center gap-1.5">
    <span className={rowLabelClassName} title={label}>
      {label}
    </span>
    {description ? (
      <TooltipProvider delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
              aria-label={label}
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top" align="start">
            {description}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ) : null}
  </div>
);

const renderRows = (rows: React.ReactNode[]) => (
  <div className={panelClassName}>
    {rows.map((row, index) => (
      <React.Fragment key={index}>
        {row}
        {index < rows.length - 1 ? <Separator /> : null}
      </React.Fragment>
    ))}
  </div>
);

interface GatewayTalkPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  talkTab: "talk" | "tts";
  onTalkTabChange: (value: "talk" | "tts") => void;
  talkDraft: TalkConfigEntity;
  setTalkDraft: React.Dispatch<React.SetStateAction<TalkConfigEntity>>;
  talkAliasesInput: string;
  setTalkAliasesInput: React.Dispatch<React.SetStateAction<string>>;
  commitTalkConfig: (next: TalkConfigEntity) => Promise<void>;
  ttsStatus: TTSStatusEntity | undefined;
  ttsDraft: TTSConfigEntity;
  setTtsDraft: React.Dispatch<React.SetStateAction<TTSConfigEntity>>;
  commitTTSConfig: (next: TTSConfigEntity) => Promise<void>;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
}

export function GatewayTalkPanel({
  t,
  gateway,
  isDisabled,
  talkTab,
  onTalkTabChange,
  talkDraft,
  setTalkDraft,
  talkAliasesInput,
  setTalkAliasesInput,
  commitTalkConfig,
  ttsStatus,
  ttsDraft,
  setTtsDraft,
  commitTTSConfig,
  updateGateway,
}: GatewayTalkPanelProps) {
  const ttsProviders = ttsStatus?.providers ?? [];
  const hasDraftProvider =
    Boolean(ttsDraft.providerId) &&
    !ttsProviders.some((provider) => provider.providerId === ttsDraft.providerId);
  const ttsFormats = ["wav", "ogg", "mp3"];
  const normalizedTtsFormat = (ttsDraft.format ?? "").trim().toLowerCase();
  if (normalizedTtsFormat && !ttsFormats.includes(normalizedTtsFormat)) {
    ttsFormats.unshift(normalizedTtsFormat);
  }

  const talkRows = [
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.voice.enabled"))}
      <Switch
        checked={gateway?.voiceEnabled ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) => updateGateway({ voiceEnabled: value })}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.talk.voiceId"))}
      <Input
        value={talkDraft.voiceId ?? ""}
        onChange={(event) => setTalkDraft((prev) => ({ ...prev, voiceId: event.target.value }))}
        onBlur={(event) => commitTalkConfig({ ...talkDraft, voiceId: event.target.value })}
        placeholder="voice-id"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.talk.voiceAliases"),
        t("settings.gateway.detailsPanel.talk.voiceAliasesDescription")
      )}
      <Input
        value={talkAliasesInput}
        onChange={(event) => setTalkAliasesInput(event.target.value)}
        onBlur={(event) =>
          commitTalkConfig({ ...talkDraft, voiceAliases: parseVoiceAliases(event.target.value) })
        }
        placeholder="friendly=voice-id, narrator=voice-id"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.talk.modelId"))}
      <Input
        value={talkDraft.modelId ?? ""}
        onChange={(event) => setTalkDraft((prev) => ({ ...prev, modelId: event.target.value }))}
        onBlur={(event) => commitTalkConfig({ ...talkDraft, modelId: event.target.value })}
        placeholder="model-id"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.talk.outputFormat"),
        t("settings.gateway.detailsPanel.talk.outputFormatDescription")
      )}
      <Input
        value={talkDraft.outputFormat ?? ""}
        onChange={(event) => setTalkDraft((prev) => ({ ...prev, outputFormat: event.target.value }))}
        onBlur={(event) => commitTalkConfig({ ...talkDraft, outputFormat: event.target.value })}
        placeholder="mp3_44100_128"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.talk.apiKey"),
        t("settings.gateway.detailsPanel.talk.apiKeyDescription")
      )}
      <Input
        type="password"
        value={talkDraft.apiKey ?? ""}
        onChange={(event) => setTalkDraft((prev) => ({ ...prev, apiKey: event.target.value }))}
        onBlur={(event) => commitTalkConfig({ ...talkDraft, apiKey: event.target.value })}
        placeholder="elevenlabs-api-key"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.talk.interruptOnSpeech"),
        t("settings.gateway.detailsPanel.talk.interruptOnSpeechDescription")
      )}
      <Switch
        checked={talkDraft.interruptOnSpeech ?? true}
        onCheckedChange={(value) => commitTalkConfig({ ...talkDraft, interruptOnSpeech: value })}
      />
    </div>,
  ];

  const ttsRows = [
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.tts.status"))}
      <Badge variant={ttsStatus?.enabled ? "default" : "secondary"}>
        {ttsStatus?.enabled
          ? t("settings.gateway.detailsPanel.tts.enabled")
          : t("settings.gateway.detailsPanel.tts.disabled")}
      </Badge>
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.tts.provider"),
        t("settings.gateway.detailsPanel.tts.providerDescription")
      )}
      <Select
        value={ttsDraft.providerId ?? ""}
        className={controlClassName}
        onChange={(event) => {
          const next = { ...ttsDraft, providerId: event.target.value };
          setTtsDraft(next);
          void commitTTSConfig(next);
        }}
      >
        <option value="">{t("settings.gateway.detailsPanel.tts.providerAuto")}</option>
        {ttsProviders.map((provider) => (
          <option
            key={provider.providerId}
            value={provider.providerId}
            disabled={!provider.available}
          >
            {provider.displayName}
          </option>
        ))}
        {hasDraftProvider ? <option value={ttsDraft.providerId}>{ttsDraft.providerId}</option> : null}
      </Select>
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.tts.voiceId"))}
      <Input
        value={ttsDraft.voiceId ?? ""}
        onChange={(event) => setTtsDraft((prev) => ({ ...prev, voiceId: event.target.value }))}
        onBlur={(event) => void commitTTSConfig({ ...ttsDraft, voiceId: event.target.value })}
        placeholder="voice-id"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.tts.modelId"))}
      <Input
        value={ttsDraft.modelId ?? ""}
        onChange={(event) => setTtsDraft((prev) => ({ ...prev, modelId: event.target.value }))}
        onBlur={(event) => void commitTTSConfig({ ...ttsDraft, modelId: event.target.value })}
        placeholder="model-id"
        size="compact"
        className={controlClassName}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.tts.format"))}
      <Select
        value={normalizedTtsFormat}
        className={controlClassName}
        onChange={(event) => {
          const next = { ...ttsDraft, format: event.target.value };
          setTtsDraft(next);
          void commitTTSConfig(next);
        }}
      >
        <option value="">{t("settings.gateway.detailsPanel.tts.formatAuto")}</option>
        {ttsFormats.map((format) => (
          <option key={format} value={format}>
            {format.toUpperCase()}
          </option>
        ))}
      </Select>
    </div>,
  ];

  return (
    <Tabs value={talkTab} onValueChange={(value) => onTalkTabChange(value as "talk" | "tts")}>
      <div className="flex justify-center">
        <TabsList className="w-fit">
          <TabsTrigger value="talk" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.talkTabs.talk")}
          </TabsTrigger>
          <TabsTrigger value="tts" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.talkTabs.tts")}
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="talk">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderRows(talkRows)}
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="tts">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderRows(ttsRows)}
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}

interface GatewayVoiceWakePanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  wakeInput: string;
  setWakeInput: React.Dispatch<React.SetStateAction<string>>;
  commitVoiceWake: (value: string) => Promise<void>;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
}

export function GatewayVoiceWakePanel({
  t,
  gateway,
  isDisabled,
  wakeInput,
  setWakeInput,
  commitVoiceWake,
  updateGateway,
}: GatewayVoiceWakePanelProps) {
  const wakeDescription = t("settings.gateway.detailsPanel.voiceWakeTriggers.description");

  return renderRows([
    <div className={rowClassName}>
      {renderRowLabel(t("settings.gateway.detailsPanel.voiceWake.enabled"))}
      <Switch
        checked={gateway?.voiceWakeEnabled ?? false}
        disabled={isDisabled}
        onCheckedChange={(value) => updateGateway({ voiceWakeEnabled: value })}
      />
    </div>,
    <div className={rowClassName}>
      {renderRowLabel(
        t("settings.gateway.detailsPanel.voiceWakeTriggers.label"),
        wakeDescription
      )}
      <Input
        value={wakeInput}
        onChange={(event) => setWakeInput(event.target.value)}
        onBlur={(event) => {
          void commitVoiceWake(event.target.value);
        }}
        placeholder="hey dreamcreator"
        size="compact"
        className={controlClassName}
      />
    </div>,
  ]);
}
