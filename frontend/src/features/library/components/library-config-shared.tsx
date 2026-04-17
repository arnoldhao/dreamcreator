import * as React from "react";
import { Search } from "lucide-react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/shared/ui/empty";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import type { LibraryTaskRuntimeSettingsDTO } from "@/shared/contracts/library";
import { Badge } from "@/shared/ui/badge";
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";

import {
  parseNonNegativeInt,
  parsePositiveInt,
} from "./library-config-utils";

export function ConfigSectionCard(props: {
  title: string;
  description?: string;
  icon?: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  const Icon = props.icon;

  return (
    <section
      className={cn(
        "space-y-3 p-4",
        DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
        props.className,
      )}
    >
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex items-center gap-2">
            {Icon ? <Icon className="h-4 w-4 text-muted-foreground" /> : null}
            <div className="text-sm font-semibold text-foreground">
              {props.title}
            </div>
          </div>
          {props.description ? (
            <div className="text-xs leading-5 text-muted-foreground">
              {props.description}
            </div>
          ) : null}
        </div>
      </div>
      <div className={props.contentClassName ?? "space-y-3"}>
        {props.children}
      </div>
    </section>
  );
}

export function ConfigMasterDetailLayout(props: {
  sidebar: React.ReactNode;
  detail: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid h-full min-h-0 w-full min-w-0 grid-cols-[minmax(240px,320px)_minmax(0,1fr)] gap-4 overflow-x-auto",
        props.className,
      )}
    >
      <div className="h-full min-h-0 min-w-[240px]">{props.sidebar}</div>
      <div className="h-full min-h-0 min-w-0">{props.detail}</div>
    </div>
  );
}

export function ConfigNavigationItem(props: {
  title: string;
  description?: string;
  badges?: React.ReactNode;
  selected: boolean;
  disabled?: boolean;
  compact?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={props.disabled}
      onClick={props.onClick}
      className={cn(
        "w-full border text-left transition-colors",
        props.compact
          ? "flex min-h-[60px] flex-col justify-center rounded-lg px-3 py-2.5"
          : "rounded-xl px-3 py-3",
        props.selected
          ? "border-border/70 bg-background/90 shadow-sm"
          : "border-transparent bg-background/45 hover:border-border/60 hover:bg-background/70",
        props.disabled ? "cursor-not-allowed opacity-70" : "",
      )}
    >
      <div
        className={cn(
          "truncate font-medium text-foreground",
          props.compact ? "text-[13px] leading-5" : "text-sm",
        )}
      >
        {props.title}
      </div>
      {props.compact ? (
        <div
          className={cn(
            "mt-0.5 min-h-4 text-[11px] leading-4 text-muted-foreground",
            props.description ? "line-clamp-1" : "opacity-0",
          )}
        >
          {props.description || "-"}
        </div>
      ) : props.description ? (
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          {props.description}
        </div>
      ) : null}
      {props.badges ? (
        <div
          className={cn(
            "flex flex-wrap gap-1.5",
            props.compact ? "mt-1.5" : "mt-2",
          )}
        >
          {props.badges}
        </div>
      ) : null}
    </button>
  );
}

export function ConfigNavigationSidebar(props: {
  showSearch?: boolean;
  searchValue: string;
  onSearchChange: (nextValue: string) => void;
  searchPlaceholder: string;
  count: number;
  emptyState: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <div
      className={cn(
        "flex h-full min-h-0 flex-col overflow-hidden p-4",
        DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
        props.className,
      )}
    >
      {props.showSearch ? (
        <div className="mb-3 flex items-center gap-2 rounded-lg border border-border/60 bg-background/75 px-2.5 py-2">
          <Search className="h-3.5 w-3.5 text-muted-foreground" />
          <Input
            value={props.searchValue}
            onChange={(event) => props.onSearchChange(event.target.value)}
            placeholder={props.searchPlaceholder}
            className="h-7 border-none bg-transparent px-0 text-xs shadow-none focus-visible:ring-0"
          />
          <Badge
            variant="outline"
            className="h-5 shrink-0 px-1.5 text-[10px]"
          >
            {props.count}
          </Badge>
        </div>
      ) : null}
      {props.count === 0 ? (
        <div className="min-h-0 flex-1">{props.emptyState}</div>
      ) : (
        <div
          className={cn(
            "min-h-0 flex-1 overflow-y-auto pr-1",
            props.bodyClassName ?? "space-y-3",
          )}
        >
          {props.children}
        </div>
      )}
    </div>
  );
}

export function ConfigNavigationGroup(props: {
  title: string;
  count: number;
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <div className={cn("space-y-2", props.className)}>
      <div className="flex items-center justify-between px-1">
        <div className="text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
          {props.title}
        </div>
        <div className="text-[10px] text-muted-foreground">{props.count}</div>
      </div>
      <div className={cn("space-y-1.5", props.contentClassName)}>
        {props.children}
      </div>
    </div>
  );
}

export function ConfigDetailPanel(props: {
  title: React.ReactNode;
  actions?: React.ReactNode;
  badges?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <div
      className={cn(
        "flex h-full min-h-0 flex-col gap-4 overflow-hidden p-4",
        DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
        props.className,
      )}
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            {typeof props.title === "string" ? (
              <div className="truncate text-sm font-semibold text-foreground">
                {props.title}
              </div>
            ) : (
              <div className="max-w-full">{props.title}</div>
            )}
          </div>
          {props.actions}
        </div>
        {props.badges ? (
          <div className="flex flex-wrap gap-1.5">{props.badges}</div>
        ) : null}
      </div>
      <div
        className={cn(
          "min-h-0 flex-1 overflow-y-auto pr-1",
          props.contentClassName ?? "space-y-4",
        )}
      >
        {props.children}
      </div>
      {props.footer}
    </div>
  );
}

