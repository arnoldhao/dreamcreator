import * as React from "react";
import {
  ArrowRight,
  Bot,
  CheckCircle2,
  Download,
  Library,
  MessageCircle,
  Sparkles,
} from "lucide-react";

import type { CurrentUserProfile } from "@/shared/query/system";
import { useI18n } from "@/shared/i18n";
import { cn } from "@/lib/utils";
import { Button } from "@/shared/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { ProductModeGlyph } from "@/shared/ui/product-mode-glyph";
import { UserAvatar, resolveUserDisplayName, resolveUserSubtitle } from "@/shared/ui/user-avatar";

type ProductModeOption = {
  id: "full" | "download";
  enabled: boolean;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  accentClassName: string;
  glowClassName: string;
  edgeClassName: string;
  badgeClassName: string;
  featureIcons: React.ComponentType<React.SVGProps<SVGSVGElement>>[];
  featureLabelKeys: string[];
};

const PRODUCT_MODE_OPTIONS: ProductModeOption[] = [
  {
    id: "full",
    enabled: true,
    icon: Sparkles,
    accentClassName:
      "bg-[linear-gradient(160deg,rgba(250,204,21,0.16),rgba(14,165,233,0.12)_38%,rgba(251,146,60,0.16)_100%)] dark:bg-[linear-gradient(160deg,rgba(250,204,21,0.12),rgba(14,165,233,0.18)_38%,rgba(251,146,60,0.12)_100%)]",
    glowClassName:
      "bg-[radial-gradient(circle_at_15%_15%,rgba(251,191,36,0.32),transparent_38%),radial-gradient(circle_at_85%_20%,rgba(56,189,248,0.3),transparent_42%),radial-gradient(circle_at_50%_90%,rgba(249,115,22,0.24),transparent_36%)]",
    edgeClassName: "border-sky-300/45 dark:border-sky-300/20",
    badgeClassName:
      "bg-sky-500/12 text-sky-700 ring-1 ring-sky-500/20 dark:bg-sky-400/12 dark:text-sky-100 dark:ring-sky-300/20",
    featureIcons: [MessageCircle, Bot, Library],
    featureLabelKeys: [
      "app.settings.title.chat",
      "app.settings.title.gateway",
      "sidebar.nav.library",
    ],
  },
  {
    id: "download",
    enabled: false,
    icon: Download,
    accentClassName:
      "bg-[linear-gradient(155deg,rgba(34,197,94,0.16),rgba(16,185,129,0.12)_40%,rgba(251,191,36,0.14)_100%)] dark:bg-[linear-gradient(155deg,rgba(34,197,94,0.1),rgba(16,185,129,0.18)_40%,rgba(251,191,36,0.1)_100%)]",
    glowClassName:
      "bg-[radial-gradient(circle_at_18%_18%,rgba(34,197,94,0.28),transparent_36%),radial-gradient(circle_at_82%_24%,rgba(16,185,129,0.28),transparent_40%),radial-gradient(circle_at_55%_88%,rgba(250,204,21,0.22),transparent_32%)]",
    edgeClassName: "border-emerald-300/45 dark:border-emerald-300/20",
    badgeClassName:
      "bg-emerald-500/12 text-emerald-700 ring-1 ring-emerald-500/20 dark:bg-emerald-400/12 dark:text-emerald-100 dark:ring-emerald-300/20",
    featureIcons: [Download, Library],
    featureLabelKeys: [
      "library.task.download",
      "sidebar.nav.library",
    ],
  },
];

export interface ProductModeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  requireSelection?: boolean;
  enabled: boolean;
  profile?: CurrentUserProfile | null;
  onSelectMode: (enabled: boolean) => void;
}

