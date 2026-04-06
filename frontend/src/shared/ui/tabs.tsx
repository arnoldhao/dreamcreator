import * as React from "react"

import {
  Tabs,
  TabsContent,
  TabsList as BaseTabsList,
  TabsTrigger as BaseTabsTrigger,
} from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

function TabsList({
  className,
  ...props
}: React.ComponentProps<typeof BaseTabsList>) {
  return <BaseTabsList className={cn("h-9", className)} {...props} />
}

function TabsTrigger({
  className,
  ...props
}: React.ComponentProps<typeof BaseTabsTrigger>) {
  return <BaseTabsTrigger className={cn("text-xs gap-1.5", className)} {...props} />
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
