import * as React from "react";

import type { JobsViewMode } from "../model/viewStore";

type CronListPageProps = {
  jobsViewMode: JobsViewMode;
  renderJobsTable: () => React.ReactNode;
  renderJobsCards: () => React.ReactNode;
  renderJobsPaginationControls: () => React.ReactNode;
};

export function CronListPage({
  jobsViewMode,
  renderJobsTable,
  renderJobsCards,
  renderJobsPaginationControls,
}: CronListPageProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {jobsViewMode === "table" ? renderJobsTable() : renderJobsCards()}
      {renderJobsPaginationControls()}
    </div>
  );
}
