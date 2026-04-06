import * as React from "react"
import type {
  ColumnDef,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  VisibilityState,
} from "@tanstack/react-table"
import {
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from "@tanstack/react-table"
import { ChevronLeft, ChevronRight } from "lucide-react"

import { Button } from "@/shared/ui/button"
import { DASHBOARD_PANEL_SURFACE_CLASS } from "@/shared/ui/dashboard"
import { Select } from "@/shared/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table"
import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"

import type { LibraryViewMode } from "../model/types"
import { formatTemplate } from "../utils/i18n"

type LibraryTableProps<TData> = {
  viewMode: LibraryViewMode
  data: TData[]
  columns: ColumnDef<TData>[]
  columnVisibility: VisibilityState
  onColumnVisibilityChange: OnChangeFn<VisibilityState>
  rowsPerPage: number
  onRowsPerPageChange: (value: number) => void
  getRowId?: (originalRow: TData, index: number, parent?: Row<TData>) => string
  rowSelection?: RowSelectionState
  onRowSelectionChange?: OnChangeFn<RowSelectionState>
  enableRowSelection?: boolean | ((row: Row<TData>) => boolean)
}

const ROWS_PER_PAGE_OPTIONS = [10, 20, 30, 50]
const COLUMN_WIDTHS: Record<string, string> = {
  name: "w-[220px] min-w-[180px] max-w-[280px]",
  task: "w-[150px] min-w-[130px] max-w-[170px]",
  source: "w-[100px] min-w-[90px] max-w-[110px]",
  platform: "w-[120px] min-w-[100px] max-w-[140px]",
  uploader: "w-[140px] min-w-[120px] max-w-[180px]",
  status: "w-[100px] min-w-[90px] max-w-[110px]",
  progress: "w-[260px] min-w-[220px] max-w-[320px]",
  outputs: "w-[100px] min-w-[90px] max-w-[110px]",
  duration: "w-[100px] min-w-[90px] max-w-[110px]",
  publishTime: "w-[120px] min-w-[100px] max-w-[140px]",
  createdAt: "w-[100px] min-w-[90px] max-w-[110px]",
  size: "w-[100px] min-w-[90px] max-w-[110px]",
  fileFormat: "w-[100px] min-w-[90px] max-w-[110px]",
  select: "w-[44px] min-w-[44px] max-w-[44px]",
  actions: "w-[36px] min-w-[36px] max-w-[36px]",
}

export function LibraryTable<TData>({
  viewMode,
  data,
  columns,
  columnVisibility,
  onColumnVisibilityChange,
  rowsPerPage,
  onRowsPerPageChange,
  getRowId,
  rowSelection,
  onRowSelectionChange,
  enableRowSelection,
}: LibraryTableProps<TData>) {
  const { t } = useI18n()
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: rowsPerPage,
  })

  React.useEffect(() => {
    setPagination((state) => ({ ...state, pageSize: rowsPerPage }))
  }, [rowsPerPage])

  const table = useReactTable({
    data,
    columns,
    state: {
      columnVisibility,
      pagination,
      rowSelection: rowSelection ?? {},
    },
    onColumnVisibilityChange,
    onPaginationChange: setPagination,
    onRowSelectionChange,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getRowId,
    enableRowSelection,
  })

  React.useEffect(() => {
    if (table.getPageCount() === 0) {
      return
    }
    if (pagination.pageIndex >= table.getPageCount()) {
      setPagination((state) => ({ ...state, pageIndex: table.getPageCount() - 1 }))
    }
  }, [pagination.pageIndex, table, pagination.pageSize])

  const totalTemplate =
    viewMode === "task"
      ? t("library.table.totalTasks")
      : t("library.table.totalFiles")
  const totalText = formatTemplate(totalTemplate, { count: data.length })
  const rowsPerPageTemplate = t("library.table.rowsPerPage")
  const pageText = formatTemplate(t("library.table.pageOf"), {
    page: table.getState().pagination.pageIndex + 1,
    total: Math.max(table.getPageCount(), 1),
  })
  const hasRows = table.getRowModel().rows.length > 0

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className={cn("min-h-0 flex-1 overflow-hidden", DASHBOARD_PANEL_SURFACE_CLASS)}>
        <div className="relative h-full overflow-auto">
          <Table className="min-w-full table-fixed">
            <TableHeader className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={cn(
                        header.id === "actions"
                          ? "text-right"
                          : header.id === "select"
                            ? "text-center"
                            : "text-left",
                        "whitespace-nowrap overflow-hidden text-ellipsis text-xs font-semibold tracking-wide text-muted-foreground",
                        COLUMN_WIDTHS[header.column.id]
                      )}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            {hasRows ? (
              <TableBody>
                {table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() ? "selected" : undefined}
                    className="odd:bg-muted/[0.14] transition-colors hover:bg-muted/40"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cn(
                          cell.column.id === "actions"
                            ? "text-right"
                            : cell.column.id === "select"
                              ? "text-center"
                              : "text-left",
                          "whitespace-nowrap overflow-hidden text-ellipsis text-xs",
                          COLUMN_WIDTHS[cell.column.id]
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            ) : null}
          </Table>
          {!hasRows ? (
            <div className="pointer-events-none absolute inset-x-0 bottom-0 top-10 flex items-center justify-center text-sm text-muted-foreground">
              {t("library.table.noResults")}
            </div>
          ) : null}
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
        <div className="text-muted-foreground">{totalText}</div>
        <div className="flex items-center gap-2">
          <Select
            value={String(rowsPerPage)}
            onChange={(event) => onRowsPerPageChange(Number(event.target.value))}
          >
            {ROWS_PER_PAGE_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {formatTemplate(rowsPerPageTemplate, { count: option })}
              </option>
            ))}
          </Select>
          <div className="text-muted-foreground">{pageText}</div>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            aria-label={t("library.table.prevPage")}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="compactIcon"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            aria-label={t("library.table.nextPage")}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
