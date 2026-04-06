import * as React from "react";

import Picker from "@emoji-mart/react";
import data from "@emoji-mart/data";

import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/shared/ui/dropdown-menu";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useUpdateAssistant } from "@/shared/query/assistant";
import type { Assistant } from "@/shared/store/assistant";

interface AssistantEmojiPickerProps {
  assistant: Assistant;
  className?: string;
  emojiClassName?: string;
}

export function AssistantEmojiPicker({
  assistant,
  className,
  emojiClassName,
}: AssistantEmojiPickerProps) {
  const { t, language } = useI18n();
  const updateAssistant = useUpdateAssistant();
  const [open, setOpen] = React.useState(false);
  const emoji = assistant.identity?.emoji?.trim() || "🙂";
  const pickerLocale = language === "zh-CN" ? "zh" : "en";

  const handleSelect = async (selected: { native: string }) => {
    try {
      await updateAssistant.mutateAsync({
        id: assistant.id,
        identity: { ...(assistant.identity ?? {}), emoji: selected.native },
      });
      setOpen(false);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("settings.gateway.changeEmoji.error"),
        description: message,
        intent: "warning",
      });
    }
  };

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex items-center justify-center rounded-md px-1.5 py-1 text-foreground transition hover:bg-muted/60",
            className
          )}
          aria-label={t("settings.gateway.changeEmoji.open")}
          onClick={(event) => {
            event.stopPropagation();
          }}
        >
          <span className={cn("text-lg leading-none", emojiClassName)}>{emoji}</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        className="w-[360px] border-0 bg-transparent p-0 shadow-none"
        sideOffset={8}
        align="start"
      >
        <Picker
          data={data}
          onEmojiSelect={handleSelect}
          previewPosition="none"
          locale={pickerLocale}
          emojiButtonSize={28}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
