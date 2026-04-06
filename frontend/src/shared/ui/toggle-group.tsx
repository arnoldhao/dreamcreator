import * as React from "react"

import {
  ToggleGroup as BaseToggleGroup,
  ToggleGroupItem as BaseToggleGroupItem,
} from "@/components/ui/toggle-group"
import { cn } from "@/lib/utils"

function ToggleGroup({
  className,
  ...props
}: React.ComponentProps<typeof BaseToggleGroup>) {
  return <BaseToggleGroup className={cn("gap-1", className)} {...props} />
}

function ToggleGroupItem({
  className,
  ...props
}: React.ComponentProps<typeof BaseToggleGroupItem>) {
  return <BaseToggleGroupItem className={cn("text-xs", className)} {...props} />
}

export { ToggleGroup, ToggleGroupItem }
