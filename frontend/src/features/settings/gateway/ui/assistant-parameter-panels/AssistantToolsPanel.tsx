import * as React from "react";
import { Loader2 } from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";

import { modelCardClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

interface ToolSpecLite {
  id: string;
  name?: string;
  description?: string;
}

interface ToolEntry {
  spec: ToolSpecLite;
  enabled: boolean;
  locked: boolean;
}

interface ToolGroup {
  categoryId: string;
  categoryLabel: string;
  items: ToolEntry[];
}

interface AssistantToolsPanelProps {
  t: Translate;
  isLoading: boolean;
  groups: ToolGroup[];
  onToggle: (toolId: string, enabled: boolean) => void;
}

export function AssistantToolsPanel({ t, isLoading, groups, onToggle }: AssistantToolsPanelProps) {
  if (isLoading) {
    return (
      <div className={panelClassName}>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t("settings.gateway.tools.loading")}
        </div>
      </div>
    );
  }

  if (groups.length === 0) {
    return (
      <div className={panelClassName}>
        <div className="text-sm text-muted-foreground">
          {t("settings.gateway.tools.empty")}
        </div>
      </div>
    );
  }

  return (
    <div className={panelClassName}>
      {groups.map((group) => (
        <div key={group.categoryId} className="space-y-2">
          <div className="px-2 text-sm font-medium text-muted-foreground">{group.categoryLabel}</div>
          <Card className={modelCardClassName}>
            <CardContent size="compact" className="space-y-2 p-2">
              {group.items.map((entry, index) => {
                const toolId = entry.spec.id?.trim().toLowerCase() ?? "";
                const toolFallbackLabel =
                  entry.spec.name?.trim() ||
                  entry.spec.id?.trim() ||
                  t("settings.gateway.tools.untitled");
                const toolLabel =
                  toolId
                    ? t(`settings.tools.builtin.${toolId}.name`)
                    : toolFallbackLabel;
                const toolDescription =
                  toolId
                    ? t(`settings.tools.builtin.${toolId}.description`)
                    : entry.spec.description || "";

                return (
                  <React.Fragment key={entry.spec.id || `${group.categoryId}-${index}`}>
                    <div className={rowClassName}>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <div className="text-sm font-medium">{toolLabel}</div>
                          {entry.locked ? (
                            <Badge variant="subtle">
                              {t("settings.gateway.tools.disabled")}
                            </Badge>
                          ) : null}
                        </div>
                        {toolDescription ? (
                          <div className="text-xs text-muted-foreground">{toolDescription}</div>
                        ) : null}
                      </div>
                      <Switch
                        checked={entry.enabled}
                        disabled={entry.locked}
                        onCheckedChange={(value) => onToggle(entry.spec.id, value)}
                      />
                    </div>
                    {index < group.items.length - 1 ? <Separator /> : null}
                  </React.Fragment>
                );
              })}
            </CardContent>
          </Card>
        </div>
      ))}
    </div>
  );
}
