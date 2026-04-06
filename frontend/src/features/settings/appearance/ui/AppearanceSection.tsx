import { useEffect, useMemo, useState } from "react";
import { Moon, Monitor, Sun } from "lucide-react";

import { useFontFamilies } from "@/hooks/useFontFamilies";
import { COLOR_SCHEME_OPTIONS } from "@/lib/theme/color-schemes";
import { useI18n } from "@/shared/i18n";
import type { ColorScheme } from "@/shared/contracts/settings";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import {
  SettingsCompactListCard,
  SettingsCompactRow,
  SettingsCompactSeparator,
} from "@/shared/ui/settings-layout";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import { cn } from "@/lib/utils";

const SYSTEM_THEME_COLOR = "system";
const DEFAULT_THEME_COLOR = "#4f46e5";

const COLOR_OPTIONS = [
  { id: "system", label: "Follow system", value: SYSTEM_THEME_COLOR },
  { id: "blue", label: "Blue", value: "#3b82f6" },
  { id: "purple", label: "Purple", value: "#a855f7" },
  { id: "pink", label: "Pink", value: "#ec4899" },
  { id: "red", label: "Red", value: "#ef4444" },
  { id: "orange", label: "Orange", value: "#f97316" },
  { id: "yellow", label: "Yellow", value: "#f59e0b" },
  { id: "green", label: "Green", value: "#10b981" },
  { id: "graphite", label: "Graphite", value: "#374151" },
];

export interface AppearanceSectionProps {
  appearance: "light" | "dark" | "auto";
  fontFamily: string;
  fontSize: number;
  themeColor: string;
  colorScheme: ColorScheme;
  systemThemeColor?: string;
  onAppearanceChange: (value: "light" | "dark" | "auto") => void;
  onFontFamilyChange: (value: string) => void;
  onFontSizeChange: (value: number) => void;
  onThemeColorChange: (value: string) => void;
  onColorSchemeChange: (value: ColorScheme) => void;
}

