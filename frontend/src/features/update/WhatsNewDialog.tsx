import * as React from "react";
import { CheckCircle2 } from "lucide-react";

import { useDismissWhatsNew, useWhatsNew } from "@/shared/query/update";
import { useI18n } from "@/shared/i18n";
import { useUpdateStore } from "@/shared/store/update";
import { Button } from "@/shared/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/shared/ui/dialog";
import { ProductModeGlyph } from "@/shared/ui/product-mode-glyph";
import { DialogMarkdown } from "@/shared/markdown/dialog-markdown";
import { useSetupCenter } from "@/features/setup";

function useWindowVisible() {
  const [visible, setVisible] = React.useState(() =>
    typeof document !== "undefined" ? document.visibilityState === "visible" : false
  );

  React.useEffect(() => {
    const update = () => {
      setVisible(document.visibilityState === "visible");
    };
    update();
    document.addEventListener("visibilitychange", update);
    window.addEventListener("focus", update);
    window.addEventListener("blur", update);
    return () => {
      document.removeEventListener("visibilitychange", update);
      window.removeEventListener("focus", update);
      window.removeEventListener("blur", update);
    };
  }, []);

  return visible;
}

function useBlockingDialogPresent() {
  const [present, setPresent] = React.useState(false);

  React.useEffect(() => {
    const resolve = () => {
      const dialogs = Array.from(document.querySelectorAll("[role='dialog']"));
      setPresent(
        dialogs.some((node) => {
          const element = node as HTMLElement;
          return element.dataset.whatsNewDialog !== "true";
        })
      );
    };

    resolve();
    const observer = new MutationObserver(() => resolve());
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["data-state", "open", "style", "hidden"],
    });

    return () => observer.disconnect();
  }, []);

  return present;
}

export interface WhatsNewDialogProps {
  activeWindow: "main" | "settings";
  autoOpen?: boolean;
}

