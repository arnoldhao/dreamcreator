import * as React from "react"

import { cn } from "@/lib/utils"

type LibraryTableSelectionCheckboxProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  "type"
> & {
  indeterminate?: boolean
}

export const LibraryTableSelectionCheckbox = React.forwardRef<
  HTMLInputElement,
  LibraryTableSelectionCheckboxProps
>(({ className, indeterminate = false, ...props }, forwardedRef) => {
  const innerRef = React.useRef<HTMLInputElement | null>(null)
  const setRefs = React.useCallback(
    (node: HTMLInputElement | null) => {
      innerRef.current = node
      if (typeof forwardedRef === "function") {
        forwardedRef(node)
        return
      }
      if (forwardedRef) {
        forwardedRef.current = node
      }
    },
    [forwardedRef],
  )

  React.useEffect(() => {
    if (!innerRef.current) {
      return
    }
    innerRef.current.indeterminate = indeterminate
  }, [indeterminate])

  return (
    <input
      {...props}
      ref={setRefs}
      type="checkbox"
      role="checkbox"
      className={cn(
        "h-4 w-4 rounded border border-border bg-background align-middle text-primary shadow-sm outline-none",
        "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      onClick={(event) => event.stopPropagation()}
    />
  )
})

LibraryTableSelectionCheckbox.displayName = "LibraryTableSelectionCheckbox"
