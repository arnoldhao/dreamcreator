import { ChevronDown, ChevronLeft, ChevronRight, ListFilter } from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Select } from "@/shared/ui/select";

import type { TranslateFn } from "../types";

export type ContextFilterOption = {
  value: string;
  label: string;
};

export type ContextFilterField = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: ContextFilterOption[];
};

export type ContextFooterStat = {
  label: string;
  value: string | number;
};

export const CONTEXT_ROWS_PER_PAGE_OPTIONS = [10, 20, 30, 50];

export function ContextFilterMenu(props: {
  t: TranslateFn;
  triggerLabel: string;
  filterCount: number;
  fields: ContextFilterField[];
  onClearAll: () => void;
  disabled?: boolean;
}) {
  const hasFilters = props.filterCount > 0;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[168px] max-w-[240px] justify-between gap-2 px-2.5"
          title={props.triggerLabel}
          disabled={props.disabled}
        >
          <span className="flex min-w-0 items-center gap-2">
            <ListFilter className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className="min-w-0 truncate text-xs text-foreground">{props.triggerLabel}</span>
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
      <DropdownMenuContent align="end" className="w-[320px] p-0">
        <div className="grid gap-2 p-2">
          {props.fields.map((field) => (
            <label key={field.label} className="grid gap-1">
              <span className="text-[11px] font-medium text-muted-foreground">{field.label}</span>
              <Select value={field.value} onChange={(event) => field.onChange(event.target.value)} className="w-full">
                {field.options.map((option) => (
                  <option key={`${field.label}-${option.value || "default"}`} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </label>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("settings.debug.contextPanel.actions.clearFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ContextTableFooter(props: {
  t: TranslateFn;
  stats: ContextFooterStat[];
  rowsPerPage: number;
  onRowsPerPageChange: (value: number) => void;
  pageIndex: number;
  pageCount: number;
  onPrevPage: () => void;
  onNextPage: () => void;
  className?: string;
}) {
  const pageText = props.t("library.table.pageOf")
    .replace("{page}", String(props.pageIndex + 1))
    .replace("{total}", String(props.pageCount));
  const rowsPerPageLabel = props.t("library.table.rowsPerPage");

  return (
    <div className={`flex shrink-0 flex-wrap items-center justify-between gap-3 text-xs ${props.className ?? ""}`}>
      <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
        {props.stats.map((stat) => (
          <span key={stat.label} className="inline-flex items-center gap-1.5 text-xs">
            <span className="text-muted-foreground">{stat.label}</span>
            <span className="text-foreground">{stat.value}</span>
          </span>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <Select
          value={String(props.rowsPerPage)}
          onChange={(event) => props.onRowsPerPageChange(Number(event.target.value))}
        >
          {CONTEXT_ROWS_PER_PAGE_OPTIONS.map((option) => (
            <option key={option} value={option}>
              {rowsPerPageLabel.replace("{count}", String(option))}
            </option>
          ))}
        </Select>
        <div className="text-muted-foreground">{pageText}</div>
        <Button
          variant="outline"
          size="compactIcon"
          onClick={props.onPrevPage}
          disabled={props.pageIndex <= 0}
          aria-label={props.t("library.table.prevPage")}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="compactIcon"
          onClick={props.onNextPage}
          disabled={props.pageIndex >= props.pageCount - 1}
          aria-label={props.t("library.table.nextPage")}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
