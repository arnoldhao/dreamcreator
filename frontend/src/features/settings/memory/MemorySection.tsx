import { Brain, ListTree, SlidersHorizontal } from "lucide-react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import { useMemorySummary } from "@/shared/query/memory";
import { useUpdateSettings } from "@/shared/query/settings";
import type { MemorySettings, UpdateMemorySettingsRequest } from "@/shared/contracts/settings";

import type { MemoryTabId } from "./types";
import { SummaryTab } from "./tabs/SummaryTab";
import { EntriesTab } from "./tabs/EntriesTab";
import { ConfigTab } from "./tabs/ConfigTab";

interface MemorySectionProps {
  tab: MemoryTabId;
  onTabChange: (tab: MemoryTabId) => void;
  memory: MemorySettings;
}

export function MemorySection({ tab, onTabChange, memory }: MemorySectionProps) {
  const { t, language } = useI18n();
  const updateSettings = useUpdateSettings();
  const summary = useMemorySummary();

  const patchMemory = (patch: UpdateMemorySettingsRequest) => {
    updateSettings.mutate({ memory: patch });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <Tabs value={tab} onValueChange={(value) => onTabChange(value as MemoryTabId)} className="flex min-h-0 flex-1 flex-col space-y-5">
        <div className="flex justify-center">
          <TabsList className="w-fit max-w-full justify-start overflow-x-auto overflow-y-hidden">
            <TabsTrigger value="summary" className="min-w-0">
              <Brain className="h-4 w-4" />
              <span className="truncate">{t("settings.memory.tabs.summary")}</span>
            </TabsTrigger>
            <TabsTrigger value="entries" className="min-w-0">
              <ListTree className="h-4 w-4" />
              <span className="truncate">{t("settings.memory.tabs.entries")}</span>
            </TabsTrigger>
            <TabsTrigger value="config" className="min-w-0">
              <SlidersHorizontal className="h-4 w-4" />
              <span className="truncate">{t("settings.memory.tabs.config")}</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {tab === "summary" ? (
          <TabsContent value="summary" className="mt-0">
            <SummaryTab
              t={t}
              language={language}
              summary={summary.data}
              isLoading={summary.isLoading}
              onRefresh={() => {
                void summary.refetch();
              }}
            />
          </TabsContent>
        ) : null}

        {tab === "entries" ? (
          <TabsContent value="entries" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
            <EntriesTab t={t} language={language} />
          </TabsContent>
        ) : null}

        {tab === "config" ? (
          <TabsContent value="config" className="mt-0">
            <ConfigTab t={t} memory={memory} onPatch={patchMemory} />
          </TabsContent>
        ) : null}
      </Tabs>
    </div>
  );
}
