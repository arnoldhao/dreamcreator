import * as React from "react";
import { Bot, LayoutGrid, Settings2, Sparkles } from "lucide-react";

import { useI18n } from "@/shared/i18n";
import { LoadingState } from "@/shared/ui/LoadingState";
import { SectionCard } from "@/shared/ui/SectionCard";
import { Card } from "@/shared/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useAssistants } from "@/shared/query/assistant";
import type { Assistant } from "@/shared/store/assistant";
import { consumePendingGatewayTarget } from "@/app/settings/sectionStorage";

import { GatewayCharacterPanel, type GatewayCharacterTab } from "./GatewayCharacterPanel";
import { GatewayDetailsPanel } from "./GatewayDetailsPanel";
import { GatewayStatusPanel } from "./GatewayStatusPanel";
import { GatewayAssistantPanel, type AssistantPanelTab } from "./GatewayAssistantPanel";
import type { AssistantParameterTab } from "./AssistantParametersPanel";
import { AssistantManagerPanel } from "./AssistantManagerPanel";
import { ChangeAssistantNameDialog } from "./ChangeAssistantNameDialog";

type GatewayTab = "status" | "details" | "character" | "assistant";

export function GatewaySection() {
  const { t } = useI18n();
  const assistantsQuery = useAssistants(true);
  const assistants = assistantsQuery.data ?? [];

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [view, setView] = React.useState<GatewayTab>("status");
  const [characterTab, setCharacterTab] = React.useState<GatewayCharacterTab>("avatar");
  const [assistantPanelTab, setAssistantPanelTab] = React.useState<AssistantPanelTab>("assistant");
  const [parameterTab, setParameterTab] = React.useState<AssistantParameterTab>("models");
  const [nameDialogOpen, setNameDialogOpen] = React.useState(false);
  const [nameTargetId, setNameTargetId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (selectedId && assistants.some((item) => item.id === selectedId)) {
      return;
    }
    if (assistants.length === 0) {
      setSelectedId(null);
      return;
    }
    const defaultAssistant = assistants.find((item) => item.isDefault) ?? assistants[0];
    setSelectedId(defaultAssistant?.id ?? null);
  }, [assistants, selectedId]);

  React.useEffect(() => {
    const pending = consumePendingGatewayTarget();
    if (!pending) {
      return;
    }
    if (pending.view) {
      setView(pending.view);
    }
    if (pending.panelTab) {
      setAssistantPanelTab(pending.panelTab);
    }
    if (pending.parameterTab) {
      setParameterTab(pending.parameterTab);
    }
    if (pending.characterTab) {
      setCharacterTab(pending.characterTab);
    }
  }, []);

  const currentAssistant: Assistant | null =
    assistants.find((item) => item.id === selectedId) ?? null;
  const nameTarget = assistants.find((item) => item.id === nameTargetId) ?? null;

  const handleOpenNameDialog = (id: string) => {
    setNameTargetId(id);
    setNameDialogOpen(true);
  };

  const handleOpenAssistant = (tab?: AssistantParameterTab) => {
    if (tab) {
      setParameterTab(tab);
      setAssistantPanelTab("parameters");
    }
    setView("assistant");
  };

  const handleOpenCharacter = (tab?: GatewayCharacterTab) => {
    if (tab) {
      setCharacterTab(tab);
    }
    setView("character");
  };

  if (assistantsQuery.isLoading) {
    return <LoadingState message={t("settings.gateway.loading")} />;
  }

  if (!currentAssistant) {
    return (
      <div className="space-y-4">
        <SectionCard
          title={t("settings.gateway.empty.title")}
          description={t("settings.gateway.empty.description")}
        >
          {null}
        </SectionCard>
        <Card className="flex min-h-0 flex-col h-[clamp(22rem,44vh,28rem)]">
          <AssistantManagerPanel
            assistants={assistants}
            currentAssistantId={selectedId}
            onSelectAssistant={setSelectedId}
            onOpenAvatar={() => {}}
            onOpenName={handleOpenNameDialog}
            active
          />
        </Card>
        <ChangeAssistantNameDialog
          open={nameDialogOpen}
          onOpenChange={setNameDialogOpen}
          assistant={nameTarget}
        />
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <Tabs
        value={view}
        onValueChange={(value) => setView(value as GatewayTab)}
        className="flex min-h-0 flex-1 flex-col gap-4"
      >
        <div className="flex flex-wrap items-center justify-center gap-2">
          <TabsList className="w-fit max-w-full justify-start overflow-x-auto overflow-y-hidden">
            <TabsTrigger value="status" className="min-w-0">
              <LayoutGrid className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{t("settings.gateway.status")}</span>
            </TabsTrigger>
            <TabsTrigger value="details" className="min-w-0">
              <Settings2 className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{t("settings.gateway.details.label")}</span>
            </TabsTrigger>
            <TabsTrigger value="character" className="min-w-0">
              <Sparkles className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{t("settings.gateway.character")}</span>
            </TabsTrigger>
            <TabsTrigger value="assistant" className="min-w-0">
              <Bot className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{t("settings.gateway.assistant")}</span>
            </TabsTrigger>
          </TabsList>
        </div>
        <TabsContent
          value="status"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <GatewayStatusPanel
            assistant={currentAssistant}
            assistants={assistants}
            onSelectAssistant={setSelectedId}
            onOpenCharacter={handleOpenCharacter}
            onOpenAssistant={handleOpenAssistant}
          />
        </TabsContent>
        <TabsContent
          value="details"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <GatewayDetailsPanel
            assistant={currentAssistant}
            assistants={assistants}
            currentAssistantId={selectedId}
            onSelectAssistant={setSelectedId}
          />
        </TabsContent>
        <TabsContent
          value="character"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <GatewayCharacterPanel
            assistant={currentAssistant}
            assistants={assistants}
            currentAssistantId={selectedId}
            activeTab={characterTab}
            onTabChange={setCharacterTab}
            onSelectAssistant={setSelectedId}
          />
        </TabsContent>
        <TabsContent
          value="assistant"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <GatewayAssistantPanel
            assistant={currentAssistant}
            assistants={assistants}
            currentAssistantId={selectedId}
            panelTab={assistantPanelTab}
            onPanelTabChange={setAssistantPanelTab}
            parameterTab={parameterTab}
            onParameterTabChange={setParameterTab}
            onSelectAssistant={setSelectedId}
            onOpenCharacter={handleOpenCharacter}
            onOpenName={handleOpenNameDialog}
          />
        </TabsContent>
      </Tabs>
      <ChangeAssistantNameDialog
        open={nameDialogOpen}
        onOpenChange={setNameDialogOpen}
        assistant={nameTarget}
      />
    </div>
  );
}
