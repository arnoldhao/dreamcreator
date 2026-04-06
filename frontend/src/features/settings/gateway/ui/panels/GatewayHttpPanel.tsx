import * as React from "react";
import { HelpCircle } from "lucide-react";

import { Card, CardContent } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import type { GatewaySettings, UpdateGatewaySettingsRequest } from "@/shared/contracts/settings";

import {
  controlClassName,
  panelClassName,
  parseCommaList,
  rowClassName,
  rowLabelClassName,
  toCommaList,
} from "../gateway-details-panel.utils";

type Translate = (key: string) => string;

interface GatewayHttpPanelProps {
  t: Translate;
  gateway: GatewaySettings | undefined;
  isDisabled: boolean;
  httpTab: "endpoints" | "files" | "images";
  onHttpTabChange: (value: "endpoints" | "files" | "images") => void;
  updateGateway: (payload: UpdateGatewaySettingsRequest) => void;
}

const renderRowLabel = (label: string, description?: string) => (
  <div className="flex min-w-0 flex-1 items-center gap-1.5">
    <span className={rowLabelClassName} title={label}>
      {label}
    </span>
    {description ? (
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
            {description}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ) : null}
  </div>
);

const renderRows = (rows: React.ReactNode[]) => (
  <div className={panelClassName}>
    {rows.map((row, index) => (
      <React.Fragment key={index}>
        {row}
        {index < rows.length - 1 ? <Separator /> : null}
      </React.Fragment>
    ))}
  </div>
);

