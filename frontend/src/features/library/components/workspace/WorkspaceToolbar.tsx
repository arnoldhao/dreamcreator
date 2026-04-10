import type { ReactNode } from "react"
import { Filter, Replace, Search, Sparkles } from "lucide-react"

import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"
import { Select } from "@/shared/ui/select"
import { Switch } from "@/shared/ui/switch"
import { cn } from "@/lib/utils"

import type {
  WorkspaceFilter,
  WorkspaceQaFilter,
} from "./types"
import {
  WORKSPACE_CONTROL_FIELD_CLASS,
  WORKSPACE_CONTROL_INPUT_CLASS,
  WORKSPACE_CONTROL_LABEL_CLASS,
  WORKSPACE_CONTROL_SELECT_CLASS,
  WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS,
} from "./controlStyles"

type WorkspaceToolbarProps = {
  mode: "video" | "subtitle"
  searchValue: string
  onSearchChange: (value: string) => void
  replaceValue?: string
  onReplaceValueChange?: (value: string) => void
  onApplyReplace?: () => void
  replaceDisabled?: boolean
  filterValue: WorkspaceFilter
  onFilterChange: (value: WorkspaceFilter) => void
  qaFilter: WorkspaceQaFilter
  onQaFilterChange: (value: WorkspaceQaFilter) => void
  autoFollow?: boolean
  onAutoFollowChange?: (value: boolean) => void
}

function ToolbarField({
  icon: Icon,
  children,
  className,
}: {
  icon: typeof Search
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn(WORKSPACE_CONTROL_FIELD_CLASS, className)}>
      <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground/80" />
      {children}
    </div>
  )
}

export function WorkspaceToolbar({
  mode,
  searchValue,
  onSearchChange,
  replaceValue = "",
  onReplaceValueChange,
  onApplyReplace,
  replaceDisabled = false,
  filterValue,
  onFilterChange,
  qaFilter,
  onQaFilterChange,
  autoFollow = true,
  onAutoFollowChange,
}: WorkspaceToolbarProps) {
  const { t } = useI18n()
  const filterOptions: Array<{ value: WorkspaceFilter; label: string }> = [
    { value: "all", label: t("library.workspace.toolbar.filterAll") },
    { value: "needs-review", label: t("library.workspace.toolbar.filterNeedsReview") },
    { value: "edited", label: t("library.workspace.toolbar.filterEdited") },
    { value: "current-window", label: t("library.workspace.toolbar.filterCurrentWindow") },
  ]
  const qaFilterOptions: Array<{ value: WorkspaceQaFilter; label: string }> = [
    { value: "all", label: t("library.workspace.toolbar.qaAll") },
    { value: "issues", label: t("library.workspace.toolbar.qaIssues") },
    { value: "warnings", label: t("library.workspace.toolbar.qaWarnings") },
    { value: "errors", label: t("library.workspace.toolbar.qaErrors") },
    { value: "clean", label: t("library.workspace.toolbar.qaClean") },
  ]
  return (
    <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-border/70 bg-muted/[0.14] px-4 py-2">
      <ToolbarField icon={Search} className={WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS}>
        <Input
          value={searchValue}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("library.workspace.toolbar.searchPlaceholder")}
          className={cn(WORKSPACE_CONTROL_INPUT_CLASS, "flex-1")}
        />
      </ToolbarField>

      {mode === "subtitle" ? (
        <ToolbarField icon={Replace} className={WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS}>
          <Input
            value={replaceValue}
            onChange={(event) => onReplaceValueChange?.(event.target.value)}
            placeholder={t("library.workspace.toolbar.replacePlaceholder")}
            className={cn(WORKSPACE_CONTROL_INPUT_CLASS, "flex-1")}
            disabled={replaceDisabled}
          />
          <Button
            variant="outline"
            size="compact"
            className="h-6 border-border/60 bg-background/[0.72] px-2 text-xs text-foreground/84 hover:bg-accent/70"
            onClick={onApplyReplace}
            disabled={replaceDisabled || searchValue.trim().length === 0 || replaceValue.trim().length === 0}
          >
            {t("library.workspace.toolbar.apply")}
          </Button>
        </ToolbarField>
      ) : null}

      <ToolbarField icon={Filter} className={WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS}>
        <Select
          value={filterValue}
          onChange={(event) => onFilterChange(event.target.value as WorkspaceFilter)}
          className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
        >
          {filterOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </Select>
      </ToolbarField>

      {mode === "video" ? (
        <div
          className={cn(
            WORKSPACE_CONTROL_FIELD_CLASS,
            "w-fit min-w-0 max-w-[14.5rem] shrink justify-between gap-2",
          )}
        >
          <span
            className={cn(WORKSPACE_CONTROL_LABEL_CLASS, "min-w-0 flex-1 truncate")}
            title={t("library.workspace.toolbar.autoFollow")}
          >
            {t("library.workspace.toolbar.autoFollow")}
          </span>
          <Switch checked={autoFollow} onCheckedChange={onAutoFollowChange} className="shrink-0" />
        </div>
      ) : null}

      {mode === "subtitle" ? (
        <ToolbarField icon={Sparkles} className={WORKSPACE_CONTROL_STANDARD_WIDTH_CLASS}>
          <Select
            value={qaFilter}
            onChange={(event) => onQaFilterChange(event.target.value as WorkspaceQaFilter)}
            className={cn(WORKSPACE_CONTROL_SELECT_CLASS, "w-full")}
          >
            {qaFilterOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        </ToolbarField>
      ) : null}
    </div>
  )
}
