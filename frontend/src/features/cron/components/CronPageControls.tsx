import * as React from "react";
import { createPortal } from "react-dom";
import { ChevronDown, Search } from "lucide-react";

import { cn } from "@/lib/utils";
import type { CronJob } from "@/shared/store/cron";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";

type TranslateFn = (key: string) => string;
type JobEnabledFilter = "" | "enabled" | "disabled";

const RUN_STATUS_OPTIONS = ["", "running", "completed", "failed", "ok", "error", "skipped"];

export type CronContextMenuItem = {
  label: string;
  Icon: React.ComponentType<{ className?: string }>;
  onSelect: () => void;
  destructive?: boolean;
  disabled?: boolean;
};

export function CronContextMenu(props: {
  anchor: { x: number; y: number } | null;
  onClose: () => void;
  items: CronContextMenuItem[];
}) {
  const menuRef = React.useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = React.useState<{ x: number; y: number } | null>(props.anchor);

  React.useEffect(() => {
    setPosition(props.anchor);
  }, [props.anchor]);

  React.useLayoutEffect(() => {
    if (!props.anchor || !menuRef.current) {
      return;
    }
    const rect = menuRef.current.getBoundingClientRect();
    const nextX = Math.max(8, Math.min(props.anchor.x, window.innerWidth - rect.width - 8));
    const nextY = Math.max(8, Math.min(props.anchor.y, window.innerHeight - rect.height - 8));
    if (nextX !== position?.x || nextY !== position?.y) {
      setPosition({ x: nextX, y: nextY });
    }
  }, [position?.x, position?.y, props.anchor]);

  React.useEffect(() => {
    if (!props.anchor) {
      return;
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (menuRef.current?.contains(event.target as Node)) {
        return;
      }
      props.onClose();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        props.onClose();
      }
    };
    const handleViewportChange = () => props.onClose();
    document.addEventListener("pointerdown", handlePointerDown);
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("resize", handleViewportChange);
    window.addEventListener("blur", handleViewportChange);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("resize", handleViewportChange);
      window.removeEventListener("blur", handleViewportChange);
    };
  }, [props.anchor, props.onClose]);

  if (!props.anchor || !position || props.items.length === 0) {
    return null;
  }

  const itemClassName =
    "app-menu-item app-motion-color flex w-full items-center text-left text-sm outline-none hover:bg-accent hover:text-accent-foreground";

  return createPortal(
    <div
      ref={menuRef}
      role="menu"
      className={cn(
        "app-menu-content app-motion-surface fixed z-[120] w-max min-w-fit text-sm",
        "animate-in fade-in-0 zoom-in-95"
      )}
      style={{ left: position.x, top: position.y }}
      onContextMenu={(event) => event.preventDefault()}
    >
      {props.items.map((item, index) => (
        <React.Fragment key={`${item.label}-${index}`}>
          {index > 0 ? <div className="app-menu-separator" /> : null}
          <button
            type="button"
            role="menuitem"
            className={cn(
              itemClassName,
              item.destructive && "text-destructive hover:text-destructive",
              item.disabled && "pointer-events-none opacity-50"
            )}
            onClick={() => {
              props.onClose();
              item.onSelect();
            }}
            disabled={item.disabled}
          >
            <item.Icon className="h-4 w-4" />
            <span>{item.label}</span>
          </button>
        </React.Fragment>
      ))}
    </div>,
    document.body
  );
}

export const CronTableSelectionCheckbox = React.forwardRef<
  HTMLInputElement,
  Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> & { indeterminate?: boolean }