export function GatewayHttpPanel({
  t,
  gateway,
  isDisabled,
  httpTab,
  onHttpTabChange,
  updateGateway,
}: GatewayHttpPanelProps) {
  const renderHttpEndpointsPanel = () =>
    renderRows([
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.chatCompletions"),
          t("settings.gateway.detailsPanel.http.chatCompletionsHelp")
        )}
        <Switch
          checked={gateway?.http.endpoints.chatCompletions.enabled ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) =>
            updateGateway({ http: { endpoints: { chatCompletions: { enabled: value } } } })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.responses"),
          t("settings.gateway.detailsPanel.http.responsesHelp")
        )}
        <Switch
          checked={gateway?.http.endpoints.responses.enabled ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) =>
            updateGateway({ http: { endpoints: { responses: { enabled: value } } } })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.maxBodyBytes"),
          t("settings.gateway.detailsPanel.http.maxBodyBytesHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.maxBodyBytes ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { maxBodyBytes: Number(event.target.value) || 0 } } },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.maxUrlParts"),
          t("settings.gateway.detailsPanel.http.maxUrlPartsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.maxUrlParts ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { maxUrlParts: Number(event.target.value) || 0 } } },
            })
          }
        />
      </div>,
    ]);

  const renderHttpFilesPanel = () =>
    renderRows([
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.allowUrl"),
          t("settings.gateway.detailsPanel.http.files.allowUrlHelp")
        )}
        <Switch
          checked={gateway?.http.endpoints.responses.files.allowUrl ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) =>
            updateGateway({ http: { endpoints: { responses: { files: { allowUrl: value } } } } })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.urlAllowlist"),
          t("settings.gateway.detailsPanel.http.files.urlAllowlistHelp")
        )}
        <Input
          value={toCommaList(gateway?.http.endpoints.responses.files.urlAllowlist)}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { files: { urlAllowlist: parseCommaList(event.target.value) } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.allowedMimes"),
          t("settings.gateway.detailsPanel.http.files.allowedMimesHelp")
        )}
        <Input
          value={toCommaList(gateway?.http.endpoints.responses.files.allowedMimes)}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { files: { allowedMimes: parseCommaList(event.target.value) } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.maxBytes"),
          t("settings.gateway.detailsPanel.http.files.maxBytesHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.maxBytes ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { files: { maxBytes: Number(event.target.value) || 0 } } } },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.maxChars"),
          t("settings.gateway.detailsPanel.http.files.maxCharsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.maxChars ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { files: { maxChars: Number(event.target.value) || 0 } } } },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.maxRedirects"),
          t("settings.gateway.detailsPanel.http.files.maxRedirectsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.maxRedirects ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { files: { maxRedirects: Number(event.target.value) || 0 } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.timeoutMs"),
          t("settings.gateway.detailsPanel.http.files.timeoutMsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.timeoutMs ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { files: { timeoutMs: Number(event.target.value) || 0 } } } },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.pdfMaxPages"),
          t("settings.gateway.detailsPanel.http.files.pdfMaxPagesHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.pdf.maxPages ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: {
                  responses: { files: { pdf: { maxPages: Number(event.target.value) || 0 } } },
                },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.pdfMaxPixels"),
          t("settings.gateway.detailsPanel.http.files.pdfMaxPixelsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.pdf.maxPixels ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: {
                  responses: { files: { pdf: { maxPixels: Number(event.target.value) || 0 } } },
                },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.files.pdfMinTextChars"),
          t("settings.gateway.detailsPanel.http.files.pdfMinTextCharsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.files.pdf.minTextChars ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: {
                  responses: { files: { pdf: { minTextChars: Number(event.target.value) || 0 } } },
                },
              },
            })
          }
        />
      </div>,
    ]);

  const renderHttpImagesPanel = () =>
    renderRows([
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.allowUrl"),
          t("settings.gateway.detailsPanel.http.images.allowUrlHelp")
        )}
        <Switch
          checked={gateway?.http.endpoints.responses.images.allowUrl ?? false}
          disabled={isDisabled}
          onCheckedChange={(value) =>
            updateGateway({ http: { endpoints: { responses: { images: { allowUrl: value } } } } })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.urlAllowlist"),
          t("settings.gateway.detailsPanel.http.images.urlAllowlistHelp")
        )}
        <Input
          value={toCommaList(gateway?.http.endpoints.responses.images.urlAllowlist)}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { images: { urlAllowlist: parseCommaList(event.target.value) } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.allowedMimes"),
          t("settings.gateway.detailsPanel.http.images.allowedMimesHelp")
        )}
        <Input
          value={toCommaList(gateway?.http.endpoints.responses.images.allowedMimes)}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { images: { allowedMimes: parseCommaList(event.target.value) } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.maxBytes"),
          t("settings.gateway.detailsPanel.http.images.maxBytesHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.images.maxBytes ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { images: { maxBytes: Number(event.target.value) || 0 } } } },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.maxRedirects"),
          t("settings.gateway.detailsPanel.http.images.maxRedirectsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.images.maxRedirects ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: {
                endpoints: { responses: { images: { maxRedirects: Number(event.target.value) || 0 } } },
              },
            })
          }
        />
      </div>,
      <div className={rowClassName}>
        {renderRowLabel(
          t("settings.gateway.detailsPanel.http.images.timeoutMs"),
          t("settings.gateway.detailsPanel.http.images.timeoutMsHelp")
        )}
        <Input
          type="number"
          value={gateway?.http.endpoints.responses.images.timeoutMs ?? 0}
          size="compact"
          className={controlClassName}
          disabled={isDisabled}
          onChange={(event) =>
            updateGateway({
              http: { endpoints: { responses: { images: { timeoutMs: Number(event.target.value) || 0 } } } },
            })
          }
        />
      </div>,
    ]);

  return (
    <Tabs value={httpTab} onValueChange={(value) => onHttpTabChange(value as typeof httpTab)}>
      <div className="flex justify-center">
        <TabsList className="w-fit">
          <TabsTrigger value="endpoints" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.httpTabs.endpoints")}
          </TabsTrigger>
          <TabsTrigger value="files" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.httpTabs.files")}
          </TabsTrigger>
          <TabsTrigger value="images" disabled={isDisabled}>
            {t("settings.gateway.detailsPanel.httpTabs.images")}
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="endpoints">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderHttpEndpointsPanel()}
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="files">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderHttpFilesPanel()}
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="images">
        <Card className="mt-3">
          <CardContent size="compact" className="pt-4">
            {renderHttpImagesPanel()}
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
