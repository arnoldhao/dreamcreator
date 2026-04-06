import * as Dialog from "@radix-ui/react-dialog";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from "lucide-react";

import { useI18n } from "@/shared/i18n";
import { Button } from "@/shared/ui/button";
import { cn } from "@/lib/utils";
import { messageBus, useMessages } from "./store";
import type {
  DialogMessage,
  MessageAction,
  MessageIntent,
  NotificationMessage,
  ToastMessage,
} from "./types";

type IntentStyle = {
  bg: string;
  border: string;
  text: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
};

const INTENT_STYLE: Record<MessageIntent, IntentStyle> = {
  info: {
    bg: "bg-slate-900/80 backdrop-blur-sm",
    border: "border-slate-800/70",
    text: "text-slate-50",
    icon: Info,
  },
  success: {
    bg: "bg-emerald-900/80 backdrop-blur-sm",
    border: "border-emerald-800/70",
    text: "text-emerald-50",
    icon: CheckCircle2,
  },
  warning: {
    bg: "bg-amber-900/80 backdrop-blur-sm",
    border: "border-amber-800/70",
    text: "text-amber-50",
    icon: AlertTriangle,
  },
  danger: {
    bg: "bg-red-900/80 backdrop-blur-sm",
    border: "border-red-800/70",
    text: "text-red-50",
    icon: XCircle,
  },
};

const DEFAULT_ITEM_HEIGHT = 72;
const STACK_GAP = 8;
const STACK_OFFSET = 12;
const STACK_SCALE_STEP = 0.03;
const STACK_OPACITY_STEP = 0.08;
const STACK_MIN_SCALE = 0.85;
const STACK_MIN_OPACITY = 0.4;

type StackLayout = {
  stacked: boolean;
  offset: number;
};

