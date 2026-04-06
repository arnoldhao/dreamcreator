import * as React from "react";

import { SectionCard } from "@/shared/ui/SectionCard";
import { Badge } from "@/shared/ui/badge";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemSeparator,
  ItemTitle,
} from "@/shared/ui/item";
import { useI18n } from "@/shared/i18n";
import { useSkillsCatalog } from "@/shared/query/skills";
import type { ProviderSkillSpec } from "@/shared/contracts/skills";

export function SkillsSection() {
  const { t } = useI18n();
  const skillsQuery = useSkillsCatalog();
  const discoveredSkills = skillsQuery.data ?? [];

  const list: ProviderSkillSpec[] = React.useMemo(() => {
    if (discoveredSkills.length === 0) {
      return [];
    }
    return discoveredSkills;
  }, [discoveredSkills]);

  return (
    <div className="space-y-6">
      <SectionCard
        title={t("settings.skills.listTitle")}
        description={t("settings.skills.listDescription")}
      >
        <ItemGroup className="rounded-lg border">
          {skillsQuery.isLoading ? (
            <div className="p-4 text-sm text-muted-foreground">
              {t("settings.skills.loading")}
            </div>
          ) : skillsQuery.isError ? (
            <div className="p-4 text-sm text-destructive">
              {t("settings.skills.error")}
            </div>
          ) : list.length === 0 ? (
            <div className="p-4 text-sm text-muted-foreground">
              {t("settings.skills.listEmpty")}
            </div>
          ) : (
            list.map((skill, index) => {
              const skillID = skill.id?.trim();
              if (!skillID) {
                return null;
              }
              const allowed = skill.enabled !== false;
              return (
                <React.Fragment key={skillID}>
                  <Item className="rounded-none border-0">
                    <ItemContent>
                      <ItemTitle>{skill.name?.trim() || skillID}</ItemTitle>
                      <ItemDescription>{skill.description?.trim() || skillID}</ItemDescription>
                    </ItemContent>
                    <ItemActions>
                      <Badge variant={allowed ? "secondary" : "outline"}>
                        {allowed
                          ? t("settings.tools.status.allowed")
                          : t("settings.skills.blocked")}
                      </Badge>
                    </ItemActions>
                  </Item>
                  {index < list.length - 1 ? <ItemSeparator /> : null}
                </React.Fragment>
              );
            })
          )}
        </ItemGroup>
      </SectionCard>
    </div>
  );
}
