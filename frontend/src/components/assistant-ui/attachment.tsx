import type { ReactNode } from "react";
import {
  AttachmentPrimitive,
  ComposerPrimitive,
  MessagePrimitive,
  useAssistantApi,
} from "@assistant-ui/react";
import { FileText, Plus, X } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/shared/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";

function AttachmentFrame({
  children,
  isComposer,
}: {
  children: ReactNode;
  isComposer: boolean;
}) {
  return (
    <AttachmentPrimitive.Root className={cn("relative", isComposer ? "only:[&>#attachment-tile]:size-24" : "")}>
      <TooltipTrigger asChild>
        <div
          id="attachment-tile"
          className={cn(
            "relative size-14 cursor-pointer overflow-hidden rounded-[14px] border bg-muted transition-opacity hover:opacity-75",
            isComposer ? "border-foreground/20" : "border-border/60"
          )}
          role="button"
          aria-label="attachment"
        >
          {children}
        </div>
      </TooltipTrigger>
      {isComposer ? (
        <AttachmentPrimitive.Remove asChild>
          <Button
            type="button"
            variant="ghost"
            size="compactIcon"
            className="absolute right-1.5 top-1.5 h-4 w-4 rounded-full bg-white text-muted-foreground opacity-100 shadow-sm hover:bg-white hover:text-destructive [&_svg]:h-3 [&_svg]:w-3 [&_svg]:text-black"
            aria-label="remove attachment"
          >
            <X className="size-3" />
          </Button>
        </AttachmentPrimitive.Remove>
      ) : null}
    </AttachmentPrimitive.Root>
  );
}

function ImageAttachmentTile() {
  const api = useAssistantApi();
  const isComposer = api.attachment.source === "composer";

  return (
    <Tooltip>
      <AttachmentFrame isComposer={isComposer}>
        <AttachmentPrimitive.unstable_Thumb className="absolute inset-0 h-full w-full object-cover" />
      </AttachmentFrame>
      <TooltipContent side="top">
        <AttachmentPrimitive.Name />
      </TooltipContent>
    </Tooltip>
  );
}

function GenericAttachmentTile() {
  const api = useAssistantApi();
  const isComposer = api.attachment.source === "composer";

  return (
    <Tooltip>
      <AttachmentFrame isComposer={isComposer}>
        <div className="absolute inset-0 flex items-center justify-center text-muted-foreground">
          <FileText className="size-6" />
        </div>
      </AttachmentFrame>
      <TooltipContent side="top">
        <AttachmentPrimitive.Name />
      </TooltipContent>
    </Tooltip>
  );
}

export function UserMessageAttachments() {
  return (
    <MessagePrimitive.If hasAttachments>
      <div className="flex w-full flex-row justify-end gap-2">
        <MessagePrimitive.Attachments
          components={{
            Image: ImageAttachmentTile,
            Document: GenericAttachmentTile,
            File: GenericAttachmentTile,
            Attachment: GenericAttachmentTile,
          }}
        />
      </div>
    </MessagePrimitive.If>
  );
}

export function ComposerAttachments() {
  return (
    <div className="mb-2 flex w-full flex-row items-center gap-2 overflow-x-auto px-1.5 pb-1 pt-0.5 empty:hidden">
      <ComposerPrimitive.Attachments
        components={{
          Image: ImageAttachmentTile,
          Document: GenericAttachmentTile,
          File: GenericAttachmentTile,
          Attachment: GenericAttachmentTile,
        }}
      />
    </div>
  );
}

type ComposerAddAttachmentButtonProps = {
  onClick: () => void;
  disabled?: boolean;
  "aria-label"?: string;
};

export function ComposerAddAttachmentButton({
  onClick,
  disabled,
  "aria-label": ariaLabel = "Add Attachment",
}: ComposerAddAttachmentButtonProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="compactIcon"
      className="size-8.5 rounded-full p-1 text-xs font-semibold hover:bg-muted-foreground/15 dark:border-muted-foreground/15 dark:hover:bg-muted-foreground/30"
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={onClick}
    >
      <Plus className="size-5 stroke-[1.5px]" />
    </Button>
  );
}
