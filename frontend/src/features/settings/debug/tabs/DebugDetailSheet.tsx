import type { ReactNode } from "react";

import { cn } from "@/lib/utils";
import { PanelCard } from "@/shared/ui/dashboard";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@/shared/ui/sheet";

export type DebugDetailField = {
  label: string;
  value: ReactNode;
  valueClassName?: string;
};

export function DebugDetailSheet(props: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  description: string;
  headerMeta?: ReactNode;
  fields?: DebugDetailField[];
  children?: ReactNode;
}) {
  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent
        side="right"
        showCloseButton={false}
        className="flex h-full w-[420px] flex-col gap-0 overflow-hidden p-4 sm:max-w-[420px]"
      >
        <PanelCard tone="solid" className="flex min-h-0 w-full flex-1 flex-col overflow-hidden">
          <div className="border-b border-border/70 px-4 py-3">
            <div className="space-y-2">
              <div className="min-w-0">
                <SheetTitle className="truncate text-base font-semibold text-foreground">{props.title}</SheetTitle>
                <SheetDescription className="sr-only">{props.description}</SheetDescription>
              </div>
              {props.headerMeta ? <div className="flex flex-wrap items-center gap-2">{props.headerMeta}</div> : null}
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
            <div className="space-y-3">
              {props.fields && props.fields.length > 0 ? (
                <div className="overflow-hidden rounded-md border">
                  <div className="divide-y divide-border/70">
                    {props.fields.map((field) => (
                      <div key={field.label} className="space-y-1.5 px-4 py-3">
                        <div className="text-[11px] font-medium text-muted-foreground">{field.label}</div>
                        <div className={cn("font-mono text-[11px] text-foreground", field.valueClassName)}>
                          {field.value}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {props.children}
            </div>
          </div>
        </PanelCard>
      </SheetContent>
    </Sheet>
  );
}