export function TaskRuntimeFields(props: {
  value: LibraryTaskRuntimeSettingsDTO;
  disabled?: boolean;
  onChange: (nextValue: LibraryTaskRuntimeSettingsDTO) => void;
}) {
  const { t } = useI18n();
  const updateField = <K extends keyof LibraryTaskRuntimeSettingsDTO>(
    field: K,
    value: LibraryTaskRuntimeSettingsDTO[K],
  ) => {
    props.onChange({
      ...props.value,
      [field]: value,
    });
  };

  const renderSelectCard = (
    label: string,
    description: string,
    value: string,
    options: Array<{ value: string; label: string }>,
    onChange: (nextValue: string) => void,
  ) => (
    <div
      className={cn(
        "grid grid-cols-[minmax(0,1fr)_144px] gap-2 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_168px]",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <div className="min-w-0 pr-2">
        <div className="truncate text-sm font-medium text-foreground">
          {label}
        </div>
      </div>
      <Select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={props.disabled}
        className="h-9 w-full min-w-0 justify-self-end border-border/70 bg-background/80 text-xs"
      >
        {options.map((option) => (
          <option key={`${option.value}-${option.label}`} value={option.value}>
            {option.label}
          </option>
        ))}
      </Select>
      <div className="col-span-2 text-xs leading-5 text-muted-foreground">
        {description}
      </div>
    </div>
  );

  const renderNumberCard = (
    label: string,
    description: string,
    value: number,
    onChange: (nextValue: number) => void,
  ) => (
    <div
      className={cn(
        "grid grid-cols-[minmax(0,1fr)_144px] gap-2 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_168px]",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <div className="min-w-0 pr-2">
        <div className="truncate text-sm font-medium text-foreground">
          {label}
        </div>
      </div>
      <Input
        type="number"
        min={0}
        value={String(value)}
        disabled={props.disabled}
        onChange={(event) =>
          onChange(parseNonNegativeInt(event.target.value, value))
        }
        className="h-9 w-full min-w-0 justify-self-end border-border/70 bg-background/80"
      />
      <div className="col-span-2 text-xs leading-5 text-muted-foreground">
        {description}
      </div>
    </div>
  );

  return (
    <div className="grid content-start gap-3">
      {renderSelectCard(
        t("library.config.taskRuntime.structuredOutput"),
        t("library.config.taskRuntime.structuredOutputDescription"),
        props.value.structuredOutputMode,
        [
          {
            value: "auto",
            label: t("library.config.taskRuntime.structuredOutputAuto"),
          },
          {
            value: "json_schema",
            label: t("library.config.taskRuntime.structuredOutputSchema"),
          },
          {
            value: "prompt_only",
            label: t("library.config.taskRuntime.structuredOutputPrompt"),
          },
        ],
        (nextValue) => updateField("structuredOutputMode", nextValue),
      )}
      {renderSelectCard(
        t("library.config.taskRuntime.thinkingMode"),
        t("library.config.taskRuntime.thinkingModeDescription"),
        props.value.thinkingMode,
        [
          {
            value: "off",
            label: t("library.config.taskRuntime.thinkingOff"),
          },
          {
            value: "on",
            label: t("library.config.taskRuntime.thinkingOn"),
          },
        ],
        (nextValue) => updateField("thinkingMode", nextValue),
      )}
      {renderNumberCard(
        t("library.config.taskRuntime.maxTokensFloor"),
        t("library.config.taskRuntime.maxTokensFloorDescription"),
        props.value.maxTokensFloor,
        (nextValue) => updateField("maxTokensFloor", nextValue),
      )}
      {renderNumberCard(
        t("library.config.taskRuntime.maxTokensCeiling"),
        t("library.config.taskRuntime.maxTokensCeilingDescription"),
        props.value.maxTokensCeiling,
        (nextValue) => updateField("maxTokensCeiling", nextValue),
      )}
      {renderNumberCard(
        t("library.config.taskRuntime.retryTokenStep"),
        t("library.config.taskRuntime.retryTokenStepDescription"),
        props.value.retryTokenStep,
        (nextValue) => updateField("retryTokenStep", nextValue),
      )}
    </div>
  );
}

export function EmptyConfigState(props: { title: string }) {
  return (
    <div
      className={cn(
        "px-4 py-8 text-sm text-muted-foreground",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        "border-dashed",
      )}
    >
      {props.title}
    </div>
  );
}

export function ConfigStandardEmptyState(props: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  title: string;
  description: string;
}) {
  const Icon = props.icon;

  return (
    <div className="flex min-h-0 flex-1 items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center">
      <Empty className="max-w-lg py-8">
        <EmptyHeader>
          <EmptyMedia className="flex h-14 w-14 items-center justify-center rounded-full border border-border/70 bg-background/80 text-muted-foreground">
            <Icon className="h-6 w-6" />
          </EmptyMedia>
          <EmptyTitle>{props.title}</EmptyTitle>
          <EmptyDescription>{props.description}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  );
}

function ConfigFieldHeader(props: { label: string; description?: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs font-medium text-foreground">{props.label}</div>
      {props.description ? (
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          {props.description}
        </div>
      ) : null}
    </div>
  );
}

export function ConfigNumberField(props: {
  label: string;
  description: string;
  value: number;
  min?: number;
  disabled?: boolean;
  inline?: boolean;
  onChange: (nextValue: number) => void;
}) {
  return (
    <div
      className={cn(
        props.inline
          ? "grid gap-3 px-3 py-2.5 md:grid-cols-2 md:items-center"
          : "px-3 py-2.5",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <ConfigFieldHeader label={props.label} description={props.description} />
      <Input
        type="number"
        min={props.min ?? 0}
        value={String(props.value)}
        disabled={props.disabled}
        onChange={(event) =>
          props.onChange(
            (props.min ?? 0) <= 0
              ? parseNonNegativeInt(event.target.value, props.value)
              : parsePositiveInt(event.target.value, props.value),
          )
        }
        className={cn(
          "h-8 border-border/70 bg-background/80",
          props.inline ? "md:w-full md:justify-self-end" : "mt-3",
        )}
      />
    </div>
  );
}

export function ConfigInputField(props: {
  label: string;
  description?: string;
  value: string;
  placeholder?: string;
  disabled?: boolean;
  inline?: boolean;
  inlineDescriptionBelow?: boolean;
  onChange: (nextValue: string) => void;
}) {
  const inlineDescriptionBelow =
    props.inline === true && props.inlineDescriptionBelow === true;

  return (
    <div
      className={cn(
        inlineDescriptionBelow
          ? "grid gap-2 px-3 py-2.5 md:grid-cols-[minmax(0,1fr)_minmax(180px,240px)]"
          : props.inline
          ? "grid gap-3 px-3 py-2.5 md:grid-cols-2 md:items-center"
          : "px-3 py-2.5",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      {inlineDescriptionBelow ? (
        <div className="min-w-0 pr-2 text-xs font-medium text-foreground">
          {props.label}
        </div>
      ) : (
        <ConfigFieldHeader label={props.label} description={props.description} />
      )}
      <Input
        value={props.value}
        disabled={props.disabled}
        onChange={(event) => props.onChange(event.target.value)}
        placeholder={props.placeholder}
        className={cn(
          "h-8 border-border/70 bg-background/80",
          inlineDescriptionBelow
            ? "md:w-full md:justify-self-end"
            : props.inline
              ? "md:w-full md:justify-self-end"
              : "mt-3",
        )}
      />
      {inlineDescriptionBelow && props.description ? (
        <div className="col-span-1 text-xs leading-5 text-muted-foreground md:col-span-2">
          {props.description}
        </div>
      ) : null}
    </div>
  );
}

export function ConfigSwitchField(props: {
  label: string;
  description: string;
  checked: boolean;
  disabled?: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div
      className={cn(
        "grid gap-3 px-3 py-2.5 md:grid-cols-[minmax(0,1fr)_auto] md:items-center",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <div className="min-w-0">
        <div className="text-xs font-medium text-foreground">{props.label}</div>
        <div className="mt-1 text-xs leading-5 text-muted-foreground">
          {props.description}
        </div>
      </div>
      <div className="flex h-8 items-center justify-end">
        <Switch
          checked={props.checked}
          disabled={props.disabled}
          onCheckedChange={props.onCheckedChange}
        />
      </div>
    </div>
  );
}

export function ConfigSelectField(props: {
  label: string;
  description?: string;
  value: string;
  disabled?: boolean;
  inline?: boolean;
  onChange: (nextValue: string) => void;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <div
      className={cn(
        props.inline
          ? "grid gap-3 px-3 py-2.5 md:grid-cols-2 md:items-center"
          : "px-3 py-2.5",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <ConfigFieldHeader label={props.label} description={props.description} />
      <Select
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        disabled={props.disabled}
        className={cn(
          "h-8 w-full min-w-0 border-border/70 bg-background/80 text-xs",
          props.inline
            ? "md:w-full md:justify-self-end"
            : props.description
              ? "mt-3"
              : "mt-2",
        )}
      >
        {props.options.map((option) => (
          <option key={`${option.value}-${option.label}`} value={option.value}>
            {option.label}
          </option>
        ))}
      </Select>
    </div>
  );
}

export function ConfigTextarea(
  props: React.TextareaHTMLAttributes<HTMLTextAreaElement>,
) {
  return (
    <textarea
      {...props}
      className={cn(
        "w-full rounded-lg border border-border/70 bg-background/80 px-3 py-2 text-sm shadow-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
        props.className,
      )}
    />
  );
}

export function ConfigTextareaField(props: {
  label: string;
  description?: string;
  value: string;
  placeholder?: string;
  disabled?: boolean;
  rows?: number;
  className?: string;
  onChange: (nextValue: string) => void;
}) {
  return (
    <div
      className={cn(
        "space-y-3 px-3 py-2.5",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      <div>
        <div className="text-xs font-medium text-foreground">{props.label}</div>
        {props.description ? (
          <div className="mt-1 text-xs leading-5 text-muted-foreground">
            {props.description}
          </div>
        ) : null}
      </div>
      <ConfigTextarea
        value={props.value}
        disabled={props.disabled}
        onChange={(event) => props.onChange(event.target.value)}
        placeholder={props.placeholder}
        rows={props.rows}
        className={cn("min-h-[88px] resize-y", props.className)}
      />
    </div>
  );
}

export function ReadOnlyInfoField(props: {
  label: string;
  value: string;
  inline?: boolean;
}) {
  return (
    <div
      className={cn(
        props.inline
          ? "grid gap-3 px-3 py-2.5 md:grid-cols-2 md:items-center"
          : "px-3 py-2.5",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
      )}
    >
      {props.inline ? (
        <div className="min-w-0 truncate text-xs font-medium text-foreground">
          {props.label}
        </div>
      ) : (
        <ConfigFieldHeader label={props.label} />
      )}
      <div
        title={props.value || "-"}
        className={cn(
          "min-w-0 text-xs leading-5 text-muted-foreground",
          props.inline ? "md:w-full md:truncate md:text-right" : "mt-2",
        )}
      >
        {props.value || "-"}
      </div>
    </div>
  );
}
