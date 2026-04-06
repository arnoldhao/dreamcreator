import { FolderOpen, Loader2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";

import { modelCardClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

interface AssistantWorkspacePanelProps {
  t: Translate;
  rootPath: string;
  isLoading: boolean;
  isOpening: boolean;
  onOpenWorkspace: () => void;
}

export function AssistantWorkspacePanel({
  t,
  rootPath,
  isLoading,
  isOpening,
  onOpenWorkspace,
}: AssistantWorkspacePanelProps) {
  const canOpen = Boolean(rootPath) && !isOpening;

  return (
    <div className={panelClassName}>
      <Card className={modelCardClassName}>
        <CardContent size="compact" className="space-y-2 p-2">
          <div className={rowClassName}>
            <div className="text-sm font-medium text-muted-foreground">
              {t("settings.gateway.workspace.path")}
            </div>
            <div className="flex min-w-0 items-center justify-end gap-2">
              {isLoading ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  <span>{t("settings.gateway.workspace.loading")}</span>
                </div>
              ) : (
                <span className="max-w-[260px] truncate text-right text-sm text-muted-foreground">
                  {rootPath ||
                    t("settings.gateway.workspace.empty")}
                </span>
              )}
              <Button
                type="button"
                variant="outline"
                size="compactIcon"
                disabled={!canOpen}
                onClick={onOpenWorkspace}
                aria-label={t("settings.gateway.workspace.open")}
              >
                {isOpening ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <FolderOpen className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
