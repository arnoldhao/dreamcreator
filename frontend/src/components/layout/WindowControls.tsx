import * as React from "react";
import type { CSSProperties } from "react";

import { Events, Window } from "@wailsio/runtime";

import { Button } from "@/shared/ui/button";
import { cn } from "@/lib/utils";

export interface WindowControlsProps {
  platform: "macos" | "windows";
}

function WindowsMinimiseGlyph() {
  return <span className="h-px w-[10px] bg-current" />;
}

function WindowsMaximiseGlyph() {
  return <span className="h-[10px] w-[10px] border border-current" />;
}

function WindowsRestoreGlyph() {
  return (
    <span className="relative h-[10px] w-[10px]">
      <span className="absolute right-0 top-0 h-[7px] w-[7px] border border-current bg-transparent" />
      <span className="absolute bottom-0 left-0 h-[7px] w-[7px] border border-current bg-transparent" />
    </span>
  );
}

function WindowsCloseGlyph() {
  return (
    <span className="relative h-[10px] w-[10px]">
      <span className="absolute left-1/2 top-0 h-full w-px -translate-x-1/2 rotate-45 bg-current" />
      <span className="absolute left-1/2 top-0 h-full w-px -translate-x-1/2 -rotate-45 bg-current" />
    </span>
  );
}

export function WindowControls({ platform }: WindowControlsProps) {
  const [isMaximised, setIsMaximised] = React.useState(false);

  React.useEffect(() => {
    if (platform !== "windows") {
      setIsMaximised(false);
      return;
    }

    let cancelled = false;

    const syncMaximised = async () => {
      try {
        const next = await Window.IsMaximised();
        if (!cancelled) {
          setIsMaximised(Boolean(next));
        }
      } catch {
        if (!cancelled) {
          setIsMaximised(false);
        }
      }
    };

    void syncMaximised();

    const offMaximise = Events.On(Events.Types.Common.WindowMaximise, () => {
      setIsMaximised(true);
    });
    const offUnMaximise = Events.On(Events.Types.Common.WindowUnMaximise, () => {
      setIsMaximised(false);
    });
    const offRestore = Events.On(Events.Types.Common.WindowRestore, () => {
      void syncMaximised();
    });

    return () => {
      cancelled = true;
      offMaximise();
      offUnMaximise();
      offRestore();
    };
  }, [platform]);

  const handleClose = () => {
    void Window.Close();
  };

  const handleMinimize = () => {
    void Window.Minimise();
  };

  const handleToggleMaximize = () => {
    void Window.ToggleMaximise();
  };

  if (platform === "macos") {
    return (
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "wails-no-drag h-3 w-3 rounded-full p-0",
            "bg-[#ff5f57] hover:bg-[#e0443e]"
          )}
          style={{ "--wails-draggable": "no-drag" } as CSSProperties}
          onClick={handleClose}
          aria-label="Close window"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "wails-no-drag h-3 w-3 rounded-full p-0",
            "bg-[#febc2e] hover:bg-[#dea123]"
          )}
          style={{ "--wails-draggable": "no-drag" } as CSSProperties}
          onClick={handleMinimize}
          aria-label="Minimize window"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "wails-no-drag h-3 w-3 rounded-full p-0",
            "bg-[#28c840] hover:bg-[#1aab29]"
          )}
          style={{ "--wails-draggable": "no-drag" } as CSSProperties}
          onClick={handleToggleMaximize}
          aria-label="Maximize window"
        />
      </div>
    );
  }

  return (
    <div
      className="flex h-[var(--app-windows-caption-button-height,var(--app-titlebar-height))] shrink-0 self-start items-stretch"
      style={{ "--wails-draggable": "no-drag" } as CSSProperties}
    >
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn(
          "wails-no-drag h-full w-[var(--app-windows-caption-button-width)] rounded-none border-0 px-0 text-foreground/75 shadow-none",
          "hover:bg-black/10 hover:text-foreground focus-visible:ring-0 focus-visible:ring-offset-0 active:bg-black/15",
          "dark:hover:bg-white/10 dark:active:bg-white/15"
        )}
        style={{ "--wails-draggable": "no-drag" } as CSSProperties}
        onClick={handleMinimize}
        aria-label="Minimize window"
      >
        <WindowsMinimiseGlyph />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn(
          "wails-no-drag h-full w-[var(--app-windows-caption-button-width)] rounded-none border-0 px-0 text-foreground/75 shadow-none",
          "hover:bg-black/10 hover:text-foreground focus-visible:ring-0 focus-visible:ring-offset-0 active:bg-black/15",
          "dark:hover:bg-white/10 dark:active:bg-white/15"
        )}
        style={{ "--wails-draggable": "no-drag" } as CSSProperties}
        onClick={handleToggleMaximize}
        aria-label={isMaximised ? "Restore window" : "Maximize window"}
      >
        {isMaximised ? <WindowsRestoreGlyph /> : <WindowsMaximiseGlyph />}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn(
          "wails-no-drag h-full w-[var(--app-windows-caption-button-width)] rounded-none border-0 px-0 text-foreground/75 shadow-none",
          "hover:bg-[#e81123] hover:text-white focus-visible:ring-0 focus-visible:ring-offset-0 active:bg-[#c50f1f]"
        )}
        style={{ "--wails-draggable": "no-drag" } as CSSProperties}
        onClick={handleClose}
        aria-label="Close window"
      >
        <WindowsCloseGlyph />
      </Button>
    </div>
  );
}
