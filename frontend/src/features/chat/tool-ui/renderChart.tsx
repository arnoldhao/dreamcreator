"use client";

import * as React from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  Cell,
  XAxis,
  YAxis,
} from "recharts";
import { makeAssistantToolUI } from "@assistant-ui/react";
import { z } from "zod";

import { cn } from "@/lib/utils";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/shared/ui/chart";
import { DASHBOARD_PANEL_SURFACE_CLASS } from "@/shared/ui/dashboard";
import { useI18n } from "@/shared/i18n";
import { ToolUIFallbackCard } from "./fallback";

type ChartPayload = {
  type?: "line" | "bar" | "area" | "pie";
  title?: string;
  description?: string;
  xKey?: string;
  yKeys?: string[];
  nameKey?: string;
  valueKey?: string;
  data?: Array<Record<string, unknown>>;
};

const chartPayloadSchema = z.object({
  type: z.enum(["line", "bar", "area", "pie"]).optional(),
  title: z.string().optional(),
  description: z.string().optional(),
  xKey: z.string().optional(),
  yKeys: z.array(z.string()).optional(),
  nameKey: z.string().optional(),
  valueKey: z.string().optional(),
  data: z.array(z.record(z.string(), z.unknown())).optional(),
});

const unwrapToolResult = (value: unknown) => {
  if (!value || typeof value !== "object") return value;
  if ("result" in value) {
    return (value as { result?: unknown }).result;
  }
  return value;
};

const resolvePayload = (args: unknown, result: unknown): ChartPayload | null => {
  const resolved = unwrapToolResult(result) ?? args;
  if (!resolved || typeof resolved !== "object") return null;
  const parsed = chartPayloadSchema.safeParse(resolved);
  if (!parsed.success) {
    return null;
  }
  return parsed.data as ChartPayload;
};

const buildChartConfig = (keys: string[]) =>
  keys.reduce<Record<string, { label?: string }>>((acc, key) => {
    acc[key] = { label: key };
    return acc;
  }, {});

const ChartCard = ({ children }: { children: React.ReactNode }) => (
  <div className={cn("min-w-0 overflow-hidden space-y-3 px-4 py-3 shadow-sm", DASHBOARD_PANEL_SURFACE_CLASS)}>
    {children}
  </div>
);

const ChartHeading = ({ title, description }: { title?: string; description?: string }) => (
  <div className="space-y-1">
    {title ? (
      <div className="break-words text-sm font-semibold text-foreground [overflow-wrap:anywhere]">{title}</div>
    ) : null}
    {description ? (
      <div className="break-words text-xs text-muted-foreground [overflow-wrap:anywhere]">{description}</div>
    ) : null}
  </div>
);

const renderCartesian = (
  type: "line" | "bar" | "area",
  data: Array<Record<string, unknown>>,
  xKey: string,
  yKeys: string[]
) => {
  const config = buildChartConfig(yKeys);
  const ChartComponent = type === "bar" ? BarChart : type === "area" ? AreaChart : LineChart;
  const series = yKeys.map((key) => {
    const color = `var(--color-${key})`;
    if (type === "bar") {
      return <Bar key={key} dataKey={key} fill={color} radius={[6, 6, 0, 0]} />;
    }
    if (type === "area") {
      return (
        <Area
          key={key}
          dataKey={key}
          stroke={color}
          fill={color}
          fillOpacity={0.25}
          type="monotone"
        />
      );
    }
    return <Line key={key} dataKey={key} stroke={color} strokeWidth={2} type="monotone" />;
  });

  return (
    <ChartContainer config={config} className="h-56">
      <ChartComponent data={data} margin={{ left: 8, right: 16, top: 8, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border/60" />
        <XAxis dataKey={xKey} className="text-xs" tickLine={false} axisLine={false} />
        <YAxis className="text-xs" tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Legend className="text-xs" />
        {series}
      </ChartComponent>
    </ChartContainer>
  );
};

const renderPie = (
  data: Array<Record<string, unknown>>,
  nameKey: string,
  valueKey: string
) => (
  <ChartContainer config={buildChartConfig([valueKey])} className="h-56">
    <PieChart>
      <ChartTooltip content={<ChartTooltipContent />} />
      <Pie data={data} dataKey={valueKey} nameKey={nameKey} innerRadius={36} outerRadius={88} paddingAngle={4}>
        {data.map((_, index) => (
          <Cell key={`cell-${index}`} fill={`hsl(var(--chart-${(index % 5) + 1}))`} />
        ))}
      </Pie>
      <Legend className="text-xs" />
    </PieChart>
  </ChartContainer>
);

export const RenderChartToolUI = makeAssistantToolUI<ChartPayload, ChartPayload>({
  toolName: "render_chart",
  render: ({ args, result, isError, toolName, toolCallId }) => {
    const { t } = useI18n();
    if (result === undefined) {
      return (
        <ChartCard>
          <div className="text-sm text-muted-foreground">
            {t("chat.tools.renderChart.loading")}
          </div>
        </ChartCard>
      );
    }

    if (isError) {
      return (
        <ChartCard>
          <div className="text-sm text-destructive">
            {t("chat.tools.renderChart.error")}
          </div>
        </ChartCard>
      );
    }

    const payload = resolvePayload(args, result);
    if (!payload) {
      return (
        <ToolUIFallbackCard
          toolName={toolName}
          toolCallId={toolCallId}
          argsText={typeof args === "string" ? args : JSON.stringify(args ?? {})}
          result={result}
          isError={isError}
        />
      );
    }

    const data = Array.isArray(payload.data) ? payload.data : [];
    if (data.length === 0) {
      return (
        <ChartCard>
          <div className="text-sm text-muted-foreground">
            {t("chat.tools.renderChart.empty")}
          </div>
        </ChartCard>
      );
    }

    const keys = Object.keys(data[0] ?? {});
    const chartType = payload.type;

    if (chartType === "pie") {
      const nameKey = payload.nameKey ?? keys[0] ?? "";
      const valueKey = payload.valueKey ?? keys[1] ?? keys[0] ?? "";
      if (!nameKey || !valueKey) {
        return (
          <ChartCard>
            <div className="text-sm text-destructive">
              {t("chat.tools.renderChart.invalid")}
            </div>
          </ChartCard>
        );
      }
      return (
        <ChartCard>
          <ChartHeading title={payload.title} description={payload.description} />
          {renderPie(data, nameKey, valueKey)}
        </ChartCard>
      );
    }

    if (!chartType) {
      return (
        <ChartCard>
          <div className="text-sm text-destructive">
            {t("chat.tools.renderChart.invalid")}
          </div>
        </ChartCard>
      );
    }

    const xKey = payload.xKey ?? keys[0] ?? "";
    const yKeys = payload.yKeys?.length ? payload.yKeys : keys.filter((key) => key !== xKey);
    if (!xKey || yKeys.length === 0) {
      return (
        <ChartCard>
          <div className="text-sm text-destructive">
            {t("chat.tools.renderChart.invalid")}
          </div>
        </ChartCard>
      );
    }

    return (
      <ChartCard>
        <ChartHeading title={payload.title} description={payload.description} />
        {renderCartesian(chartType, data, xKey, yKeys)}
      </ChartCard>
    );
  },
});
