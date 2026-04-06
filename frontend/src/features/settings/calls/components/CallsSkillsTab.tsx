import * as React from "react";
import { BookOpen, FolderTree, ScrollText, Shield } from "lucide-react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { useI18n } from "@/shared/i18n";

import { SkillsAuditTab } from "./SkillsAuditTab";
import { SkillsCatalogTab } from "./SkillsCatalogTab";
import { SkillsSecurityTab } from "./SkillsSecurityTab";
import { SkillsSourcesTab } from "./SkillsSourcesTab";

type SkillsPanelTab = "catalog" | "sources" | "security" | "audit";

export function CallsSkillsTab({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useI18n();
  const [tab, setTab] = React.useState<SkillsPanelTab>("catalog");

  const content = (
    <Tabs
      value={tab}
      onValueChange={(value) => setTab(value as SkillsPanelTab)}
      className="flex min-h-0 flex-1 flex-col space-y-5"
    >
      <div className="flex justify-center">
        <TabsList className="w-fit max-w-full justify-start overflow-x-auto overflow-y-hidden">
          <TabsTrigger value="catalog" className="min-w-0">
            <BookOpen className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.tab.catalog")}</span>
          </TabsTrigger>
          <TabsTrigger value="sources" className="min-w-0">
            <FolderTree className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.tab.sources")}</span>
          </TabsTrigger>
          <TabsTrigger value="security" className="min-w-0">
            <Shield className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.tab.security")}</span>
          </TabsTrigger>
          <TabsTrigger value="audit" className="min-w-0">
            <ScrollText className="h-4 w-4" />
            <span className="truncate">{t("settings.calls.skills.tab.audit")}</span>
          </TabsTrigger>
        </TabsList>
      </div>

      {tab === "catalog" ? (
        <TabsContent value="catalog" className="mt-0 flex min-h-0 flex-1 flex-col">
          <SkillsCatalogTab />
        </TabsContent>
      ) : null}
      {tab === "sources" ? (
        <TabsContent value="sources" className="mt-0 flex min-h-0 flex-1 flex-col">
          <SkillsSourcesTab />
        </TabsContent>
      ) : null}
      {tab === "security" ? (
        <TabsContent value="security" className="mt-0 flex min-h-0 flex-1 flex-col">
          <SkillsSecurityTab />
        </TabsContent>
      ) : null}
      {tab === "audit" ? (
        <TabsContent value="audit" className="mt-0 flex min-h-0 flex-1 flex-col">
          <SkillsAuditTab />
        </TabsContent>
      ) : null}
    </Tabs>
  );

  if (embedded) {
    return (
      <TabsContent
        value="skills"
        className="mt-0 data-[state=active]:flex data-[state=active]:min-h-0 data-[state=active]:flex-1 data-[state=active]:flex-col"
      >
        {content}
      </TabsContent>
    );
  }

  return <div className="flex min-h-0 flex-1 flex-col">{content}</div>;
}