>(({ className, indeterminate = false, ...props }, forwardedRef) => {
  const innerRef = React.useRef<HTMLInputElement | null>(null);
  const setRefs = React.useCallback(
    (node: HTMLInputElement | null) => {
      innerRef.current = node;
      if (typeof forwardedRef === "function") {
        forwardedRef(node);
        return;
      }
      if (forwardedRef) {
        forwardedRef.current = node;
      }
    },
    [forwardedRef]
  );

  React.useEffect(() => {
    if (!innerRef.current) {
      return;
    }
    innerRef.current.indeterminate = indeterminate;
  }, [indeterminate]);

  return (
    <input
      {...props}
      ref={setRefs}
      type="checkbox"
      role="checkbox"
      className={cn(
        "h-4 w-4 rounded border border-border bg-background align-middle text-primary shadow-sm outline-none",
        "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      onClick={(event) => event.stopPropagation()}
    />
  );
});

CronTableSelectionCheckbox.displayName = "CronTableSelectionCheckbox";

export function CronJobFilterCombobox(props: {
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  enabledFilter: JobEnabledFilter;
  onEnabledFilterChange: (value: JobEnabledFilter) => void;
  lastRunStatusFilter: string;
  onLastRunStatusFilterChange: (value: string) => void;
  onClearAll: () => void;
  filterCount: number;
  t: TranslateFn;
}) {
  const hasFilters = props.filterCount > 0;
  const hasSearchQuery = props.searchQuery.trim().length > 0;
  const triggerLabel = hasSearchQuery ? props.searchQuery : props.t("cron.filter.searchAndFilterJobs");
  const enabledOptions: Array<{ value: JobEnabledFilter; label: string }> = [
    { value: "", label: props.t("cron.filter.allEnableStates") },
    { value: "enabled", label: props.t("cron.filter.enabledOnly") },
    { value: "disabled", label: props.t("cron.filter.disabledOnly") },
  ];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
          title={triggerLabel}
        >
          <span className="flex min-w-0 items-center gap-2">
            <Search className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className={cn("min-w-0 truncate text-xs", hasSearchQuery ? "text-foreground" : "text-muted-foreground")}>
              {triggerLabel}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            {hasFilters ? (
              <Badge variant="subtle" className="h-5 px-1.5 text-[10px] font-medium">
                {props.filterCount}
              </Badge>
            ) : null}
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("cron.filter.searchJobsPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>{props.t("cron.filter.enableState")}</DropdownMenuLabel>
          {enabledOptions.map((option) => (
            <DropdownMenuCheckboxItem
              key={option.value || "all"}
              checked={props.enabledFilter === option.value}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onEnabledFilterChange(option.value);
                }
              }}
            >
              {option.label}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{props.t("cron.filter.lastRunStatus")}</DropdownMenuLabel>
          {RUN_STATUS_OPTIONS.map((status) => (
            <DropdownMenuCheckboxItem
              key={status || "all"}
              checked={props.lastRunStatusFilter === status}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onLastRunStatusFilterChange(status);
                }
              }}
            >
              {status ? props.t(`cron.status.${status}`) : props.t("cron.filter.allStatus")}
            </DropdownMenuCheckboxItem>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("cron.filter.clearJobFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function CronRunFilterCombobox(props: {
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  jobFilter: string;
  onJobFilterChange: (value: string) => void;
  runStatusFilter: string;
  onRunStatusFilterChange: (value: string) => void;
  jobs: CronJob[];
  onClearAll: () => void;
  filterCount: number;
  t: TranslateFn;
}) {
  const hasFilters = props.filterCount > 0;
  const hasSearchQuery = props.searchQuery.trim().length > 0;
  const triggerLabel = hasSearchQuery ? props.searchQuery : props.t("cron.filter.searchAndFilterRuns");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
          title={triggerLabel}
        >
          <span className="flex min-w-0 items-center gap-2">
            <Search className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className={cn("min-w-0 truncate text-xs", hasSearchQuery ? "text-foreground" : "text-muted-foreground")}>
              {triggerLabel}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            {hasFilters ? (
              <Badge variant="subtle" className="h-5 px-1.5 text-[10px] font-medium">
                {props.filterCount}
              </Badge>
            ) : null}
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("cron.filter.searchRunsPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>{props.t("cron.filter.job")}</DropdownMenuLabel>
          <DropdownMenuCheckboxItem
            checked={props.jobFilter === ""}
            onSelect={(event) => event.preventDefault()}
            onCheckedChange={(checked) => {
              if (checked) {
                props.onJobFilterChange("");
              }
            }}
          >
            {props.t("cron.filter.allJobs")}
          </DropdownMenuCheckboxItem>
          {props.jobs.map((job) => (
            <DropdownMenuCheckboxItem
              key={job.id}
              checked={props.jobFilter === job.id}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onJobFilterChange(job.id);
                }
              }}
            >
              {job.name.trim() || job.id}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{props.t("cron.filter.runStatus")}</DropdownMenuLabel>
          {RUN_STATUS_OPTIONS.map((status) => (
            <DropdownMenuCheckboxItem
              key={status || "all"}
              checked={props.runStatusFilter === status}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onRunStatusFilterChange(status);
                }
              }}
            >
              {status ? props.t(`cron.status.${status}`) : props.t("cron.filter.allStatus")}
            </DropdownMenuCheckboxItem>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("cron.filter.clearRunFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
