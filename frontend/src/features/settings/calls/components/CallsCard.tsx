import type * as React from "react";

import { Card, CardContent } from "@/shared/ui/card";
import { Separator } from "@/shared/ui/separator";
import { cn } from "@/lib/utils";

export function CallsCard({
  className,
  leftHeader,
  leftList,
  leftFooter,
  rightContent,
}: {
  className?: string;
  leftHeader?: React.ReactNode;
  leftList: React.ReactNode;
  leftFooter?: React.ReactNode;
  rightContent: React.ReactNode;
}) {
  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <Card className={cn("flex min-h-0 min-w-0 flex-1 self-stretch overflow-hidden", className)}>
        <CardContent className="flex min-h-0 min-w-0 flex-1 flex-col p-0">
          <div className="flex min-h-0 flex-1">
            <div className="flex min-h-0 w-[var(--sidebar-width)] shrink-0 flex-col">
              {leftHeader ? (
                <div className="px-[var(--app-sidebar-padding)] pt-[var(--app-sidebar-padding)]">
                  {leftHeader}
                </div>
              ) : null}
              <div
                className={cn(
                  "min-h-0 flex-1 overflow-y-auto",
                  leftHeader
                    ? "px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]"
                    : "p-[var(--app-sidebar-padding)]"
                )}
              >
                {leftList}
              </div>
              {leftFooter ? (
                <div className="border-t px-[var(--app-sidebar-padding)] py-[6px]">
                  {leftFooter}
                </div>
              ) : null}
            </div>

            <Separator orientation="vertical" className="self-stretch" />

            <div className="flex min-h-0 min-w-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-y-auto p-[var(--app-sidebar-padding)]">
                {rightContent}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
