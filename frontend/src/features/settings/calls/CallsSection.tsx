import { Hammer, Sparkles } from "lucide-react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import { CallsSkillsTab } from "./components/CallsSkillsTab";
import { CallsToolsTab } from "./components/CallsToolsTab";
import type { CallsTabId } from "./types";

export function CallsSection({ tab, onTabChange }: { tab: CallsTabId; onTabChange: (tab: CallsTabId) => void }) {
  const { t } = useI18n();

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <Tabs
        value={tab}
        onValueChange={(value) => onTabChange(value as CallsTabId)}
        className="flex min-h-0 flex-1 flex-col space-y-5"
      >
        <div className="flex justify-center">
          <TabsList className="w-fit max-w-full justify-start overflow-x-auto overflow-y-hidden">
            <TabsTrigger value="tools" className="min-w-0">
              <Hammer className="h-4 w-4" />
              <span className="truncate">{t("app.settings.title.tools")}</span>
            </TabsTrigger>
            <TabsTrigger value="skills" className="min-w-0">
              <Sparkles className="h-4 w-4" />
              <span className="truncate">{t("app.settings.title.skills")}</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {tab === "tools" ? (
          <TabsContent
            value="tools"
            className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
          >
            <CallsToolsTab />
          </TabsContent>
        ) : null}

        <CallsSkillsTab embedded />
      </Tabs>
    </div>
  );
}
