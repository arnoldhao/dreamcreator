import { Link2, Plug2, Wrench } from "lucide-react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";
import { ConnectorsSection } from "@/features/settings/connectors";
import { ExternalToolsSection } from "@/features/settings/external-tools";
import { ChannelsSection } from "./ChannelsSection";

export type IntegrationTabId = "channel" | "connector" | "external-tool";

interface IntegrationSectionProps {
  tab: IntegrationTabId;
  onTabChange: (tab: IntegrationTabId) => void;
  showChannelTab?: boolean;
}

export function IntegrationSection({ tab, onTabChange, showChannelTab = true }: IntegrationSectionProps) {
  const { t } = useI18n();
  const activeTab = !showChannelTab && tab === "channel" ? "connector" : tab;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <Tabs
        value={activeTab}
        onValueChange={(value) => onTabChange(value as IntegrationTabId)}
        className="flex min-h-0 flex-1 flex-col space-y-5"
      >
        <div className="flex justify-center">
          <TabsList className="w-fit max-w-full justify-start overflow-x-auto overflow-y-hidden">
            {showChannelTab ? (
              <TabsTrigger value="channel" className="min-w-0">
                <Plug2 className="h-4 w-4" />
                <span className="truncate">{t("app.settings.title.channels")}</span>
              </TabsTrigger>
            ) : null}
            <TabsTrigger value="connector" className="min-w-0">
              <Link2 className="h-4 w-4" />
              <span className="truncate">{t("app.settings.title.connectors")}</span>
            </TabsTrigger>
            <TabsTrigger value="external-tool" className="min-w-0">
              <Wrench className="h-4 w-4" />
              <span className="truncate">{t("app.settings.title.external-tools")}</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {showChannelTab ? (
          <TabsContent
            value="channel"
            className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
          >
            <ChannelsSection />
          </TabsContent>
        ) : null}
        <TabsContent
          value="connector"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <ConnectorsSection />
        </TabsContent>
        <TabsContent
          value="external-tool"
          className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
        >
          <ExternalToolsSection />
        </TabsContent>
      </Tabs>
    </div>
  );
}
