import * as React from "react";
import { HelpCircle } from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import {
  SETTINGS_ROW_CONTENT_BASE_CLASS,
  SETTINGS_ROW_CLASS,
  SETTINGS_ROW_LABEL_CLASS,
  SettingsCompactListCard,
  SettingsCompactRow,
  SettingsCompactSeparator,
  SettingsSeparator,
} from "@/shared/ui/settings-layout";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { cn } from "@/lib/utils";

function ToolConfigEmptyIllustration() {
  return (
    <svg
      width="200"
      height="120"
      viewBox="0 0 200 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <path d="M30 60 L68 60" className="stroke-muted-foreground/30" strokeWidth="2" strokeLinecap="round" />
      <polygon points="66,56 74,60 66,64" className="fill-muted-foreground/30" />

      <rect
        x="76"
        y="42"
        width="56"
        height="36"
        rx="18"
        className="stroke-primary/60 fill-primary/5 dark:fill-primary/10"
        strokeWidth="2"
      />
      <circle cx="94" cy="60" r="12" className="fill-primary/40" />
      <circle cx="94" cy="60" r="6" className="fill-primary" />

      <path
        d="M134 60 Q150 60 158 48"
        className="stroke-muted-foreground/30"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
      />
      <circle cx="162" cy="44" r="3" className="fill-muted-foreground/20" />

      <path
        d="M134 60 Q150 60 158 72"
        className="stroke-muted-foreground/30"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
      />
      <circle cx="162" cy="76" r="3" className="fill-muted-foreground/20" />

      <circle cx="22" cy="60" r="2" className="fill-muted-foreground/20" />
      <circle cx="174" cy="44" r="2" className="fill-muted-foreground/15" />
      <circle cx="174" cy="76" r="2" className="fill-muted-foreground/15" />
    </svg>
  );
}

export type ToolRequirementItem = {
  id: string;
  name: string;
  value: string;
  tone?: "neutral" | "success" | "warning" | "danger";
};

export type ToolPermissionBadge = {
  id: string;
  label: string;
  tone?: "neutral" | "info" | "warning";
};

export function ToolDetailLayout({
  overview,
  content,
}: {
  overview: React.ReactNode;
  content: React.ReactNode;
}) {
  return (
    <div className="space-y-0">
      {overview}
      {content}
    </div>
  );
}

function ToolIdentityBlock({
  title,
  description,
  descriptionLabel,
}: {
  title: string;
  description: string;
  descriptionLabel: string;
}) {
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-1.5">
        <div className="truncate text-xs font-medium text-muted-foreground">{title}</div>
        <TooltipProvider delayDuration={0}>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                aria-label={descriptionLabel}
              >
                <HelpCircle className="h-3.5 w-3.5" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" align="start">
              {description}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}

function ToolHeaderMetaRow({
  label,
  value,
}: {
  label: React.ReactNode;
  value: React.ReactNode;
}) {
  return (
    <div className={SETTINGS_ROW_CLASS}>
      <div className={cn(SETTINGS_ROW_LABEL_CLASS, "min-w-0")}>{label}</div>
      <div className={cn(SETTINGS_ROW_CONTENT_BASE_CLASS, "flex-wrap text-right")}>
        {value}
      </div>
    </div>
  );
}

function ToolStatusControlBlock({
  statusBadge,
  enabledLabel,
  enabled,
  disabled,
  disabledReason,
  onEnabledChange,
}: {
  statusBadge: React.ReactNode;
  enabledLabel: string;
  enabled: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onEnabledChange: (enabled: boolean) => void;
}) {
  const switchControl = <Switch checked={enabled} disabled={disabled} onCheckedChange={onEnabledChange} />;
  return (
    <div className={cn(SETTINGS_ROW_CONTENT_BASE_CLASS, "items-center")}>
      {statusBadge}
      <div className="flex items-center gap-2">
        <span>{enabledLabel}</span>
        {disabled && disabledReason ? (
          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex cursor-not-allowed">{switchControl}</span>
              </TooltipTrigger>
              <TooltipContent side="bottom">{disabledReason}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : switchControl}
      </div>
    </div>
  );
}

function permissionBadgeClassName(tone: ToolPermissionBadge["tone"]) {
  switch (tone) {
    case "info":
      return "border-sky-500/25 bg-sky-500/10 text-sky-700 dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-200";
    case "warning":
      return "border-amber-500/25 bg-amber-500/10 text-amber-800 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100";
    case "neutral":
    default:
      return "border-border/70 bg-muted text-muted-foreground";
  }
}

function requirementValueClassName(tone: ToolRequirementItem["tone"]) {
  switch (tone) {
    case "success":
      return "text-emerald-700 dark:text-emerald-300";
    case "warning":
      return "text-amber-800 dark:text-amber-100";
    case "danger":
      return "text-destructive";
    case "neutral":
    default:
      return "text-muted-foreground";
  }
}

function ToolPermissionsValue({ badges }: { badges: ToolPermissionBadge[] }) {
  return (
    <>
      {badges.map((badge) => (
        <Badge
          key={badge.id}
          variant="outline"
          className={cn("h-5 px-1.5 text-[10px] font-medium", permissionBadgeClassName(badge.tone))}
        >
          {badge.label}
        </Badge>
      ))}
    </>
  );
}

function ToolRequirementCard({ items }: { items: ToolRequirementItem[] }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <SettingsCompactListCard>
      {items.map((item, index) => (
        <React.Fragment key={item.id}>
          {index > 0 ? <SettingsCompactSeparator /> : null}
          <SettingsCompactRow label={item.name} contentClassName="min-w-0 max-w-[60%] items-start">
            <span
              className={cn("min-w-0 shrink break-words text-right leading-5", requirementValueClassName(item.tone))}
              title={item.value}
            >
              {item.value}
            </span>
          </SettingsCompactRow>
        </React.Fragment>
      ))}
    </SettingsCompactListCard>
  );
}