export function WhatsNewDialog({
  activeWindow,
  autoOpen = activeWindow === "main",
}: WhatsNewDialogProps) {
  const { t } = useI18n();
  const { data: backendNotice } = useWhatsNew();
  const dismissWhatsNew = useDismissWhatsNew();
  const { open: isSetupCenterOpen } = useSetupCenter();
  const whatsNewPreview = useUpdateStore((state) => state.whatsNewPreview);
  const clearWhatsNewPreview = useUpdateStore((state) => state.clearWhatsNewPreview);
  const isWindowVisible = useWindowVisible();
  const hasBlockingDialog = useBlockingDialogPresent();
  const [open, setOpen] = React.useState(false);
  const [presentationReady, setPresentationReady] = React.useState(false);
  const dismissedVersionRef = React.useRef("");
  const previewNotice =
    whatsNewPreview?.targetWindow === activeWindow ? whatsNewPreview.notice : null;
  const notice = previewNotice ?? backendNotice;

  React.useEffect(() => {
    setOpen(false);
    setPresentationReady(false);
    if (!notice?.version) {
      dismissedVersionRef.current = "";
      return;
    }
    const timer = window.setTimeout(() => {
      setPresentationReady(true);
    }, 900);
    return () => window.clearTimeout(timer);
  }, [notice?.version]);

  React.useEffect(() => {
    if (!notice?.version) {
      return;
    }
    if (previewNotice) {
      setOpen(true);
      return;
    }
    if (dismissedVersionRef.current === notice.version) {
      return;
    }
    if (
      !autoOpen ||
      !presentationReady ||
      !isWindowVisible ||
      isSetupCenterOpen ||
      hasBlockingDialog ||
      open
    ) {
      return;
    }
    setOpen(true);
  }, [
    activeWindow,
    autoOpen,
    hasBlockingDialog,
    isSetupCenterOpen,
    isWindowVisible,
    notice?.version,
    open,
    presentationReady,
    previewNotice,
  ]);

  const handleOpenChange = React.useCallback(
    (nextOpen: boolean) => {
      setOpen(nextOpen);
      if (nextOpen || !notice?.version) {
        return;
      }
      if (previewNotice) {
        clearWhatsNewPreview();
        return;
      }
      dismissedVersionRef.current = notice.version;
      dismissWhatsNew.mutate(notice.version);
    },
    [clearWhatsNewPreview, dismissWhatsNew, notice?.version, previewNotice]
  );

  const title = notice?.version
    ? t("whatsNew.title").replace("{version}", notice.version)
    : t("whatsNew.title").replace("{version}", "");
  const description = t("whatsNew.description").trim();

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        showCloseButton={false}
        data-whats-new-dialog="true"
        className="max-w-[min(92vw,58rem)] overflow-hidden border-0 bg-transparent p-0 shadow-none"
      >
        <div className="relative flex h-[min(80vh,38rem)] overflow-hidden rounded-[30px] border border-white/55 bg-[linear-gradient(155deg,rgba(255,255,255,0.97),rgba(248,250,252,0.94)_45%,rgba(241,245,249,0.96)_100%)] shadow-[0_40px_130px_-54px_rgba(15,23,42,0.55)] dark:border-white/10 dark:bg-[linear-gradient(155deg,rgba(15,23,42,0.98),rgba(2,6,23,0.95)_45%,rgba(15,23,42,0.98)_100%)]">
          <div className="pointer-events-none absolute inset-0">
            <div className="absolute left-[-12%] top-[-18%] h-72 w-72 rounded-full bg-[radial-gradient(circle,rgba(14,165,233,0.22),transparent_62%)] blur-3xl" />
            <div className="absolute right-[-10%] top-[-22%] h-80 w-80 rounded-full bg-[radial-gradient(circle,rgba(249,115,22,0.18),transparent_62%)] blur-3xl" />
            <div className="absolute bottom-[-28%] left-[35%] h-80 w-80 rounded-full bg-[radial-gradient(circle,rgba(34,197,94,0.14),transparent_62%)] blur-3xl" />
          </div>

          <div className="relative flex h-full min-h-0 w-full flex-col gap-6 p-6 sm:p-8">
            <DialogHeader className="shrink-0 space-y-4 text-left">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="space-y-4">
                  <div className="inline-flex items-center gap-3 rounded-full border border-slate-200/70 bg-white/80 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.22em] text-slate-600 shadow-sm backdrop-blur-sm dark:border-white/10 dark:bg-white/5 dark:text-slate-200">
                    <ProductModeGlyph className="h-7 w-7 rounded-full" iconClassName="h-3.5 w-3.5" />
                    <span>{t("whatsNew.eyebrow")}</span>
                  </div>
                  <div className={description ? "space-y-2" : "space-y-1.5"}>
                    <DialogTitle className="text-3xl font-semibold tracking-[-0.03em] text-slate-950 dark:text-white">
                      {title}
                    </DialogTitle>
                    {description ? (
                      <DialogDescription className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
                        {description}
                      </DialogDescription>
                    ) : null}
                  </div>
                </div>

                <div className="flex items-center gap-3 rounded-2xl border border-white/55 bg-white/80 px-3 py-3 shadow-[0_20px_50px_-28px_rgba(15,23,42,0.5)] backdrop-blur-xl dark:border-white/10 dark:bg-white/[0.04]">
                  <div className="relative flex h-12 w-12 items-center justify-center rounded-[18px] border border-white/45 bg-white/80 shadow-[inset_0_1px_0_rgba(255,255,255,0.75),0_10px_30px_-16px_rgba(15,23,42,0.4)] dark:border-white/10 dark:bg-white/5">
                    <CheckCircle2 className="h-5 w-5 text-emerald-600 dark:text-emerald-200" />
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-slate-950 dark:text-white">
                      {t("whatsNew.currentVersion")}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">
                      {notice?.version}
                    </div>
                  </div>
                </div>
              </div>
            </DialogHeader>

            <div className="min-h-0 flex-1 overflow-hidden rounded-[26px] border border-white/55 bg-white/78 p-5 shadow-[0_14px_50px_-36px_rgba(15,23,42,0.4)] backdrop-blur-xl dark:border-white/10 dark:bg-white/[0.05]">
              {notice?.changelog?.trim() ? (
                <DialogMarkdown
                  content={notice.changelog}
                  className="h-full max-h-none overflow-y-auto overflow-x-hidden pr-2"
                />
              ) : (
                <div className="h-full overflow-y-auto space-y-2 pr-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
                  <p>{t("whatsNew.emptyState")}</p>
                  <p className="font-medium text-slate-900 dark:text-white">
                    {t("whatsNew.versionLabel").replace("{version}", notice?.version ?? "")}
                  </p>
                </div>
              )}
            </div>

            <div className="shrink-0 flex justify-end">
              <Button variant="ghost" size="compact" onClick={() => handleOpenChange(false)}>
                {t("common.close")}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
