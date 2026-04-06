import * as React from "react";
import { Check, HelpCircle } from "lucide-react";

import { Badge } from "@/shared/ui/badge";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Separator } from "@/shared/ui/separator";
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

export type ToolDependencyStatus = {
  id: string;
  name: string;
  ok: boolean;
  reason: string;
  badges?: string[];
};

export function ToolDetailLayout({
  overview,
  content,
}: {
  overview: React.ReactNode;
  content: React.ReactNode;
}) {
  return (
    <div className="space-y-3">
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
        <div className="truncate text-sm font-medium text-foreground">{title}</div>
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

function ToolStatusControlBlock({
  statusBadge,
  enabledLabel,
  enabled,
  disabled,
  onEnabledChange,
}: {
  statusBadge: React.ReactNode;
  enabledLabel: string;
  enabled: boolean;
  disabled?: boolean;
  onEnabledChange: (enabled: boolean) => void;
}) {
  return (
    <div className="flex items-center gap-3">
      {statusBadge}
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground">{enabledLabel}</span>
        <Switch checked={enabled} disabled={disabled} onCheckedChange={onEnabledChange} />
      </div>
    </div>
  );
}

function ToolDependencyItem({ item }: { item: ToolDependencyStatus }) {
  const badges = (item.badges ?? []).map((entry) => entry.trim()).filter((entry) => entry !== "");
  return (
    <div className="flex min-w-0 items-center justify-between gap-3 px-3 py-2.5">
      <div className="truncate text-xs text-muted-foreground">{item.name}</div>
      {badges.length > 0 ? (
        <div className="flex flex-wrap items-center justify-end gap-1">
          {badges.map((badge, index) => (
            <Badge key={`${item.id}-${badge}-${index}`} variant="outline" className="h-5 px-1.5 text-[10px] font-medium">
              {badge}
            </Badge>
          ))}
        </div>
      ) : item.ok ? (
        <Check className="h-3.5 w-3.5 text-emerald-500" />
      ) : (
        <span className="text-xs text-destructive">{item.reason}</span>
      )}
    </div>
  );
}

function ToolDependenciesBlock({ items }: { items: ToolDependencyStatus[] }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <>
      <Separator />
      {items.map((item, index) => (
        <React.Fragment key={item.id}>
          <ToolDependencyItem item={item} />
          {index < items.length - 1 ? <Separator /> : null}
        </React.Fragment>
      ))}
    </>
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
  onEnabledChange,
  dependencies,
}: {
  title: string;
  description: string;
  descriptionLabel: string;
  statusBadge: React.ReactNode;
  enabledLabel: string;
  enabled: boolean;
  enabledDisabled?: boolean;
  onEnabledChange: (enabled: boolean) => void;
  dependencies: ToolDependencyStatus[];
}) {
  return (
    <Card>
      <CardContent className="p-0">
        <div className="flex items-start justify-between gap-4 px-3 py-3">
          <ToolIdentityBlock title={title} description={description} descriptionLabel={descriptionLabel} />
          <ToolStatusControlBlock
            statusBadge={statusBadge}
            enabledLabel={enabledLabel}
            enabled={enabled}
            disabled={enabledDisabled}
            onEnabledChange={onEnabledChange}
          />
        </div>
        <ToolDependenciesBlock items={dependencies} />
      </CardContent>
    </Card>
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