export function ProductModeDialog({
  open,
  onOpenChange,
  requireSelection = false,
  enabled,
  profile,
  onSelectMode,
}: ProductModeDialogProps) {
  const { t } = useI18n();
  const userName = resolveUserDisplayName(profile);
  const userSubtitle = resolveUserSubtitle(profile);

  const handleSelect = (nextEnabled: boolean) => {
    onSelectMode(nextEnabled);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => (!requireSelection || nextOpen ? onOpenChange(nextOpen) : undefined)}>
      <DialogContent
        showCloseButton={!requireSelection}
        className="max-w-[min(92vw,58rem)] border-0 bg-transparent p-0 shadow-none"
        onEscapeKeyDown={requireSelection ? (event) => event.preventDefault() : undefined}
        onPointerDownOutside={requireSelection ? (event) => event.preventDefault() : undefined}
        onInteractOutside={requireSelection ? (event) => event.preventDefault() : undefined}
      >
        <div className="relative overflow-hidden rounded-[30px] border border-white/55 bg-[linear-gradient(155deg,rgba(255,255,255,0.95),rgba(247,250,252,0.9)_45%,rgba(241,245,249,0.92)_100%)] shadow-[0_35px_120px_-50px_rgba(15,23,42,0.55)] dark:border-white/10 dark:bg-[linear-gradient(155deg,rgba(15,23,42,0.98),rgba(2,6,23,0.94)_48%,rgba(15,23,42,0.98)_100%)]">
          <div className="pointer-events-none absolute inset-0">
            <div className="absolute left-[-8%] top-[-20%] h-72 w-72 rounded-full bg-[radial-gradient(circle,rgba(56,189,248,0.24),transparent_62%)] blur-3xl dark:bg-[radial-gradient(circle,rgba(56,189,248,0.18),transparent_62%)]" />
            <div className="absolute right-[-12%] top-[-16%] h-80 w-80 rounded-full bg-[radial-gradient(circle,rgba(250,204,21,0.18),transparent_62%)] blur-3xl dark:bg-[radial-gradient(circle,rgba(251,146,60,0.16),transparent_64%)]" />
            <div className="absolute bottom-[-28%] left-[28%] h-80 w-80 rounded-full bg-[radial-gradient(circle,rgba(16,185,129,0.16),transparent_64%)] blur-3xl dark:bg-[radial-gradient(circle,rgba(16,185,129,0.14),transparent_64%)]" />
          </div>

          <div className="relative space-y-8 p-6 sm:p-8">
            <DialogHeader className="space-y-4 text-left">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="space-y-4">
                  <div className="inline-flex items-center gap-3 rounded-full border border-slate-200/70 bg-white/80 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.22em] text-slate-600 shadow-sm backdrop-blur-sm dark:border-white/10 dark:bg-white/5 dark:text-slate-200">
                    <ProductModeGlyph className="h-7 w-7 rounded-full" iconClassName="h-3.5 w-3.5" />
                    <span>{t("productMode.eyebrow")}</span>
                  </div>
                  <div className="space-y-2">
                    <DialogTitle className="text-3xl font-semibold tracking-[-0.03em] text-slate-950 dark:text-white">
                      {t("productMode.title")}
                    </DialogTitle>
                    <DialogDescription className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
                      {t("productMode.description")}
                    </DialogDescription>
                  </div>
                </div>

                <div className="flex items-center gap-3 rounded-2xl border border-white/55 bg-white/80 px-3 py-3 shadow-[0_20px_50px_-28px_rgba(15,23,42,0.5)] backdrop-blur-xl dark:border-white/10 dark:bg-white/[0.04]">
                  <UserAvatar profile={profile} className="h-12 w-12 rounded-[18px]" fallbackClassName="text-xs" />
                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold text-slate-950 dark:text-white">{userName}</div>
                    <div className="truncate text-xs text-slate-500 dark:text-slate-400">
                      {userSubtitle || t("productMode.profileHint")}
                    </div>
                  </div>
                </div>
              </div>
            </DialogHeader>

            <div className="grid gap-4 lg:grid-cols-2">
              {PRODUCT_MODE_OPTIONS.map((option) => {
                const isActive = enabled === option.enabled;
                const Icon = option.icon;

                return (
                  <button
                    key={option.id}
                    type="button"
                    className={cn(
                      "group relative overflow-hidden rounded-[26px] border p-5 text-left transition-all duration-300",
                      "hover:-translate-y-0.5 hover:shadow-[0_22px_60px_-34px_rgba(15,23,42,0.45)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/70 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-slate-950",
                      option.edgeClassName,
                      option.accentClassName,
                      isActive
                        ? "shadow-[0_24px_80px_-36px_rgba(14,165,233,0.55)] ring-1 ring-white/70 dark:ring-white/10"
                        : "shadow-[0_14px_50px_-36px_rgba(15,23,42,0.4)]"
                    )}
                    onClick={() => handleSelect(option.enabled)}
                  >
                    <div className={cn("pointer-events-none absolute inset-0 opacity-90", option.glowClassName)} />
                    <div className="relative flex h-full flex-col gap-5">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex items-center gap-3">
                          {option.id === "full" ? (
                            <ProductModeGlyph className="h-12 w-12 rounded-[18px]" iconClassName="h-5 w-5" />
                          ) : (
                            <div className="relative flex h-12 w-12 items-center justify-center rounded-[18px] border border-white/45 bg-white/80 shadow-[inset_0_1px_0_rgba(255,255,255,0.75),0_10px_30px_-16px_rgba(15,23,42,0.4)] dark:border-white/10 dark:bg-white/5">
                              <Icon className="h-5 w-5 text-emerald-700 dark:text-emerald-200" />
                            </div>
                          )}
                          <div>
                            <div className="text-lg font-semibold tracking-[-0.02em] text-slate-950 dark:text-white">
                              {t(`productMode.options.${option.id}.title`)}
                            </div>
                            <div className="text-sm text-slate-600 dark:text-slate-300">
                              {t(`productMode.options.${option.id}.subtitle`)}
                            </div>
                          </div>
                        </div>

                        <div
                          className={cn(
                            "inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.16em]",
                            isActive
                              ? "bg-slate-950 text-white dark:bg-white dark:text-slate-950"
                              : option.badgeClassName
                          )}
                        >
                          {isActive ? <CheckCircle2 className="h-3.5 w-3.5" /> : null}
                          <span>
                            {isActive
                              ? t("productMode.current")
                              : t(`productMode.options.${option.id}.badge`)}
                          </span>
                        </div>
                      </div>

                      <p className="max-w-xl text-sm leading-6 text-slate-700 dark:text-slate-300">
                        {t(`productMode.options.${option.id}.description`)}
                      </p>

                      <div className="flex flex-wrap items-center gap-2">
                        {option.featureIcons.map((FeatureIcon, index) => {
                          const labelKey = option.featureLabelKeys[index];
                          if (!labelKey) {
                            return null;
                          }
                          return (
                            <span
                              key={`${option.id}-${labelKey}`}
                              className="inline-flex items-center gap-1 rounded-full border border-white/55 bg-white/75 px-2.5 py-1 text-xs text-slate-700 shadow-sm dark:border-white/10 dark:bg-white/[0.06] dark:text-slate-200"
                            >
                              <FeatureIcon className="h-3.5 w-3.5" />
                              <span>{t(labelKey)}</span>
                            </span>
                          );
                        })}
                      </div>

                      <div className="mt-auto flex items-center justify-between pt-2 text-sm font-medium text-slate-700 dark:text-slate-200">
                        <span>
                          {t(`productMode.options.${option.id}.action`)}
                        </span>
                        <ArrowRight className="h-4 w-4 transition-transform duration-300 group-hover:translate-x-1" />
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>

            <div className="flex flex-col gap-3 border-t border-slate-200/70 pt-5 text-sm text-slate-600 dark:border-white/10 dark:text-slate-300 sm:flex-row sm:items-center sm:justify-between">
              <div className="max-w-2xl">
                {requireSelection
                  ? t("productMode.firstRunHint")
                  : t("productMode.changeHint")}
              </div>
              {!requireSelection ? (
                <Button variant="ghost" size="compact" onClick={() => onOpenChange(false)}>
                  {t("productMode.close")}
                </Button>
              ) : null}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
