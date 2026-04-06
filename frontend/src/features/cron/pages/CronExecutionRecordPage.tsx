import * as React from "react";

import type { RunsViewMode } from "../model/viewStore";

type CronExecutionRecordPageProps = {
  runsViewMode: RunsViewMode;
  renderRunsTable: () => React.ReactNode;
  renderRunsCards: () => React.ReactNode;
  renderRunsPaginationControls: () => React.ReactNode;
};

export function CronExecutionRecordPage({
  runsViewMode,
  renderRunsTable,
  renderRunsCards,
  renderRunsPaginationControls,
}: CronExecutionRecordPageProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {runsViewMode === "table" ? renderRunsTable() : renderRunsCards()}
      {renderRunsPaginationControls()}
    </div>
  );
}