export function AppearanceSection({
  appearance,
  fontFamily,
  fontSize,
  themeColor,
  colorScheme,
  systemThemeColor,
  onAppearanceChange,
  onFontFamilyChange,
  onFontSizeChange,
  onThemeColorChange,
  onColorSchemeChange,
}: AppearanceSectionProps) {
  const { data: fontFamilies, isLoading: isFontsLoading } = useFontFamilies();
  const { t } = useI18n();
  const [customColor, setCustomColor] = useState(themeColor ?? "");
  const [selectedColor, setSelectedColor] = useState(themeColor ?? COLOR_OPTIONS[0].value);
  const normalizedThemeColor = (themeColor ?? "").trim();
  const resolvedSystemColor = systemThemeColor?.trim() || DEFAULT_THEME_COLOR;
  const resolvedSelectedColor =
    selectedColor === SYSTEM_THEME_COLOR
      ? resolvedSystemColor
      : selectedColor || (normalizedThemeColor && normalizedThemeColor !== SYSTEM_THEME_COLOR ? normalizedThemeColor : DEFAULT_THEME_COLOR);
  const highlightColor = resolvedSelectedColor || DEFAULT_THEME_COLOR;
  const selectionShadow = `0 0 0 1px hsl(var(--border)), 0 0 0 3px ${highlightColor}`;

  useEffect(() => {
    const nextSelected = normalizedThemeColor || SYSTEM_THEME_COLOR;
    setSelectedColor(nextSelected);
    if (nextSelected === SYSTEM_THEME_COLOR) {
      setCustomColor((previous) => previous || DEFAULT_THEME_COLOR);
      return;
    }
    setCustomColor(normalizedThemeColor);
  }, [themeColor]);

  const fontOptions = useMemo(() => {
    const normalized = (fontFamilies ?? [])
      .map((value) => value.trim())
      .filter((value) => value.length > 0);
    const unique = Array.from(new Set(normalized));
    unique.sort((a, b) => a.localeCompare(b));
    return unique;
  }, [fontFamilies]);

  const selectedFont = (fontFamily ?? "").trim();
  const hasSelectedFontInList = selectedFont.length === 0 || fontOptions.includes(selectedFont);
  const clampedFontSize = Math.min(Math.max(fontSize || 16, 12), 24);
  const fontSelectClass = "w-48";
  const fontSizeWrapperClass = "w-24";
  const previewFontStack = (family: string) =>
    family ? `"${family}", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif` : undefined;

  const handleCustomColorChange = (value: string) => {
    setSelectedColor(value);
    setCustomColor(value);
    onThemeColorChange(value);
  };

  const handlePresetColorChange = (value: string) => {
    setSelectedColor(value);
    if (value !== SYSTEM_THEME_COLOR) {
      setCustomColor(value);
    }
    onThemeColorChange(value);
  };

  return (
    <SettingsCompactListCard>
      <SettingsCompactRow label={t("appearance.section.appearance")}>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="compact"
            className={cn(appearance === "light" ? "border-transparent" : "")}
            onClick={() => onAppearanceChange("light")}
            style={
              appearance === "light"
                ? {
                    boxShadow: selectionShadow,
                  }
                : undefined
            }
          >
            <Sun className="mr-1 h-4 w-4" />
            {t("appearance.option.light")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="compact"
            className={cn(appearance === "dark" ? "border-transparent" : "")}
            onClick={() => onAppearanceChange("dark")}
            style={
              appearance === "dark"
                ? {
                    boxShadow: selectionShadow,
                  }
                : undefined
            }
          >
            <Moon className="mr-1 h-4 w-4" />
            {t("appearance.option.dark")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="compact"
            className={cn(appearance === "auto" ? "border-transparent" : "")}
            onClick={() => onAppearanceChange("auto")}
            style={
              appearance === "auto"
                ? {
                    boxShadow: selectionShadow,
                  }
                : undefined
            }
          >
            <Monitor className="mr-1 h-4 w-4" />
            {t("appearance.option.auto")}
          </Button>
        </div>
      </SettingsCompactRow>

      <SettingsCompactSeparator />

      <SettingsCompactRow
        label={t("appearance.section.scheme")}
        contentClassName="max-w-[28rem] flex-wrap"
      >
        <TooltipProvider delayDuration={0}>
          {COLOR_SCHEME_OPTIONS.map((option) => {
            const isSelected = colorScheme === option.id;
            return (
              <Tooltip key={option.id}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className={cn(
                      "app-motion-surface flex min-w-[6.5rem] flex-col gap-2 rounded-lg border px-2 py-2 text-left",
                      isSelected
                        ? "border-transparent bg-accent/55"
                        : "border-border/70 bg-background/70 hover:bg-accent/35"
                    )}
                    onClick={() => onColorSchemeChange(option.id)}
                    aria-label={t(`appearance.scheme.option.${option.id}.label`)}
                    style={isSelected ? { boxShadow: selectionShadow } : undefined}
                  >
                    <span
                      className="grid h-8 w-full grid-cols-[1.1fr_1fr] overflow-hidden rounded-md border border-black/8"
                      aria-hidden="true"
                    >
                      <span style={{ backgroundColor: option.preview.sidebar }} />
                      <span className="grid grid-rows-2" style={{ backgroundColor: option.preview.shell }}>
                        <span style={{ backgroundColor: option.preview.panel }} />
                        <span style={{ backgroundColor: option.preview.accent, opacity: 0.9 }} />
                      </span>
                    </span>
                    <span className="truncate text-[11px] font-medium text-foreground">
                      {t(`appearance.scheme.option.${option.id}.label`)}
                    </span>
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">{t(`appearance.scheme.option.${option.id}.description`)}</TooltipContent>
              </Tooltip>
            );
          })}
        </TooltipProvider>
      </SettingsCompactRow>

      <SettingsCompactSeparator />

      <SettingsCompactRow label={t("appearance.section.color")}>
        <div className="flex flex-wrap items-center justify-end gap-2">
          <TooltipProvider delayDuration={0}>
            {COLOR_OPTIONS.map((option) => {
              const isSelected = selectedColor?.toLowerCase() === option.value.toLowerCase();
              const label = t(`appearance.color.option.${option.id}`);
              const displayColor = option.value === SYSTEM_THEME_COLOR ? resolvedSystemColor : option.value;
              return (
                <Tooltip key={option.value}>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      className={cn(
                        "flex h-4 w-4 items-center justify-center rounded-full border transition hover:shadow-sm",
                        isSelected ? "border-transparent" : "border-border"
                      )}
                      onClick={() => handlePresetColorChange(option.value)}
                      aria-label={label}
                      style={
                        isSelected
                          ? {
                              boxShadow: `0 0 0 1px hsl(var(--border)), 0 0 0 3px ${displayColor}`,
                            }
                          : undefined
                      }
                    >
                      <span className="h-full w-full rounded-full" style={{ backgroundColor: displayColor }} />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="top">{label}</TooltipContent>
                </Tooltip>
              );
            })}
            <div className="flex items-center gap-2">
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className={cn(
                      "flex h-4 w-4 items-center justify-center rounded-full border transition hover:shadow-sm",
                      COLOR_OPTIONS.some((opt) => selectedColor?.toLowerCase() === opt.value.toLowerCase())
                        ? "border-border"
                        : "border-transparent"
                    )}
                    onClick={() => handleCustomColorChange(customColor || DEFAULT_THEME_COLOR)}
                    aria-label={t("appearance.color.custom")}
                    style={
                      COLOR_OPTIONS.some((opt) => selectedColor?.toLowerCase() === opt.value.toLowerCase())
                        ? undefined
                        : {
                            boxShadow: `0 0 0 1px hsl(var(--border)), 0 0 0 3px ${customColor || DEFAULT_THEME_COLOR}`,
                          }
                    }
                  >
                    <span
                      className="h-full w-full rounded-full"
                      style={{ backgroundColor: customColor || DEFAULT_THEME_COLOR }}
                    />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">{t("appearance.color.custom")}</TooltipContent>
              </Tooltip>
              <input
                type="color"
                value={customColor || DEFAULT_THEME_COLOR}
                onChange={(event) => handleCustomColorChange(event.target.value)}
                className="h-4 w-4 cursor-pointer rounded-full border border-input bg-transparent p-0"
                aria-label={t("appearance.color.custom")}
              />
            </div>
          </TooltipProvider>
        </div>
      </SettingsCompactRow>

      <SettingsCompactSeparator />

      <SettingsCompactRow label={t("appearance.section.font")}>
        <Select
          id="settings-font-family"
          value={hasSelectedFontInList ? selectedFont : ""}
          onChange={(event) => onFontFamilyChange(event.target.value)}
          disabled={isFontsLoading}
          className={fontSelectClass}
        >
          <option value="">{t("appearance.font.systemDefault")}</option>
          {!hasSelectedFontInList && selectedFont.length > 0 ? (
            <option value={selectedFont} style={{ fontFamily: previewFontStack(selectedFont) }}>
              {selectedFont} {t("appearance.font.current")}
            </option>
          ) : null}
          {fontOptions.map((family) => (
            <option key={family} value={family} style={{ fontFamily: previewFontStack(family) }}>
              {family}
            </option>
          ))}
        </Select>
      </SettingsCompactRow>

      <SettingsCompactSeparator />

      <SettingsCompactRow label={t("appearance.section.size")}>
        <div className={fontSizeWrapperClass}>
          <Input
            type="number"
            min={12}
            max={24}
            step={1}
            value={clampedFontSize}
            onChange={(event) => onFontSizeChange(Number(event.target.value))}
            size="compact"
            className="w-full appearance-none text-xs"
          />
        </div>
      </SettingsCompactRow>
    </SettingsCompactListCard>
  );
}
