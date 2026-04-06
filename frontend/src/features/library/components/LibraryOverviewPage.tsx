import * as React from "react"
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"

import { CardContent, CardHeader, CardTitle } from "@/shared/ui/card"
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from "@/shared/ui/chart"
import {
  DASHBOARD_CHART_SURFACE_CLASS,
  DASHBOARD_EMPTY_PANEL_CLASS,
  MetricCard,
  PanelCard,
} from "@/shared/ui/dashboard"
import { Select } from "@/shared/ui/select"
import { cn } from "@/lib/utils"

type OverviewCardItem = {
  id: string
  label: string
  value: string
  detail: string
  icon: React.ComponentType<{ className?: string }>
}

type OverviewChartPoint = {
  label: string
  success: number
  failed: number
}

type GranularityOption = {
  value: string
  label: string
}

type LibraryOverviewPageProps = {
  cards: OverviewCardItem[]
  chartData: OverviewChartPoint[]
  chartTitle: string
  chartSuccessLabel: string
  chartFailedLabel: string
  chartGranularity: string
  chartGranularityOptions: GranularityOption[]
  onChartGranularityChange: (value: string) => void
  recentTitle: string
  recentContent?: React.ReactNode
  emptyRecentText: string
}

export function LibraryOverviewPage({
  cards,
  chartData,
  chartTitle,
  chartSuccessLabel,
  chartFailedLabel,
  chartGranularity,
  chartGranularityOptions,
  onChartGranularityChange,
  recentTitle,
  recentContent,
  emptyRecentText,
}: LibraryOverviewPageProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => {
          return (
            <MetricCard
              key={card.id}
              title={card.label}
              value={card.value}
              description={card.detail}
              icon={card.icon}
              minHeightClassName="min-h-[116px]"
              valueClassName={cn("text-2xl")}
            />
          )
        })}
      </div>

      <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[minmax(0,2fr)_minmax(300px,1fr)]">
        <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col">
          <CardHeader className="pb-2">
            <div className="flex items-start justify-between gap-3">
              <CardTitle className="text-sm font-semibold">{chartTitle}</CardTitle>
              <Select
                className="w-[132px]"
                value={chartGranularity}
                onChange={(event) => onChartGranularityChange(event.target.value)}
              >
                {chartGranularityOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            </div>
          </CardHeader>
          <CardContent className="min-h-0 flex-1 pb-3">
            <div className={DASHBOARD_CHART_SURFACE_CLASS}>
              <ChartContainer
                config={{
                  success: { label: chartSuccessLabel, color: "hsl(var(--primary))" },
                  failed: { label: chartFailedLabel, color: "hsl(var(--primary) / 0.55)" },
                }}
                className="h-full w-full"
              >
                <BarChart data={chartData} margin={{ left: 8, right: 8, top: 24, bottom: 4 }}>
                  <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border/70" />
                  <XAxis
                    dataKey="label"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={10}
                    fontSize="0.75rem"
                    minTickGap={20}
                  />
                  <YAxis
                    allowDecimals={false}
                    tickLine={false}
                    axisLine={false}
                    fontSize="0.75rem"
                    width={32}
                  />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <ChartLegend align="right" verticalAlign="top" content={<ChartLegendContent />} />
                  <Bar dataKey="success" stackId="executions" fill="var(--color-success)" radius={[0, 0, 4, 4]} />
                  <Bar dataKey="failed" stackId="executions" fill="var(--color-failed)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ChartContainer>
            </div>
          </CardContent>
        </PanelCard>

        <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">{recentTitle}</CardTitle>
          </CardHeader>
          <CardContent className="min-h-0 flex-1 overflow-hidden pb-3">
            {recentContent ? (
              <div className="h-full min-h-0 overflow-auto pr-1">
                {recentContent}
              </div>
            ) : (
              <div className={DASHBOARD_EMPTY_PANEL_CLASS}>
                {emptyRecentText}
              </div>
            )}
          </CardContent>
        </PanelCard>
      </div>
    </div>
  )
}
