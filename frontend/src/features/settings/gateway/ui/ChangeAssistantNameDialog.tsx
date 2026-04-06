import * as React from "react";

import { Button } from "@/shared/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useUpdateAssistant } from "@/shared/query/assistant";
import type { Assistant } from "@/shared/store/assistant";

interface ChangeAssistantNameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  assistant: Assistant | null;
}

export function ChangeAssistantNameDialog({
  open,
  onOpenChange,
  assistant,
}: ChangeAssistantNameDialogProps) {
  const { t } = useI18n();
  const updateAssistant = useUpdateAssistant();
  const [name, setName] = React.useState("");
  const [creature, setCreature] = React.useState("");

  React.useEffect(() => {
    if (!open || !assistant) {
      return;
    }
    setName(assistant.identity?.name ?? "");
    setCreature(assistant.identity?.creature ?? "");
  }, [open, assistant?.id, assistant?.updatedAt, assistant]);

  const handleSave = async () => {
    if (!assistant) {
      return;
    }
    const trimmedName = name.trim();
    if (!trimmedName) {
      return;
    }
    try {
      await updateAssistant.mutateAsync({
        id: assistant.id,
        identity: {
          ...(assistant.identity ?? {}),
          name: trimmedName,
          creature: creature.trim(),
        },
      });
      onOpenChange(false);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.manager.updateError"),
        description: message,
        intent: "warning",
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("settings.gateway.changeName.title")}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <div className="text-xs font-medium text-muted-foreground">
              {t("settings.gateway.manager.name")}
            </div>
            <Input
              size="compact"
              className="h-7 w-48"
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <div className="text-xs font-medium text-muted-foreground">
              {t("settings.gateway.identity.creature")}
            </div>
            <Input
              size="compact"
              className="h-7 w-48"
              value={creature}
              onChange={(event) => setCreature(event.target.value)}
            />
          </div>
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            size="compact"
            className="h-7"
            onClick={() => onOpenChange(false)}
          >
            {t("settings.gateway.changeName.cancel")}
          </Button>
          <Button
            size="compact"
            className="h-7"
            onClick={handleSave}
            disabled={!name.trim() || updateAssistant.isPending}
          >
            {t("settings.gateway.changeName.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
