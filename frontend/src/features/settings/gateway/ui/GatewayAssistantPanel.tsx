import * as React from "react";

import { Card } from "@/shared/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import type { Assistant } from "@/shared/store/assistant";

import { AssistantManagerPanel } from "./AssistantManagerPanel";
import { AssistantParametersPanel, type AssistantParameterTab } from "./AssistantParametersPanel";

export type AssistantPanelTab = "assistant" | "parameters";

interface GatewayAssistantPanelProps {
  assistant: Assistant;
  assistants: Assistant[];
  currentAssistantId: string | null;
  panelTab?: AssistantPanelTab;
  onPanelTabChange?: (tab: AssistantPanelTab) => void;
  parameterTab: AssistantParameterTab;
  onParameterTabChange: (tab: AssistantParameterTab) => void;
  onSelectAssistant: (id: string) => void;
  onOpenCharacter: (tab?: "avatar" | "motion") => void;
  onOpenName: (id: string) => void;
}

export function GatewayAssistantPanel({
  assistant,
  assistants,
  currentAssistantId,
  panelTab,
  onPanelTabChange,
  parameterTab,
  onParameterTabChange,
  onSelectAssistant,
  onOpenCharacter,
  onOpenName,
}: GatewayAssistantPanelProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = React.useState<AssistantPanelTab>(panelTab ?? "assistant");

  React.useEffect(() => {
    if (panelTab) {
      setActiveTab(panelTab);
    }
  }, [panelTab]);

  const sidebarTabs = (
    <TabsList className="grid h-8 w-full grid-cols-2 p-0.5">
      <TabsTrigger
        value="assistant"
        className="h-full py-0 leading-none data-[state=active]:shadow-none"
      >
        {t("settings.gateway.assistantTab.assistant")}
      </TabsTrigger>
      <TabsTrigger
        value="parameters"
        className="h-full py-0 leading-none data-[state=active]:shadow-none"
      >
        {t("settings.gateway.assistantTab.parameters")}
      </TabsTrigger>
    </TabsList>
  );

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => {
        if (value === "assistant" || value === "parameters") {
          setActiveTab(value);
          onPanelTabChange?.(value);
        }
      }}
      className="flex min-h-0 flex-1 flex-col gap-4"
    >
      <TabsContent
        value="assistant"
        className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1"
      >
        <Card className="flex min-h-0 flex-1 overflow-hidden">
          <AssistantManagerPanel
            assistants={assistants}
            currentAssistantId={currentAssistantId}
            onSelectAssistant={onSelectAssistant}
            onOpenAvatar={(id) => {
              onSelectAssistant(id);
              onOpenCharacter("avatar");
            }}
            onOpenName={onOpenName}
            active={activeTab === "assistant"}
            sidebarHeader={sidebarTabs}
          />
        </Card>
      </TabsContent>

      <TabsContent
        value="parameters"
        className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1"
      >
        <AssistantParametersPanel
          assistant={assistant}
          assistants={assistants}
          initialTab={parameterTab}
          onTabChange={onParameterTabChange}
          onSelectAssistant={onSelectAssistant}
          sidebarHeader={sidebarTabs}
        />
      </TabsContent>
    </Tabs>
  );
}
