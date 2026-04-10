import { Rows2, Rows3 } from "lucide-react"

import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"
import { Button } from "@/shared/ui/button"
import { DASHBOARD_CONTROL_GROUP_CLASS } from "@/shared/ui/dashboard"

import type { WorkspaceDensity } from "./types"

type WorkspaceDensityToggleProps = {
  density: WorkspaceDensity
  onDensityChange: (value: WorkspaceDensity) => void
  className?: string
}

export function WorkspaceDensityToggle({
  density,
  onDensityChange,
  className,
}: WorkspaceDensityToggleProps) {
  const { t } = useI18n()

  return (
    <div className={cn(DASHBOARD_CONTROL_GROUP_CLASS, className)}>
      <Button
        type="button"
        variant={density === "comfortable" ? "secondary" : "ghost"}
        size="compactIcon"
        aria-label={t("library.workspace.toolbar.densityComfortable")}
        title={t("library.workspace.toolbar.densityComfortable")}
        className="rounded-none border-0"
        onClick={() => onDensityChange("comfortable")}
      >
        <Rows2 className="h-3.5 w-3.5" />
      </Button>
      <Button
        type="button"
        variant={density === "compact" ? "secondary" : "ghost"}
        size="compactIcon"
        aria-label={t("library.workspace.toolbar.densityCompact")}
        title={t("library.workspace.toolbar.densityCompact")}
        className="rounded-none border-0 border-l border-border/70"
        onClick={() => onDensityChange("compact")}
      >
        <Rows3 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}