export function ToolOverviewCard({
  title,
  description,
  descriptionLabel,
  statusBadge,
  enabledLabel,
  enabled,
  enabledDisabled,
  enabledDisabledReason,
  onEnabledChange,
  permissionsLabel,
  permissions,
  requirements,
}: {
  title: string;
  description: string;
  descriptionLabel: string;
  statusBadge: React.ReactNode;
  enabledLabel: string;
  enabled: boolean;
  enabledDisabled?: boolean;
  enabledDisabledReason?: string;
  onEnabledChange: (enabled: boolean) => void;
  permissionsLabel: string;
  permissions: ToolPermissionBadge[];
  requirements: ToolRequirementItem[];
}) {
  return (
    <div className="space-y-2 pb-2 text-sm">
      <div className={SETTINGS_ROW_CLASS}>
        <ToolIdentityBlock title={title} description={description} descriptionLabel={descriptionLabel} />
        <ToolStatusControlBlock
          statusBadge={statusBadge}
          enabledLabel={enabledLabel}
          enabled={enabled}
          disabled={enabledDisabled}
          disabledReason={enabledDisabledReason}
          onEnabledChange={onEnabledChange}
        />
      </div>
      <SettingsSeparator />
      <ToolHeaderMetaRow
        label={permissionsLabel}
        value={<ToolPermissionsValue badges={permissions} />}
      />
      {requirements.length > 0 ? <ToolRequirementCard items={requirements} /> : null}
    </div>
  );
}

export type ToolContentTabValue = "config" | "io";

export function ToolContentTabs({
  value,
  onValueChange,
  configLabel,
  ioLabel,
  children,
}: {
  value: ToolContentTabValue;
  onValueChange: (value: ToolContentTabValue) => void;
  configLabel: string;
  ioLabel: string;
  children: React.ReactNode;
}) {
  return (
    <Tabs
      value={value}
      onValueChange={(nextValue) => onValueChange(nextValue as ToolContentTabValue)}
    >
      <div className="flex justify-center">
        <TabsList className="w-fit">
          <TabsTrigger value="config">
            {configLabel}
          </TabsTrigger>
          <TabsTrigger value="io">
            {ioLabel}
          </TabsTrigger>
        </TabsList>
      </div>
      {children}
    </Tabs>
  );
}

export function ToolConfigTabPanel({ children }: { children: React.ReactNode }) {
  return (
    <TabsContent value="config" className="mt-3">
      {children}
    </TabsContent>
  );
}

export function ToolConfigCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

export function ToolConfigEmptyState({ title, description }: { title: string; description: string }) {
  return (
    <Empty className="py-8">
      <EmptyHeader>
        <EmptyMedia>
          <ToolConfigEmptyIllustration />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

export function ToolIOTabPanel({ children }: { children: React.ReactNode }) {
  return (
    <TabsContent value="io" className="mt-3">
      {children}
    </TabsContent>
  );
}

export function ToolIOCard({ children }: { children: React.ReactNode }) {
  return (
    <Card>
      <CardContent className="p-0">{children}</CardContent>
    </Card>
  );
}

export function ToolMethodSelectorBlock({
  rowClassName,
  label,
  control,
}: {
  rowClassName: string;
  label: React.ReactNode;
  control: React.ReactNode;
}) {
  return <div className={cn(rowClassName, "px-4 py-3")}>{label}{control}</div>;
}

export function ToolInputExampleBlock({
  title,
  labelClassName,
  payload,
}: {
  title: string;
  labelClassName: string;
  payload: string;
}) {
  return (
    <div className="space-y-2 px-4 py-3">
      <div className={labelClassName}>{title}</div>
      <pre className="overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-3 text-xs">{payload}</pre>
    </div>
  );
}

export function ToolOutputExampleBlock({
  title,
  labelClassName,
  payload,
}: {
  title: string;
  labelClassName: string;
  payload: string;
}) {
  return (
    <div className="space-y-2 px-4 py-3">
      <div className={labelClassName}>{title}</div>
      <pre className="overflow-x-auto rounded-md border border-border/60 bg-muted/20 p-3 text-xs">{payload}</pre>
    </div>
  );
}
