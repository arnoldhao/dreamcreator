import * as React from "react"
import { ChevronLeft, ChevronRight, MoreHorizontal } from "lucide-react"

import { cn } from "@/lib/utils"

import { buttonVariants } from "./button"

const Pagination = ({ className, ...props }: React.ComponentProps<"nav">) => (
  <nav
    role="navigation"
    aria-label="pagination"
    className={cn("flex w-full justify-center", className)}
    {...props}
  />
)

const PaginationContent = React.forwardRef<HTMLUListElement, React.ComponentProps<"ul">>(
  ({ className, ...props }, ref) => (
    <ul ref={ref} className={cn("flex flex-row items-center gap-1", className)} {...props} />
  )
)
PaginationContent.displayName = "PaginationContent"

const PaginationItem = React.forwardRef<HTMLLIElement, React.ComponentProps<"li">>(
  ({ className, ...props }, ref) => <li ref={ref} className={cn(className)} {...props} />
)
PaginationItem.displayName = "PaginationItem"

type PaginationLinkProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  isActive?: boolean
}

const PaginationLink = ({ className, isActive, ...props }: PaginationLinkProps) => (
  <button
    type="button"
    aria-current={isActive ? "page" : undefined}
    className={cn(
      buttonVariants({
        variant: isActive ? "outline" : "ghost",
        size: "icon",
      }),
      "h-8 w-8 text-xs",
      className
    )}
    {...props}
  />
)

const PaginationPrevious = ({ className, children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
  <button
    type="button"
    aria-label="Go to previous page"
    className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-8 gap-1 pl-2.5 pr-2 text-xs", className)}
    {...props}
  >
    <ChevronLeft className="h-4 w-4" />
    {children}
  </button>
)

const PaginationNext = ({ className, children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
  <button
    type="button"
    aria-label="Go to next page"
    className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-8 gap-1 pl-2 pr-2.5 text-xs", className)}
    {...props}
  >
    {children}
    <ChevronRight className="h-4 w-4" />
  </button>
)

const PaginationEllipsis = ({ className, ...props }: React.ComponentProps<"span">) => (
  <span
    aria-hidden
    className={cn("flex h-8 w-8 items-center justify-center text-xs", className)}
    {...props}
  >
    <MoreHorizontal className="h-4 w-4" />
    <span className="sr-only">More pages</span>
  </span>
)

export {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
}
