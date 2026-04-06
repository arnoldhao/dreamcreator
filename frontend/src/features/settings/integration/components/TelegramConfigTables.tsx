import { HelpCircle, Loader2, Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";

import type { GroupAllowEntry } from "../channels-section.utils";

type Translate = (key: string) => string;

type PairingRequestItem = {
  code: string;
  id: string;
  meta?: Record<string, string>;
  createdAt?: string;
};

const HintTooltip = ({ label }: { label: string }) => (
  <TooltipProvider delayDuration={0}>
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
          aria-label={label}
        >
          <HelpCircle className="h-3.5 w-3.5" />
        </button>
      </TooltipTrigger>
      <TooltipContent side="top" align="start">
        {label}
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
);

interface AllowFromTableProps {
  t: Translate;
  show: boolean;
  rows: string[];
  editingIndex: number | null;
  onSetEditingIndex: (index: number | null) => void;
  onRowsChange: (rows: string[]) => void;
  disabled: boolean;
}

export function AllowFromTable({
  t,
  show,
  rows,
  editingIndex,
  onSetEditingIndex,
  onRowsChange,
  disabled,
}: AllowFromTableProps) {
  if (!show) {
    return null;
  }

  const handleAddRow = () => {
    const nextIndex = rows.length;
    onRowsChange([...rows, ""]);
    onSetEditingIndex(nextIndex);
  };

  const handleChange = (index: number, value: string) => {
    onRowsChange(rows.map((entry, idx) => (idx === index ? value : entry)));
  };

  const handleRemove = (index: number) => {
    onRowsChange(rows.filter((_, idx) => idx !== index));
  };

  return (
    <>
      <Separator />
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-1 text-sm font-medium text-muted-foreground">
              {t("settings.integration.channels.config.fields.allowFrom")}
              <HintTooltip
                label={t("settings.integration.channels.config.hints.allowFrom")}
              />
            </div>
          </div>
          <Button variant="ghost" size="compact" onClick={handleAddRow} disabled={disabled}>
            <Plus className="mr-1 h-4 w-4" />
            {t("settings.integration.channels.config.actions.addRow")}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-full truncate">
                  {t("settings.integration.channels.config.allowFrom.columns.userId")}
                </TableHead>
                <TableHead className="w-16 truncate">
                  {t("settings.integration.channels.config.allowFrom.columns.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2} className="text-xs text-muted-foreground">
                    {t("settings.integration.channels.config.allowFrom.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((value, index) => (
                  <TableRow key={`allowfrom-${index}`}>
                    <TableCell>
                      {editingIndex === index || value.trim() === "" ? (
                        <Input
                          value={value}
                          onChange={(event) => handleChange(index, event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === "Escape") {
                              onSetEditingIndex(null);
                              (event.currentTarget as HTMLInputElement).blur();
                            }
                          }}
                          placeholder={t("settings.integration.channels.config.placeholders.allowFrom")}
                          size="compact"
                          className="w-full"
                          autoFocus={editingIndex === index}
                          disabled={disabled}
                        />
                      ) : (
                        <button
                          type="button"
                          className="w-full truncate text-left text-xs text-foreground"
                          onClick={() => onSetEditingIndex(index)}
                          disabled={disabled}
                        >
                          {value}
                        </button>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => handleRemove(index)}
                        disabled={disabled}
                        aria-label={t("settings.integration.channels.config.actions.removeRow")}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </>
  );
}

interface GroupAllowListTableProps {
  t: Translate;
  show: boolean;
  rows: GroupAllowEntry[];
  editingIndex: number | null;
  onSetEditingIndex: (index: number | null) => void;
  onRowsChange: (rows: GroupAllowEntry[]) => void;
  disabled: boolean;
}

export function GroupAllowListTable({
  t,
  show,
  rows,
  editingIndex,
  onSetEditingIndex,
  onRowsChange,
  disabled,
}: GroupAllowListTableProps) {
  if (!show) {
    return null;
  }

  const handleAddRow = () => {
    const nextIndex = rows.length;
    onRowsChange([...rows, { id: "", requireMention: true }]);
    onSetEditingIndex(nextIndex);
  };

  const handleChangeId = (index: number, value: string) => {
    onRowsChange(rows.map((entry, idx) => (idx === index ? { ...entry, id: value } : entry)));
  };

  const handleToggleMention = (index: number, value: boolean) => {
    onRowsChange(
      rows.map((entry, idx) => (idx === index ? { ...entry, requireMention: value } : entry))
    );
  };

  const handleRemove = (index: number) => {
    onRowsChange(rows.filter((_, idx) => idx !== index));
  };

  return (
    <>
      <Separator />
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-1 text-sm font-medium text-muted-foreground">
              {t("settings.integration.channels.config.fields.groupAllowList")}
              <HintTooltip
                label={t("settings.integration.channels.config.hints.groups")}
              />
            </div>
          </div>
          <Button variant="ghost" size="compact" onClick={handleAddRow} disabled={disabled}>
            <Plus className="mr-1 h-4 w-4" />
            {t("settings.integration.channels.config.actions.addRow")}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-full truncate">
                  {t("settings.integration.channels.config.groups.columns.groupId")}
                </TableHead>
                <TableHead className="w-40 truncate">
                  {t("settings.integration.channels.config.groups.columns.requireMention")}
                </TableHead>
                <TableHead className="w-16 truncate">
                  {t("settings.integration.channels.config.groups.columns.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-xs text-muted-foreground">
                    {t("settings.integration.channels.config.groups.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((entry, index) => (
                  <TableRow key={`group-${index}`}>
                    <TableCell>
                      {editingIndex === index || entry.id.trim() === "" ? (
                        <Input
                          value={entry.id}
                          onChange={(event) => handleChangeId(index, event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === "Escape") {
                              onSetEditingIndex(null);
                              (event.currentTarget as HTMLInputElement).blur();
                            }
                          }}
                          placeholder={t("settings.integration.channels.config.placeholders.groupId")}
                          size="compact"
                          className="w-full"
                          autoFocus={editingIndex === index}
                          disabled={disabled}
                        />
                      ) : (
                        <button
                          type="button"
                          className="w-full truncate text-left text-xs text-foreground"
                          onClick={() => onSetEditingIndex(index)}
                          disabled={disabled}
                        >
                          {entry.id}
                        </button>
                      )}
                    </TableCell>
                    <TableCell>
                      {editingIndex === index || entry.id.trim() === "" ? (
                        <Switch
                          checked={entry.requireMention}
                          onCheckedChange={(value) => handleToggleMention(index, value)}
                          disabled={disabled}
                        />
                      ) : (
                        <button
                          type="button"
                          className="w-full truncate text-left text-xs text-foreground"
                          onClick={() => onSetEditingIndex(index)}
                          disabled={disabled}
                        >
                          {entry.requireMention
                            ? t("settings.integration.channels.config.groups.values.requireMention")
                            : t("settings.integration.channels.config.groups.values.noMention")}
                        </button>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => handleRemove(index)}
                        disabled={disabled}
                        aria-label={t("settings.integration.channels.config.actions.removeRow")}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </>
  );
}

interface GroupAllowFromTableProps {
  t: Translate;
  show: boolean;
  rows: string[];
  editingIndex: number | null;
  onSetEditingIndex: (index: number | null) => void;
  onRowsChange: (rows: string[]) => void;
  disabled: boolean;
}

export function GroupAllowFromTable({
  t,
  show,
  rows,
  editingIndex,
  onSetEditingIndex,
  onRowsChange,
  disabled,
}: GroupAllowFromTableProps) {
  if (!show) {
    return null;
  }

  const handleAddRow = () => {
    const nextIndex = rows.length;
    onRowsChange([...rows, ""]);
    onSetEditingIndex(nextIndex);
  };

  const handleChange = (index: number, value: string) => {
    onRowsChange(rows.map((entry, idx) => (idx === index ? value : entry)));
  };

  const handleRemove = (index: number) => {
    onRowsChange(rows.filter((_, idx) => idx !== index));
  };

  return (
    <>
      <Separator />
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-1 text-sm font-medium text-muted-foreground">
              {t("settings.integration.channels.config.fields.groupAllowFrom")}
              <HintTooltip
                label={t("settings.integration.channels.config.hints.groupAllowFrom")}
              />
            </div>
          </div>
          <Button variant="ghost" size="compact" onClick={handleAddRow} disabled={disabled}>
            <Plus className="mr-1 h-4 w-4" />
            {t("settings.integration.channels.config.actions.addRow")}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-full truncate">
                  {t("settings.integration.channels.config.groupAllowFrom.columns.userId")}
                </TableHead>
                <TableHead className="w-16 truncate">
                  {t("settings.integration.channels.config.groupAllowFrom.columns.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2} className="text-xs text-muted-foreground">
                    {t("settings.integration.channels.config.groupAllowFrom.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((value, index) => (
                  <TableRow key={`group-allowfrom-${index}`}>
                    <TableCell>
                      {editingIndex === index || value.trim() === "" ? (
                        <Input
                          value={value}
                          onChange={(event) => handleChange(index, event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === "Escape") {
                              onSetEditingIndex(null);
                              (event.currentTarget as HTMLInputElement).blur();
                            }
                          }}
                          placeholder={t("settings.integration.channels.config.placeholders.allowFrom")}
                          size="compact"
                          className="w-full"
                          autoFocus={editingIndex === index}
                          disabled={disabled}
                        />
                      ) : (
                        <button
                          type="button"
                          className="w-full truncate text-left text-xs text-foreground"
                          onClick={() => onSetEditingIndex(index)}
                          disabled={disabled}
                        >
                          {value}
                        </button>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => handleRemove(index)}
                        disabled={disabled}
                        aria-label={t("settings.integration.channels.config.actions.removeRow")}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </>
  );
}

interface PairingRequestsPanelProps {
  t: Translate;
  show: boolean;
  requests: PairingRequestItem[];
  isLoading: boolean;
  isError: boolean;
  errorMessage: string;
  pairingBusy: boolean;
  approvePending: boolean;
  rejectPending: boolean;
  pairingBusyCode: string | null;
  onApprove: (code: string) => void;
  onReject: (code: string) => void;
  formatPairingMeta: (meta?: Record<string, string>) => string;
  formatPairingTime: (value?: string) => string;
}

export function PairingRequestsPanel({
  t,
  show,
  requests,
  isLoading,
  isError,
  errorMessage,
  pairingBusy,
  approvePending,
  rejectPending,
  pairingBusyCode,
  onApprove,
  onReject,
  formatPairingMeta,
  formatPairingTime,
}: PairingRequestsPanelProps) {
  if (!show) {
    return null;
  }

  return (
    <>
      <Separator />
      <div className="space-y-2">
        <div className="flex items-center gap-1 text-sm font-medium text-muted-foreground">
          {t("settings.integration.channels.config.pairing.title")}
          <HintTooltip
            label={t("settings.integration.channels.config.pairing.description")}
          />
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-24 truncate">
                  {t("settings.integration.channels.config.pairing.columns.code")}
                </TableHead>
                <TableHead className="w-32 truncate">
                  {t("settings.integration.channels.config.pairing.columns.userId")}
                </TableHead>
                <TableHead className="min-w-0 truncate">
                  {t("settings.integration.channels.config.pairing.columns.meta")}
                </TableHead>
                <TableHead className="w-32 truncate">
                  {t("settings.integration.channels.config.pairing.columns.requested")}
                </TableHead>
                <TableHead className="w-28 truncate text-right">
                  {t("settings.integration.channels.config.pairing.columns.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-xs text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      {t("settings.integration.channels.config.pairing.loading")}
                    </div>
                  </TableCell>
                </TableRow>
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-xs text-destructive">
                    {errorMessage ||
                      t("settings.integration.channels.config.pairing.loadError")}
                  </TableCell>
                </TableRow>
              ) : requests.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-xs text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      {t("settings.integration.channels.config.pairing.empty")}
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                requests.map((request) => {
                  const metaLabel = formatPairingMeta(request.meta);
                  const requestedAt = formatPairingTime(request.createdAt);
                  const approving = approvePending && pairingBusyCode === request.code;
                  const rejecting = rejectPending && pairingBusyCode === request.code;
                  return (
                    <TableRow key={`pairing-${request.code}`}>
                      <TableCell className="truncate font-mono text-xs" title={request.code}>
                        {request.code}
                      </TableCell>
                      <TableCell className="truncate text-xs" title={request.id}>
                        {request.id}
                      </TableCell>
                      <TableCell
                        className="min-w-0 truncate text-xs text-muted-foreground"
                        title={metaLabel || "-"}
                      >
                        {metaLabel || "-"}
                      </TableCell>
                      <TableCell className="truncate text-xs text-muted-foreground" title={requestedAt}>
                        {requestedAt}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="outline"
                            size="compact"
                            onClick={() => onApprove(request.code)}
                            disabled={pairingBusy}
                          >
                            {approving ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : null}
                            {t("settings.integration.channels.config.pairing.actions.approve")}
                          </Button>
                          <Button
                            variant="ghost"
                            size="compact"
                            onClick={() => onReject(request.code)}
                            disabled={pairingBusy}
                          >
                            {rejecting ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : null}
                            {t("settings.integration.channels.config.pairing.actions.reject")}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </>
  );
}
