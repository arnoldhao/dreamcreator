import * as React from "react"

import {
  Item as BaseItem,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
} from "@/components/ui/item"
import { cn } from "@/lib/utils"

type BaseItemProps = React.ComponentProps<typeof BaseItem>
type AppItemSize = "compact"

export type ItemProps = Omit<BaseItemProps, "size"> & {
  size?: BaseItemProps["size"] | AppItemSize
}

function Item({ size, className, ...props }: ItemProps) {
  const compactClass = size === "compact" ? "h-7 gap-2 px-2 py-0" : undefined
  const mappedSize = size === "compact" ? "sm" : size

  return (
    <BaseItem
      size={mappedSize as BaseItemProps["size"]}
      className={cn(compactClass, className)}
      {...props}
    />
  )
}

export {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
}
