import * as React from "react";
import { Legend, ResponsiveContainer, Tooltip, type LegendProps, type TooltipProps } from "recharts";

import { cn } from "@/lib/utils";

export type ChartConfig = Record<string, { label?: string; color?: string }>;

const ChartConfigContext = React.createContext<ChartConfig | null>(null);

const ChartContainer = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { config: ChartConfig; children: React.ReactNode }
>(({ className, config, style, children, ...props }, ref) => {
  const variables = React.useMemo(() => {
    const entries = Object.entries(config ?? {});
    return entries.reduce<Record<string, string>>((acc, [key, value], index) => {
      const color = value?.color ?? `hsl(var(--chart-${(index % 5) + 1}))`;
      acc[`--color-${key}`] = color;
      return acc;
    }, {});
  }, [config]);
  const resolvedChildren = React.isValidElement(children) ? children : <>{children}</>;

  return (
    <ChartConfigContext.Provider value={config}>
      <div
        ref={ref}
        className={cn("w-full", className)}
        style={{ ...(variables as React.CSSProperties), ...style }}
        {...props}
      >
        <ResponsiveContainer width="100%" height="100%">
          {resolvedChildren}
        </ResponsiveContainer>
      </div>
    </ChartConfigContext.Provider>
  );
});
ChartContainer.displayName = "ChartContainer";

const ChartTooltip = Tooltip;

const ChartTooltipContent = React.forwardRef<HTMLDivElement, TooltipProps<number, string>>(
  ({ active, payload, label }, ref) => {
    const config = React.useContext(ChartConfigContext) ?? {};
    if (!active || !payload || payload.length === 0) {
      return null;
    }

    return (
      <div
        ref={ref}
        className="rounded-md border border-border/70 bg-popover px-3 py-2 text-xs text-popover-foreground shadow"
      >
        {label != null ? <div className="mb-1 text-muted-foreground">{label}</div> : null}
        <div className="space-y-1">
          {payload.map((entry) => {
            const key = String(entry.dataKey ?? entry.name ?? "");
            const configEntry = config[key] ?? {};
            const color = configEntry.color ?? `var(--color-${key})`;
            const labelValue = configEntry.label ?? entry.name ?? key;
            return (
              <div key={`${key}`} className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
                  <span>{labelValue}</span>
                </div>
                <span className="font-medium">{entry.value}</span>
              </div>
            );
          })}
        </div>
      </div>
    );
  }
);
ChartTooltipContent.displayName = "ChartTooltipContent";

const ChartLegend = Legend;

type ChartLegendContentProps = React.ComponentProps<"div"> &
  Pick<LegendProps, "payload" | "verticalAlign"> & {
    hideIcon?: boolean;
    nameKey?: string;
  };

const ChartLegendContent = React.forwardRef<HTMLDivElement, ChartLegendContentProps>(
  ({ className, payload, verticalAlign = "bottom", hideIcon = false }, ref) => {
    const config = React.useContext(ChartConfigContext) ?? {};

    if (!payload || payload.length === 0) {
      return null;
    }

    return (
      <div
        ref={ref}
        className={cn(
          "flex items-center justify-center gap-4 text-xs",
          verticalAlign === "top" ? "pb-3" : "pt-3",
          className
        )}
      >
        {payload.map((entry) => {
          const key = String(entry.dataKey ?? entry.value ?? "");
          const configEntry = config[key] ?? {};
          const color = configEntry.color ?? String(entry.color ?? `var(--color-${key})`);
          const labelValue = configEntry.label ?? String(entry.value ?? key);
          return (
            <div key={`${key}`} className="flex items-center gap-1.5 text-muted-foreground">
              {hideIcon ? null : <span className="h-2.5 w-2.5 rounded-[2px]" style={{ backgroundColor: color }} />}
              <span>{labelValue}</span>
            </div>
          );
        })}
      </div>
    );
  }
);
ChartLegendContent.displayName = "ChartLegendContent";

export { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent };
