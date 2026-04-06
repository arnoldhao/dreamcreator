import * as React from "react";

import {
  Table as BaseTable,
  TableHeader as BaseTableHeader,
  TableBody as BaseTableBody,
  TableFooter as BaseTableFooter,
  TableHead as BaseTableHead,
  TableRow as BaseTableRow,
  TableCell as BaseTableCell,
  TableCaption as BaseTableCaption,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

const Table = React.forwardRef<
  React.ElementRef<typeof BaseTable>,
  React.ComponentPropsWithoutRef<typeof BaseTable>
>(({ className, ...props }, ref) => (
  <BaseTable ref={ref} className={cn("text-xs", className)} {...props} />
));
Table.displayName = "Table";

const TableHeader = React.forwardRef<
  React.ElementRef<typeof BaseTableHeader>,
  React.ComponentPropsWithoutRef<typeof BaseTableHeader>
>(({ className, ...props }, ref) => (
  <BaseTableHeader ref={ref} className={cn(className)} {...props} />
));
TableHeader.displayName = "TableHeader";

const TableBody = React.forwardRef<
  React.ElementRef<typeof BaseTableBody>,
  React.ComponentPropsWithoutRef<typeof BaseTableBody>
>(({ className, ...props }, ref) => (
  <BaseTableBody ref={ref} className={cn(className)} {...props} />
));
TableBody.displayName = "TableBody";

const TableFooter = React.forwardRef<
  React.ElementRef<typeof BaseTableFooter>,
  React.ComponentPropsWithoutRef<typeof BaseTableFooter>
>(({ className, ...props }, ref) => (
  <BaseTableFooter ref={ref} className={cn(className)} {...props} />
));
TableFooter.displayName = "TableFooter";

const TableRow = React.forwardRef<
  React.ElementRef<typeof BaseTableRow>,
  React.ComponentPropsWithoutRef<typeof BaseTableRow>
>(({ className, ...props }, ref) => (
  <BaseTableRow ref={ref} className={cn("app-motion-color", className)} {...props} />
));
TableRow.displayName = "TableRow";

const TableHead = React.forwardRef<
  React.ElementRef<typeof BaseTableHead>,
  React.ComponentPropsWithoutRef<typeof BaseTableHead>
>(({ className, ...props }, ref) => (
  <BaseTableHead
    ref={ref}
    className={cn("h-9 px-3 text-xs font-medium uppercase tracking-[0.06em] text-muted-foreground", className)}
    {...props}
  />
));
TableHead.displayName = "TableHead";

const TableCell = React.forwardRef<
  React.ElementRef<typeof BaseTableCell>,
  React.ComponentPropsWithoutRef<typeof BaseTableCell>
>(({ className, ...props }, ref) => (
  <BaseTableCell ref={ref} className={cn("px-3 py-2.5 leading-[1.45]", className)} {...props} />
));
TableCell.displayName = "TableCell";

const TableCaption = React.forwardRef<
  React.ElementRef<typeof BaseTableCaption>,
  React.ComponentPropsWithoutRef<typeof BaseTableCaption>
>(({ className, ...props }, ref) => (
  <BaseTableCaption ref={ref} className={cn("mt-4 text-xs text-muted-foreground", className)} {...props} />
));
TableCaption.displayName = "TableCaption";

export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
};
