import * as React from "react";

import { cn } from "@/lib/utils";

import { Card, CardContent } from "./card";
import { Separator } from "./separator";

export const SETTINGS_LIST_CARD_CLASS = "app-settings-card border";
export const SETTINGS_LIST_CARD_CONTENT_CLASS = "app-settings-card-content";
export const SETTINGS_ROW_BASE_CLASS = "app-settings-row";
export const SETTINGS_ROW_CLASS = `${SETTINGS_ROW_BASE_CLASS} items-center`;
export const SETTINGS_ROW_START_CLASS = `${SETTINGS_ROW_BASE_CLASS} app-settings-row-start`;
export const SETTINGS_ROW_LABEL_CLASS = "text-xs font-medium text-muted-foreground";
export const SETTINGS_ROW_LABEL_TRUNCATE_CLASS = "min-w-0 truncate text-xs font-medium text-muted-foreground";
export const SETTINGS_ROW_DESCRIPTION_CLASS = "text-xs text-muted-foreground";
export const SETTINGS_ROW_CONTENT_BASE_CLASS = "ml-auto flex min-w-0 justify-end gap-2 text-xs font-medium text-foreground";
export const SETTINGS_SEPARATOR_CLASS = "app-divider-soft my-1.5";
export const SETTINGS_CONTROL_WIDTH_CLASS = "w-[var(--app-settings-control-width)] min-w-0";
export const SETTINGS_WIDE_CONTROL_WIDTH_CLASS = "w-[var(--app-settings-control-width-wide)] min-w-0";
export const SETTINGS_COMPACT_LIST_CARD_CLASS = "overflow-hidden";
export const SETTINGS_COMPACT_LIST_CARD_CONTENT_CLASS = "p-0";
export const SETTINGS_COMPACT_ROW_CLASS = "app-settings-row-compact";
export const SETTINGS_COMPACT_ROW_LABEL_CLASS = "text-xs font-medium text-muted-foreground";
export const SETTINGS_COMPACT_ROW_DESCRIPTION_CLASS = "text-xs text-muted-foreground";
export const SETTINGS_COMPACT_ROW_CONTENT_CLASS = "text-xs";
export const SETTINGS_COMPACT_SEPARATOR_CLASS = "app-divider-inset my-0";

export interface SettingsListCardProps extends React.ComponentPropsWithoutRef<typeof Card> {
  contentClassName?: string;
}

export const SettingsListCard = React.forwardRef<HTMLDivElement, SettingsListCardProps>(
  ({ className, contentClassName, children, ...props }, ref) => (
    <Card ref={ref} className={cn(SETTINGS_LIST_CARD_CLASS, className)} {...props}>
      <CardContent className={cn(SETTINGS_LIST_CARD_CONTENT_CLASS, contentClassName)}>{children}</CardContent>
    </Card>
  ),
);
SettingsListCard.displayName = "SettingsListCard";

export const SettingsCompactListCard = React.forwardRef<HTMLDivElement, SettingsListCardProps>(
  ({ className, contentClassName, children, ...props }, ref) => (
    <SettingsListCard
      ref={ref}
      className={cn(SETTINGS_COMPACT_LIST_CARD_CLASS, className)}
      contentClassName={cn(SETTINGS_COMPACT_LIST_CARD_CONTENT_CLASS, contentClassName)}
      {...props}
    >
      {children}
    </SettingsListCard>
  ),
);
SettingsCompactListCard.displayName = "SettingsCompactListCard";

export interface SettingsRowProps extends React.HTMLAttributes<HTMLDivElement> {
  label: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
  align?: "center" | "start";
  labelClassName?: string;
  descriptionClassName?: string;
  contentClassName?: string;
}

export function SettingsRow({
  label,
  description,
  children,
  align = "center",
  className,
  labelClassName,
  descriptionClassName,
  contentClassName,
  ...props
}: SettingsRowProps) {
  return (
    <div
      className={cn(
        align === "start" ? SETTINGS_ROW_START_CLASS : SETTINGS_ROW_CLASS,
        className,
      )}
      {...props}
    >
      <div className={cn("min-w-0", description ? "space-y-1" : null)}>
        <div className={cn(SETTINGS_ROW_LABEL_CLASS, labelClassName)}>
          {label}
        </div>
        {description ? (
          <div className={cn(SETTINGS_ROW_DESCRIPTION_CLASS, descriptionClassName)}>
            {description}
          </div>
        ) : null}
      </div>
      <div
        className={cn(
          SETTINGS_ROW_CONTENT_BASE_CLASS,
          align === "start" ? "items-start" : "items-center",
          contentClassName,
        )}
      >
        {children}
      </div>
    </div>
  );
}

export function SettingsCompactRow({
  className,
  labelClassName,
  descriptionClassName,
  contentClassName,
  ...props
}: SettingsRowProps) {
  return (
    <SettingsRow
      className={cn(SETTINGS_COMPACT_ROW_CLASS, className)}
      labelClassName={cn(SETTINGS_COMPACT_ROW_LABEL_CLASS, labelClassName)}
      descriptionClassName={cn(SETTINGS_COMPACT_ROW_DESCRIPTION_CLASS, descriptionClassName)}
      contentClassName={cn(SETTINGS_COMPACT_ROW_CONTENT_CLASS, contentClassName)}
      {...props}
    />
  );
}

export function SettingsSeparator({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof Separator>) {
  return <Separator className={cn(SETTINGS_SEPARATOR_CLASS, className)} {...props} />;
}

export function SettingsCompactSeparator({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof Separator>) {
  return <SettingsSeparator className={cn(SETTINGS_COMPACT_SEPARATOR_CLASS, className)} {...props} />;
}
