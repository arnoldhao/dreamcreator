import { ToolUIErrorBoundary } from "./error-boundary";
import { ToolUIFallbackCard } from "./fallback";
import { RenderChartToolUI } from "./renderChart";

const NON_COLLAPSIBLE_TOOL_UI_NAMES = new Set(["render_chart"]);

export const isNonCollapsibleToolUI = (toolName?: string) => {
  const normalized = toolName?.trim();
  if (!normalized) {
    return false;
  }
  return NON_COLLAPSIBLE_TOOL_UI_NAMES.has(normalized);
};

export function ToolUIRegistry() {
  return (
    <ToolUIErrorBoundary fallback={<ToolUIFallbackCard toolName="tool_ui_registry" isError />}>
      <RenderChartToolUI />
    </ToolUIErrorBoundary>
  );
}
