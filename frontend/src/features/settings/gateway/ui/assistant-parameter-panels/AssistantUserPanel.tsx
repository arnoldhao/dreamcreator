import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import type { AssistantUser, UserExtraField, UserLocale } from "@/shared/store/assistant";

import { fieldClassName, fieldSelectClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

type LanguageOption = { value: string; label: string };
type RegionOption = { value: string; label: string };

const normalizeLocale = (locale?: UserLocale) => {
  if (!locale) {
    return { mode: "auto" as const, value: "", current: "" };
  }
  const mode = locale.mode === "manual" ? "manual" : "auto";
  const value = mode === "manual" ? locale.value ?? "" : "";
  return { mode, value, current: locale.current ?? "" };
};

const buildOptionList = (options: Array<{ value: string; label: string }>, current: string) => {
  if (!current) {
    return options;
  }
  if (options.some((option) => option.value === current)) {
    return options;
  }
  return [{ value: current, label: current }, ...options];
};

interface AssistantUserPanelProps {
  t: Translate;
  user: AssistantUser;
  userExtra: UserExtraField[];
  supportedLanguages: LanguageOption[];
  appLanguageLabel: string;
  resolvedTimezone: string;
  timezoneOptions: string[];
  regionOptions: RegionOption[];
  onUpdateUser: (next: AssistantUser, commit?: boolean) => void;
  onRefreshLocale: () => void;
}

export function AssistantUserPanel({
  t,
  user,
  userExtra,
  supportedLanguages,
  appLanguageLabel,
  resolvedTimezone,
  timezoneOptions,
  regionOptions,
  onUpdateUser,
  onRefreshLocale,
}: AssistantUserPanelProps) {
  const renderLanguageRow = () => {
    const locale = normalizeLocale(user.language);
    const selection = locale.mode === "manual" && locale.value ? locale.value : "auto";
    const handleChange = (value: string) => {
      if (value === "auto") {
        onUpdateUser({ ...user, language: { ...locale, mode: "auto", value: "" } }, true);
        return;
      }
      onUpdateUser({ ...user, language: { ...locale, mode: "manual", value } }, true);
    };
    const hasCustom =
      selection !== "auto" && !supportedLanguages.some((option) => option.value === selection);

    return (
      <div className="space-y-2">
        <div className={rowClassName}>
          <div className="text-sm font-medium text-muted-foreground">
            {t("settings.gateway.user.language")}
          </div>
          <Select
            value={selection}
            onChange={(event) => handleChange(event.target.value)}
            className={fieldSelectClassName}
          >
            <option value="auto">{t("settings.gateway.user.languageFollowApp")}</option>
            {hasCustom ? (
              <option value={selection}>{selection}</option>
            ) : null}
            {supportedLanguages.map((option) => (
              <option key={option.value} value={option.value}>
                {t(`settings.language.option.${option.value}`)}
              </option>
            ))}
          </Select>
        </div>
        {selection === "auto" ? (
          <div className={rowClassName}>
            <div className="text-xs text-muted-foreground">
              {t("settings.gateway.user.autoValue")}
            </div>
            <div className="text-xs text-muted-foreground">{locale.current?.trim() || appLanguageLabel}</div>
          </div>
        ) : null}
      </div>
    );
  };

  const renderTimezoneRow = () => {
    const locale = normalizeLocale(user.timezone);
    const options = buildOptionList(
      timezoneOptions.map((value) => ({ value, label: value })),
      locale.value
    );

    return (
      <div className="space-y-2">
        <div className={rowClassName}>
          <div className="text-sm font-medium text-muted-foreground">
            {t("settings.gateway.user.timezone")}
          </div>
          <Select
            value={locale.mode}
            onChange={(event) => {
              const nextMode = event.target.value === "manual" ? "manual" : "auto";
              const nextValue = nextMode === "manual" ? locale.value : "";
              onUpdateUser({ ...user, timezone: { ...locale, mode: nextMode, value: nextValue } }, true);
            }}
            className={fieldSelectClassName}
          >
            <option value="auto">{t("settings.gateway.user.localeAuto")}</option>
            <option value="manual">{t("settings.gateway.user.localeManual")}</option>
          </Select>
        </div>
        <div className={rowClassName}>
          <div className="text-xs text-muted-foreground">
            {locale.mode === "auto"
              ? t("settings.gateway.user.autoValue")
              : t("settings.gateway.user.manualValue")}
          </div>
          {locale.mode === "auto" ? (
            <div className="text-xs text-muted-foreground">
              {locale.current?.trim() || resolvedTimezone || t("settings.gateway.user.autoValueUnknown")}
            </div>
          ) : timezoneOptions.length > 0 ? (
            <Select
              value={locale.value}
              onChange={(event) =>
                onUpdateUser(
                  { ...user, timezone: { ...locale, mode: "manual", value: event.target.value } },
                  true
                )
              }
              className={fieldSelectClassName}
            >
              <option value="">{t("settings.gateway.user.timezonePlaceholder")}</option>
              {options.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          ) : (
            <Input
              value={locale.value}
              onChange={(event) =>
                onUpdateUser({ ...user, timezone: { ...locale, mode: "manual", value: event.target.value } })
              }
              onBlur={() => onUpdateUser(user, true)}
              size="compact"
              className={fieldClassName}
              placeholder={t("settings.gateway.user.timezonePlaceholder")}
            />
          )}
        </div>
      </div>
    );
  };

  const renderLocationRow = () => {
    const locale = normalizeLocale(user.location);
    const options = buildOptionList(regionOptions, locale.value);

    return (
      <div className="space-y-2">
        <div className={rowClassName}>
          <div className="text-sm font-medium text-muted-foreground">
            {t("settings.gateway.user.location")}
          </div>
          <Select
            value={locale.mode}
            onChange={(event) => {
              const nextMode = event.target.value === "manual" ? "manual" : "auto";
              const nextValue = nextMode === "manual" ? locale.value : "";
              onUpdateUser({ ...user, location: { ...locale, mode: nextMode, value: nextValue } }, true);
              if (nextMode === "auto") {
                onRefreshLocale();
              }
            }}
            className={fieldSelectClassName}
          >
            <option value="auto">{t("settings.gateway.user.localeAuto")}</option>
            <option value="manual">{t("settings.gateway.user.localeManual")}</option>
          </Select>
        </div>
        <div className={rowClassName}>
          <div className="text-xs text-muted-foreground">
            {locale.mode === "auto"
              ? t("settings.gateway.user.autoValue")
              : t("settings.gateway.user.manualValue")}
          </div>
          {locale.mode === "auto" ? (
            <div className="text-xs text-muted-foreground">
              {locale.current?.trim() || t("settings.gateway.user.autoValueUnknown")}
            </div>
          ) : regionOptions.length > 0 ? (
            <Select
              value={locale.value}
              onChange={(event) =>
                onUpdateUser(
                  { ...user, location: { ...locale, mode: "manual", value: event.target.value } },
                  true
                )
              }
              className={fieldSelectClassName}
            >
              <option value="">{t("settings.gateway.user.locationPlaceholder")}</option>
              {options.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          ) : (
            <Input
              value={locale.value}
              onChange={(event) =>
                onUpdateUser({ ...user, location: { ...locale, mode: "manual", value: event.target.value } })
              }
              onBlur={() => onUpdateUser(user, true)}
              size="compact"
              className={fieldClassName}
              placeholder={t("settings.gateway.user.locationPlaceholder")}
            />
          )}
        </div>
      </div>
    );
  };

  return (
    <div className={panelClassName}>
      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.user.name")}
        </div>
        <Input
          value={user.name ?? ""}
          onChange={(event) => onUpdateUser({ ...user, name: event.target.value })}
          onBlur={() => onUpdateUser(user, true)}
          size="compact"
          className={fieldClassName}
        />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.user.preferredAddress")}
        </div>
        <Input
          value={user.preferredAddress ?? ""}
          onChange={(event) => onUpdateUser({ ...user, preferredAddress: event.target.value })}
          onBlur={() => onUpdateUser(user, true)}
          size="compact"
          className={fieldClassName}
        />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.user.pronouns")}
        </div>
        <Input
          value={user.pronouns ?? ""}
          onChange={(event) => onUpdateUser({ ...user, pronouns: event.target.value })}
          onBlur={() => onUpdateUser(user, true)}
          size="compact"
          className={fieldClassName}
        />
      </div>

      <Separator />

      <div className="space-y-2">
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.user.notes")}
        </div>
        <textarea
          className="min-h-[110px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          value={user.notes ?? ""}
          onChange={(event) => onUpdateUser({ ...user, notes: event.target.value })}
          onBlur={() => onUpdateUser(user, true)}
          placeholder={t("settings.gateway.user.notesPlaceholder")}
        />
      </div>

      <Separator />

      {renderLanguageRow()}

      <Separator />

      {renderTimezoneRow()}

      <Separator />

      {renderLocationRow()}

      <Separator />

      <div className="space-y-2">
        <div className="flex items-start justify-between gap-3">
          <div className="text-sm font-medium text-muted-foreground">
            {t("settings.gateway.user.extra")}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="compact"
            onClick={() => onUpdateUser({ ...user, extra: [...userExtra, { key: "", value: "" }] })}
          >
            <Plus className="mr-1 h-4 w-4" />
            {t("settings.gateway.user.extraAdd")}
          </Button>
        </div>
        <div className="overflow-hidden rounded-md border border-border/60">
          <Table className="table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[calc(50%-2.5rem)] truncate">
                  {t("settings.gateway.user.extraColumnKey")}
                </TableHead>
                <TableHead className="w-[calc(50%-2.5rem)] truncate">
                  {t("settings.gateway.user.extraColumnValue")}
                </TableHead>
                <TableHead className="w-20 truncate text-right">
                  {t("settings.gateway.user.extraColumnActions")}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {userExtra.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-xs text-muted-foreground">
                    {t("settings.gateway.user.extraEmpty")}
                  </TableCell>
                </TableRow>
              ) : (
                userExtra.map((item, index) => (
                  <TableRow key={`extra-${index}`}>
                    <TableCell>
                      <Input
                        value={item.key ?? ""}
                        size="compact"
                        className="w-full !text-xs"
                        placeholder={t("settings.gateway.user.extraKey")}
                        onChange={(event) => {
                          const next = [...userExtra];
                          next[index] = { ...next[index], key: event.target.value };
                          onUpdateUser({ ...user, extra: next });
                        }}
                        onBlur={(event) => {
                          const next = [...userExtra];
                          next[index] = { ...next[index], key: event.target.value };
                          onUpdateUser({ ...user, extra: next }, true);
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={item.value ?? ""}
                        size="compact"
                        className="w-full !text-xs"
                        placeholder={t("settings.gateway.user.extraValue")}
                        onChange={(event) => {
                          const next = [...userExtra];
                          next[index] = { ...next[index], value: event.target.value };
                          onUpdateUser({ ...user, extra: next });
                        }}
                        onBlur={(event) => {
                          const next = [...userExtra];
                          next[index] = { ...next[index], value: event.target.value };
                          onUpdateUser({ ...user, extra: next }, true);
                        }}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="ghost"
                        size="compactIcon"
                        onClick={() => {
                          const next = userExtra.filter((_, itemIndex) => itemIndex !== index);
                          onUpdateUser({ ...user, extra: next }, true);
                        }}
                        aria-label={t("settings.gateway.user.extraRemove")}
                        title={t("settings.gateway.user.extraRemove")}
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
    </div>
  );
}
