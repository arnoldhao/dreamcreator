import * as React from "react"
import { Copy, FolderOpen, Image as ImageIcon } from "lucide-react"

import { messageBus } from "@/shared/message"
import { useI18n } from "@/shared/i18n"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { DashboardDialogContent, DashboardDialogHeader } from "@/shared/ui/dashboard-dialog"
import { Dialog, DialogTitle } from "@/shared/ui/dialog"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip"

import { extractExtensionFromPath, getPathBaseName } from "../utils/resourceHelpers"

type ImagePreviewTarget = {
  name?: string | null
  path?: string | null
}

type LibraryImagePreviewDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  image: ImagePreviewTarget | null
  imageUrl?: string
  onOpenFolder?: (() => void | Promise<void>) | undefined
}

export function LibraryImagePreviewDialog({
  open,
  onOpenChange,
  image,
  imageUrl,
  onOpenFolder,
}: LibraryImagePreviewDialogProps) {
  const { t } = useI18n()
  const contentRef = React.useRef<React.ElementRef<typeof DashboardDialogContent>>(null)
  const [dimensions, setDimensions] = React.useState<{ width: number; height: number } | null>(null)
  const [loadFailed, setLoadFailed] = React.useState(false)

  React.useEffect(() => {
    setDimensions(null)
    setLoadFailed(false)
  }, [image?.name, image?.path, imageUrl])

  const normalizedName = image?.name?.trim() ?? ""
  const normalizedPath = image?.path?.trim() ?? ""
  const fileName =
    getPathBaseName(normalizedPath) || getPathBaseName(normalizedName) || normalizedName || t("library.preview.imageTitle")
  const format = React.useMemo(() => {
    const extension = extractExtensionFromPath(normalizedPath) || extractExtensionFromPath(normalizedName)
    return extension ? extension.toUpperCase() : ""
  }, [normalizedName, normalizedPath])
  const dimensionText = dimensions ? `${dimensions.width} × ${dimensions.height}` : t("common.notAvailable")
  const canShowImage = Boolean(imageUrl) && !loadFailed

  const handleCopyPath = React.useCallback(async () => {
    if (!normalizedPath || typeof navigator === "undefined" || !navigator.clipboard) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.preview.copyPath"),
        description: t("library.preview.copyPathFailed"),
      })
      return
    }
    try {
      await navigator.clipboard.writeText(normalizedPath)
      messageBus.publishToast({
        intent: "success",
        title: t("library.preview.copyPath"),
        description: t("library.preview.copyPathSuccess"),
      })
    } catch {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.preview.copyPath"),
        description: t("library.preview.copyPathFailed"),
      })
    }
  }, [normalizedPath, t])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DashboardDialogContent
        ref={contentRef}
        size="workspace"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          contentRef.current?.focus()
        }}
        className="flex max-h-[90vh] w-full flex-col gap-4 overflow-hidden outline-none focus:outline-none focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0"
      >
        <DashboardDialogHeader className="min-w-0 border-b border-border/60 pb-3">
          <div className="min-w-0 space-y-2">
            <DialogTitle className="truncate">{fileName}</DialogTitle>
            <div className="flex min-w-0 flex-wrap items-center justify-start gap-2">
              <Badge variant="outline">{format || t("common.notAvailable")}</Badge>
              <Badge variant="outline">{dimensionText}</Badge>
            </div>
          </div>
        </DashboardDialogHeader>

        <div className="flex min-h-0 flex-1 flex-col items-center gap-4 overflow-hidden">
          {canShowImage ? (
            <div className="flex h-full min-h-[360px] w-full items-center justify-center overflow-auto rounded-[22px] border border-border/60 bg-gradient-to-br from-muted/55 via-background to-muted/15 px-4 py-5 sm:min-h-[520px] sm:px-8 sm:py-8">
              <div className="group relative flex items-center justify-center">
                <img
                  src={imageUrl}
                  alt={fileName}
                  className="block h-auto max-h-full w-auto max-w-full rounded-[18px] object-contain shadow-[0_28px_80px_-28px_rgba(15,23,42,0.55)] transition duration-300 ease-out group-hover:-translate-y-1 group-hover:scale-[1.01] group-hover:shadow-[0_36px_110px_-32px_rgba(15,23,42,0.62)]"
                  onLoad={(event) => {
                    const nextWidth = event.currentTarget.naturalWidth
                    const nextHeight = event.currentTarget.naturalHeight
                    if (nextWidth > 0 && nextHeight > 0) {
                      setDimensions({ width: nextWidth, height: nextHeight })
                    }
                  }}
                  onError={() => setLoadFailed(true)}
                />
              </div>
            </div>
          ) : (
            <div className="flex h-full min-h-[320px] w-full items-center justify-center rounded-[22px] border border-dashed border-border/70 bg-muted/20 text-xs text-muted-foreground sm:min-h-[420px]">
              <ImageIcon className="mr-2 h-4 w-4" />
              {t("library.preview.imageUnavailable")}
            </div>
          )}

          {normalizedPath || onOpenFolder ? (
            <TooltipProvider delayDuration={0}>
              <div className="flex flex-wrap items-center justify-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <Button
                        variant="outline"
                        size="compactIcon"
                        onClick={() => {
                          void handleCopyPath()
                        }}
                        disabled={!normalizedPath}
                        aria-label={t("library.preview.copyPath")}
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="top">{t("library.preview.copyPath")}</TooltipContent>
                </Tooltip>
                {onOpenFolder ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex">
                        <Button
                          variant="outline"
                          size="compactIcon"
                          onClick={() => {
                            void onOpenFolder()
                          }}
                          aria-label={t("library.tooltips.openFolder")}
                        >
                          <FolderOpen className="h-4 w-4" />
                        </Button>
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="top">{t("library.tooltips.openFolder")}</TooltipContent>
                  </Tooltip>
                ) : null}
              </div>
            </TooltipProvider>
          ) : null}
        </div>
      </DashboardDialogContent>
    </Dialog>
  )
}
