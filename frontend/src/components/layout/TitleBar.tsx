import type { CSSProperties, ReactNode } from "react";
import { System } from "@wailsio/runtime";

import { Separator } from "@/shared/ui/separator";
import { cn } from "@/lib/utils";

import { WindowControls } from "@/components/layout/WindowControls";

export interface TitleBarProps {
  title?: string;
  titleContent?: ReactNode;
  navActions?: ReactNode;
  titleActions?: ReactNode;
  extraActions?: ReactNode;
  reserveMacTrafficLightsGap?: boolean;
  showTitleSeparator?: boolean;
  boldTitle?: boolean;
  showBottomBorder?: boolean;
}

export function TitleBar({
  title,
  titleContent,
  navActions,
  titleActions,
  extraActions,
  reserveMacTrafficLightsGap,
  showTitleSeparator = true,
  boldTitle = false,
  showBottomBorder = true,
}: TitleBarProps) {
  const isWindows = System.IsWindows();
  const isMac = System.IsMac();

  const hasLeadingGroup = Boolean(navActions) || Boolean(titleActions);
  const hasTitleContent = Boolean(titleContent) || Boolean(title);
  const shouldReserveMacTrafficLightsGap = isMac && reserveMacTrafficLightsGap;

  return (
    <div
      className={cn(
        "relative flex h-[var(--app-titlebar-height)] shrink-0 items-center gap-3",
        "bg-background pl-3",
        isWindows ? "pr-0" : "pr-3"
      )}
      style={
        {
          "--wails-draggable": "drag",
          paddingLeft: shouldReserveMacTrafficLightsGap ? "var(--app-macos-traffic-lights-gap)" : undefined,
        } as CSSProperties
      }
    >
      {hasLeadingGroup ? (
        <div
          className="flex items-center gap-2"
          style={{ "--wails-draggable": "no-drag" } as CSSProperties}
        >
          {navActions}
          {titleActions}
        </div>
      ) : null}

      {showTitleSeparator && hasLeadingGroup && hasTitleContent ? (
        <Separator orientation="vertical" className="h-4" />
      ) : null}

      <div className={cn("min-w-0 flex-1", !hasLeadingGroup && "pl-3")}>
        {titleContent ?? (title ? (
          <div
            className={cn(
              "truncate font-display text-lg",
              boldTitle ? "font-bold text-foreground" : "font-semibold"
            )}
          >
            {title}
          </div>
        ) : null)}
      </div>

      {extraActions ? (
        <div
          className="flex items-center gap-2"
          style={{ "--wails-draggable": "no-drag" } as CSSProperties}
        >
          {extraActions}
        </div>
      ) : null}

      {isWindows ? <WindowControls platform="windows" /> : null}

      {showBottomBorder ? (
        <div
          className={cn(
            "pointer-events-none absolute bottom-0 left-2 h-px bg-border",
            isWindows ? "right-0" : "right-2"
          )}
        />
      ) : null}
    </div>
  );
}
