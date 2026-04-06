import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import type { AssistantIdentity } from "@/shared/store/assistant";

import { fieldClassName, fieldSelectClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

type SoulField = "coreTruths" | "boundaries" | "rules";

const normalizeStringRows = (items?: string[]) =>
  (items ?? [])
    .map((item) => item.trim())
    .filter(Boolean);

interface AssistantSoulPanelProps {
  t: Translate;
  identity: AssistantIdentity;
  soul: NonNullable<AssistantIdentity["soul"]>;
  vibeOptions: string[];
  defaultVibe: string;
  onUpdateIdentity: (next: AssistantIdentity, commit?: boolean) => void;
}

export function AssistantSoulPanel({
  t,
  identity,
  soul,
  vibeOptions,
  defaultVibe,
  onUpdateIdentity,
}: AssistantSoulPanelProps) {
  const renderSoulRowsSection = (
    field: SoulField,
    labels: {
      titleKey: string;
      titleFallback: string;
      addKey: string;
      addFallback: string;
      emptyKey: string;
      emptyFallback: string;
    }
  ) => {
    const rows = Array.isArray(soul[field]) ? [...(soul[field] as string[])] : [];
    const commitRows = (nextRows: string[]) => {
      onUpdateIdentity({ ...identity, soul: { ...soul, [field]: normalizeStringRows(nextRows) } }, true);
    };

    return (
      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <div className="text-sm font-medium text-muted-foreground">
            {t(labels.titleKey)}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="compact"
            onClick={() => onUpdateIdentity({ ...identity, soul: { ...soul, [field]: [...rows, ""] } })}
          >
            <Plus className="mr-1 h-4 w-4" />
            {t(labels.addKey)}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-full truncate">
                  {t("settings.gateway.soul.column.content")}
                </TableHead>
                <TableHead className="w-20 truncate text-right">
                  {t("settings.gateway.soul.column.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2} className="text-xs text-muted-foreground">
                    {t(labels.emptyKey)}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((item, index) => (
                  <TableRow key={`${field}-${index}`}>
                    <TableCell>
                      <Input
                        value={item}
                        size="compact"
                        className="w-full !text-xs"
                        onChange={(event) => {
                          const nextRows = [...rows];
                          nextRows[index] = event.target.value;
                          onUpdateIdentity({ ...identity, soul: { ...soul, [field]: nextRows } });
                        }}
                        onBlur={(event) => {
                          const nextRows = [...rows];
                          nextRows[index] = event.target.value;
                          commitRows(nextRows);
                        }}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => {
                          const nextRows = rows.filter((_, itemIndex) => itemIndex !== index);
                          commitRows(nextRows);
                        }}
                        aria-label={t("settings.gateway.soul.remove")}
                        title={t("settings.gateway.soul.remove")}
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
    );
  };

  return (
    <div className={panelClassName}>
      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.soul.vibe")}
        </div>
        {vibeOptions.length > 0 ? (
          <Select
            value={soul.vibe?.trim() || defaultVibe || vibeOptions[0]}
            onChange={(event) => onUpdateIdentity({ ...identity, soul: { ...soul, vibe: event.target.value } }, true)}
            className={fieldSelectClassName}
          >
            {vibeOptions.map((vibe) => (
              <option key={vibe} value={vibe}>
                {vibe}
              </option>
            ))}
          </Select>
        ) : (
          <Input
            value={soul.vibe ?? ""}
            onChange={(event) => onUpdateIdentity({ ...identity, soul: { ...soul, vibe: event.target.value } })}
            onBlur={() => onUpdateIdentity({ ...identity, soul }, true)}
            size="compact"
            className={fieldClassName}
          />
        )}
      </div>

      <Separator />

      {renderSoulRowsSection("coreTruths", {
        titleKey: "settings.gateway.soul.coreTruths",
        titleFallback: "Core truths",
        addKey: "settings.gateway.soul.coreTruthsAdd",
        addFallback: "Add",
        emptyKey: "settings.gateway.soul.coreTruthsEmpty",
        emptyFallback: "No core truths",
      })}

      <Separator />

      {renderSoulRowsSection("boundaries", {
        titleKey: "settings.gateway.soul.boundaries",
        titleFallback: "Boundaries",
        addKey: "settings.gateway.soul.boundariesAdd",
        addFallback: "Add",
        emptyKey: "settings.gateway.soul.boundariesEmpty",
        emptyFallback: "No boundaries",
      })}

      <Separator />

      {renderSoulRowsSection("rules", {
        titleKey: "settings.gateway.soul.rules",
        titleFallback: "Rules",
        addKey: "settings.gateway.soul.rulesAdd",
        addFallback: "Add",
        emptyKey: "settings.gateway.soul.rulesEmpty",
        emptyFallback: "No rules",
      })}

      <Separator />

      <div className="space-y-2">
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.soul.continuity")}
        </div>
        <textarea
          className="min-h-[110px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          value={soul.continuity ?? ""}
          onChange={(event) => onUpdateIdentity({ ...identity, soul: { ...soul, continuity: event.target.value } })}
          onBlur={() => onUpdateIdentity({ ...identity, soul }, true)}
          placeholder={t("settings.gateway.soul.continuityPlaceholder")}
        />
      </div>
    </div>
  );
}