function useViewportHeight() {
  const [height, setHeight] = useState(() =>
    typeof window === "undefined" ? 0 : window.innerHeight
  );

  useEffect(() => {
    const handleResize = () => setHeight(window.innerHeight);
    handleResize();
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return height;
}

function useElementHeight<T extends HTMLElement>() {
  const [height, setHeight] = useState(0);
  const observerRef = useRef<ResizeObserver | null>(null);

  const ref = useCallback((node: T | null) => {
    if (observerRef.current) {
      observerRef.current.disconnect();
      observerRef.current = null;
    }
    if (!node) {
      return;
    }
    const update = () => setHeight(node.getBoundingClientRect().height);
    update();
    observerRef.current = new ResizeObserver(update);
    observerRef.current.observe(node);
  }, []);

  useEffect(() => () => observerRef.current?.disconnect(), []);

  return { ref, height };
}

function getStackLayout(count: number, itemHeight: number, maxHeight: number): StackLayout {
  if (!count || !maxHeight) {
    return { stacked: false, offset: STACK_OFFSET };
  }
  const resolvedHeight = itemHeight > 0 ? itemHeight : DEFAULT_ITEM_HEIGHT;
  const totalHeight = count * (resolvedHeight + STACK_GAP) - STACK_GAP;
  if (totalHeight <= maxHeight) {
    return { stacked: false, offset: STACK_OFFSET };
  }
  const offset =
    maxHeight > resolvedHeight
      ? Math.min(STACK_OFFSET, (maxHeight - resolvedHeight) / Math.max(1, count - 1))
      : 0;
  return { stacked: true, offset: Math.max(0, offset) };
}

function getStackItemStyle(index: number, total: number, layout: StackLayout) {
  if (!layout.stacked) {
    return undefined;
  }
  const depthIndex = index;
  const translateY = depthIndex * layout.offset;
  const scale = Math.max(STACK_MIN_SCALE, 1 - depthIndex * STACK_SCALE_STEP);
  const opacity = Math.max(STACK_MIN_OPACITY, 1 - depthIndex * STACK_OPACITY_STEP);

  return {
    transform: `translateY(${translateY}px) scale(${scale})`,
    transformOrigin: "top center",
    opacity,
    zIndex: 100 + (total - index),
  } as const;
}

function formatMessageText(template: string, params?: Record<string, string>) {
  if (!params) {
    return template;
  }
  let output = template;
  Object.entries(params).forEach(([key, value]) => {
    output = output.split(`{${key}}`).join(value ?? "");
  });
  return output;
}

function resolveMessageText(
  raw: string | undefined,
  key: string | undefined,
  params: Record<string, string> | undefined,
  t: (key: string) => string
) {
  if (key) {
    return formatMessageText(t(key), params);
  }
  return raw ?? "";
}

function resolveActionLabel(action: MessageAction | undefined, t: (key: string) => string) {
  if (!action) {
    return "";
  }
  if (action.labelKey) {
    return t(action.labelKey);
  }
  return action.label ?? "";
}

function ToastItem({ message }: { message: ToastMessage }) {
  const style = INTENT_STYLE[message.intent];
  const Icon = style.icon;
  const { t } = useI18n();
  const title = resolveMessageText(message.title, message.i18n?.titleKey, message.i18n?.params, t);
  const description = resolveMessageText(
    message.description,
    message.i18n?.descriptionKey,
    message.i18n?.params,
    t
  );
  const actionLabel = resolveActionLabel(message.action, t);

  useEffect(() => {
    let timer: number | null = null;
    let cancelled = false;

    const scheduleDismiss = () => {
      if (!message.autoCloseMs || message.autoCloseMs <= 0) {
        return;
      }
      timer = window.setTimeout(() => messageBus.dismiss(message.id), message.autoCloseMs);
    };

    if (message.awaitFor) {
      message.awaitFor
        .catch(() => null)
        .finally(() => {
          if (cancelled) {
            return;
          }
          if (!message.autoCloseMs || message.autoCloseMs <= 0) {
            messageBus.dismiss(message.id);
            return;
          }
          scheduleDismiss();
        });
    } else {
      scheduleDismiss();
    }

    return () => {
      cancelled = true;
      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [message.id, message.autoCloseMs, message.awaitFor]);

  return (
    <div
      className={cn(
        "flex w-full gap-3 rounded-lg border p-3 shadow-lg",
        style.bg,
        style.border,
        style.text
      )}
    >
      <Icon className="h-4 w-4 shrink-0 opacity-90" />
      <div className="min-w-0 flex-1 space-y-1">
        {title ? <div className="text-sm font-semibold leading-tight">{title}</div> : null}
        {description ? (
          <div className="text-xs leading-snug text-white/80">{description}</div>
        ) : null}
        {message.action && actionLabel ? (
          <div>
            <Button
              size="compact"
              variant="secondary"
              onClick={() => {
                message.action?.onClick?.();
                messageBus.dismiss(message.id);
              }}
            >
              {actionLabel}
            </Button>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function NotificationItem({ message }: { message: NotificationMessage }) {
  const style = INTENT_STYLE[message.intent];
  const Icon = style.icon;
  const { t } = useI18n();
  const [thumbnailFailed, setThumbnailFailed] = useState(false);
  const thumbnailUrl = resolveNotificationThumbnail(message);
  const showThumbnail = Boolean(thumbnailUrl) && !thumbnailFailed;
  const title = resolveMessageText(message.title, message.i18n?.titleKey, message.i18n?.params, t);
  const description = resolveMessageText(
    message.description,
    message.i18n?.descriptionKey,
    message.i18n?.params,
    t
  );

  const handleActionClick = (action?: () => void) => {
    action?.();
    messageBus.markNotificationRead(message.id, false);
    messageBus.dismiss(message.id);
  };

  return (
    <div
      className={cn(
        "flex w-full gap-3 rounded-lg border p-3 shadow-lg",
        style.bg,
        style.border,
        style.text
      )}
    >
      {showThumbnail ? (
        <div className="h-10 w-16 shrink-0 overflow-hidden rounded-md bg-black/10">
          <img
            src={thumbnailUrl}
            alt={message.title ?? "Thumbnail"}
            className="h-full w-full object-cover"
            onError={() => setThumbnailFailed(true)}
          />
        </div>
      ) : (
        <Icon className="h-4 w-4 shrink-0 opacity-90" />
      )}
      <div className="min-w-0 flex-1 space-y-1">
        {title ? <div className="text-sm font-semibold leading-tight">{title}</div> : null}
        {description ? (
          <div className="text-xs leading-snug text-white/80">{description}</div>
        ) : null}
        {message.actions && message.actions.length > 0 ? (
          <div className="flex flex-wrap gap-2 pt-1">
            {message.actions.map((action) => {
              const actionLabel = resolveActionLabel(action, t);
              if (!actionLabel) {
                return null;
              }
              return (
                <Button
                  key={`${message.id}-${actionLabel}`}
                  size="compact"
                  variant={action.intent === "danger" ? "destructive" : "secondary"}
                  onClick={() => handleActionClick(action.onClick)}
                >
                  {actionLabel}
                </Button>
              );
            })}
          </div>
        ) : null}
      </div>
      <Button
        size="compactIcon"
        variant="ghost"
        className="text-white/70 hover:text-white"
        onClick={() => {
          messageBus.markNotificationRead(message.id, false);
          messageBus.dismiss(message.id);
        }}
        aria-label="dismiss notification"
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
}

function resolveNotificationThumbnail(message: NotificationMessage) {
  const data = message.data;
  if (!data || typeof data !== "object") {
    return "";
  }
  const value = (data as { thumbnailUrl?: unknown }).thumbnailUrl;
  return typeof value === "string" ? value.trim() : "";
}

function DialogHost({ message }: { message: DialogMessage }) {
  const { t } = useI18n();
  const title = resolveMessageText(message.title, message.i18n?.titleKey, message.i18n?.params, t);
  const description = resolveMessageText(
    message.description,
    message.i18n?.descriptionKey,
    message.i18n?.params,
    t
  );
  const cancelLabel = message.cancelLabelKey ? t(message.cancelLabelKey) : message.cancelLabel ?? "Cancel";
  const confirmLabel = message.confirmLabelKey ? t(message.confirmLabelKey) : message.confirmLabel ?? "Confirm";

  const handleClose = () => {
    message.onCancel?.();
    messageBus.dismiss(message.id);
  };

  const handleConfirm = async () => {
    try {
      await message.onConfirm?.();
    } finally {
      messageBus.dismiss(message.id);
    }
  };

  return (
    <Dialog.Root open onOpenChange={(open) => (!open ? handleClose() : null)}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/80 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
        <Dialog.Content
          className={cn(
            "fixed left-1/2 top-1/2 z-50 grid w-full max-w-lg -translate-x-1/2 -translate-y-1/2 border bg-background shadow-lg duration-200 overflow-hidden",
            "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
            "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
            "data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%]",
            "data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]",
            "sm:rounded-lg"
          )}
        >
          <div className="flex flex-col space-y-1.5 px-6 pt-6 pb-4 text-center sm:text-left">
            {title ? (
              <Dialog.Title className="text-lg font-semibold leading-none tracking-tight">
                {title}
              </Dialog.Title>
            ) : null}
            {description ? (
              <Dialog.Description className="text-sm text-muted-foreground">
                {description}
              </Dialog.Description>
            ) : null}
          </div>

          <div className="flex flex-col-reverse gap-2 px-6 py-4 sm:flex-row sm:justify-end">
            <Button variant="outline" size="compact" onClick={handleClose}>
              {cancelLabel}
            </Button>
            <Button
              variant={message.destructive ? "destructive" : "default"}
              size="compact"
              onClick={handleConfirm}
            >
              {confirmLabel}
            </Button>
          </div>

          <Dialog.Close asChild>
            <Button
              variant="ghost"
              size="compactIcon"
              className="absolute right-4 top-4"
              aria-label="close"
            >
              <X className="h-4 w-4" />
            </Button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

export function MessageHost() {
  const viewportHeight = useViewportHeight();
  const stackMeasure = useElementHeight<HTMLDivElement>();
  const messages = useMessages();
  const stackedMessages = useMemo(
    () =>
      messages
        .filter((m): m is ToastMessage | NotificationMessage => m.kind !== "dialog")
        .sort((a, b) => b.ts - a.ts),
    [messages]
  );
  const dialog = useMemo(() => messages.find((m): m is DialogMessage => m.kind === "dialog"), [messages]);
  const maxStackHeight = viewportHeight * 0.5;
  const stackLayout = getStackLayout(stackedMessages.length, stackMeasure.height, maxStackHeight);

  return (
    <>
      <div
        className={cn(
          "pointer-events-none absolute left-1/2 top-[calc(var(--app-titlebar-height)+1.5rem)] z-[60] w-[360px] max-w-[92vw] -translate-x-1/2 max-h-[50vh]",
          stackLayout.stacked ? "h-[50vh] overflow-hidden" : "flex flex-col gap-2"
        )}
      >
        {stackedMessages.map((message, index) => (
          <div
            key={message.id}
            ref={index === 0 ? stackMeasure.ref : undefined}
            className={cn(
              "pointer-events-auto",
              stackLayout.stacked ? "absolute left-0 right-0 top-0" : null
            )}
            style={getStackItemStyle(index, stackedMessages.length, stackLayout)}
          >
            <div className="animate-in slide-in-from-top-2 fade-in duration-200">
              {message.kind === "toast" ? (
                <ToastItem message={message} />
              ) : (
                <NotificationItem message={message} />
              )}
            </div>
          </div>
        ))}
      </div>

      {dialog ? <DialogHost message={dialog} /> : null}
    </>
  );
}
