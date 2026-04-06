import * as React from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { SectionCard } from "@/shared/ui/SectionCard";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { cn } from "@/lib/utils";

import {
  COMMAND_SETTING_OPTIONS,
  type CommandSettingValue,
  type TelegramCustomCommand,
} from "../channels-section.utils";

type Translate = (key: string) => string;

interface TelegramMenuConfigCardProps {
  t: Translate;
  busy: boolean;
  controlClassName: string;
  nativeSetting: CommandSettingValue;
  nativeSkillsSetting: CommandSettingValue;
  customCommands: TelegramCustomCommand[];
  customCommandsEditingIndex: number | null;
  customCommandsError: string | null;
  onChangeCommandSetting: (key: "native" | "nativeSkills", value: CommandSettingValue) => void;
  onAddCustomCommand: () => void;
  onRemoveCustomCommand: (index: number) => void;
  onCustomCommandChange: (
    index: number,
    key: keyof TelegramCustomCommand,
    value: string
  ) => void;
  onSetCustomCommandsEditingIndex: (index: number | null) => void;
  onCommitCustomCommands: () => void;
}

export function TelegramMenuConfigCard({
  t,
  busy,
  controlClassName,
  nativeSetting,
  nativeSkillsSetting,
  customCommands,
  customCommandsEditingIndex,
  customCommandsError,
  onChangeCommandSetting,
  onAddCustomCommand,
  onRemoveCustomCommand,
  onCustomCommandChange,
  onSetCustomCommandsEditingIndex,
  onCommitCustomCommands,
}: TelegramMenuConfigCardProps) {
  const handleCustomCommandsBlur = (event: React.FocusEvent<HTMLDivElement>) => {
    const nextTarget = event.relatedTarget as Node | null;
    if (nextTarget && event.currentTarget.contains(nextTarget)) {
      return;
    }
    onSetCustomCommandsEditingIndex(null);
    onCommitCustomCommands();
  };

  return (
    <SectionCard contentClassName="space-y-3 px-3 py-2">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">
          {t("settings.integration.channels.menu.fields.native")}
        </span>
        <Select
          value={nativeSetting}
          onChange={(event) => onChangeCommandSetting("native", event.target.value as CommandSettingValue)}
          className={controlClassName}
          disabled={busy}
        >
          {COMMAND_SETTING_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.labelKey)}
            </option>
          ))}
        </Select>
      </div>
      <Separator />
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">
          {t("settings.integration.channels.menu.fields.nativeSkills")}
        </span>
        <Select
          value={nativeSkillsSetting}
          onChange={(event) =>
            onChangeCommandSetting("nativeSkills", event.target.value as CommandSettingValue)
          }
          className={controlClassName}
          disabled={busy}
        >
          {COMMAND_SETTING_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.labelKey)}
            </option>
          ))}
        </Select>
      </div>
      <Separator />
      <div className="space-y-2" onBlur={handleCustomCommandsBlur}>
        <div className="flex items-center justify-between">
          <span className="min-w-0 truncate text-sm font-medium text-muted-foreground">
            {t("settings.integration.channels.menu.fields.customCommands")}
          </span>
          <Button variant="ghost" size="compact" onClick={onAddCustomCommand} disabled={busy}>
            <Plus className="h-4 w-4" />
            {t("settings.integration.channels.menu.actions.add")}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-40 truncate">
                  {t("settings.integration.channels.menu.columns.command")}
                </TableHead>
                <TableHead className="w-full truncate">
                  {t("settings.integration.channels.menu.columns.description")}
                </TableHead>
                <TableHead className="w-16 truncate">
                  {t("settings.integration.channels.menu.columns.actions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {customCommands.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-xs text-muted-foreground">
                    {t("settings.integration.channels.menu.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                customCommands.map((command, index) => {
                  const isEditing = customCommandsEditingIndex === index || command.command.trim() === "";
                  const description = command.description.trim();
                  return (
                    <TableRow key={`custom-command-${index}`}>
                      <TableCell>
                        {isEditing ? (
                          <Input
                            value={command.command}
                            placeholder={t("settings.integration.channels.menu.placeholders.command")}
                            onChange={(event) =>
                              onCustomCommandChange(index, "command", event.target.value)
                            }
                            onKeyDown={(event) => {
                              if (event.key === "Enter" || event.key === "Escape") {
                                onSetCustomCommandsEditingIndex(null);
                                (event.currentTarget as HTMLInputElement).blur();
                              }
                            }}
                            size="compact"
                            className="w-full"
                            autoFocus={customCommandsEditingIndex === index}
                            disabled={busy}
                          />
                        ) : (
                          <button
                            type="button"
                            className="w-full truncate text-left text-xs text-foreground"
                            onClick={() => onSetCustomCommandsEditingIndex(index)}
                            disabled={busy}
                          >
                            {command.command}
                          </button>
                        )}
                      </TableCell>
                      <TableCell>
                        {isEditing ? (
                          <Input
                            value={command.description}
                            placeholder={t("settings.integration.channels.menu.placeholders.description")}
                            onChange={(event) =>
                              onCustomCommandChange(index, "description", event.target.value)
                            }
                            onKeyDown={(event) => {
                              if (event.key === "Enter" || event.key === "Escape") {
                                onSetCustomCommandsEditingIndex(null);
                                (event.currentTarget as HTMLInputElement).blur();
                              }
                            }}
                            size="compact"
                            className="w-full"
                            disabled={busy}
                          />
                        ) : (
                          <button
                            type="button"
                            className={cn(
                              "w-full truncate text-left text-xs",
                              description ? "text-foreground" : "text-muted-foreground"
                            )}
                            onClick={() => onSetCustomCommandsEditingIndex(index)}
                            disabled={busy}
                          >
                            {description ||
                              t("settings.integration.channels.menu.placeholders.description")}
                          </button>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="compactIcon"
                          onClick={() => onRemoveCustomCommand(index)}
                          disabled={busy}
                          aria-label={t("settings.integration.channels.config.actions.removeRow")}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
        {customCommandsError ? <div className="text-xs text-destructive">{customCommandsError}</div> : null}
      </div>
    </SectionCard>
  );
}
