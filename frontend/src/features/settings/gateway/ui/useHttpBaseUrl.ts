import * as React from "react";
import { Call } from "@wailsio/runtime";

let cachedHttpBaseUrl = "";
let pendingHttpBaseUrl: Promise<string> | null = null;

export function useHttpBaseUrl() {
  const [httpBaseUrl, setHttpBaseUrl] = React.useState(cachedHttpBaseUrl);

  React.useEffect(() => {
    if (cachedHttpBaseUrl) {
      setHttpBaseUrl(cachedHttpBaseUrl);
      return;
    }
    if (!pendingHttpBaseUrl) {
      pendingHttpBaseUrl = Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.HTTPBaseURL")
        .then((resolved) => {
          cachedHttpBaseUrl = String(resolved ?? "");
          return cachedHttpBaseUrl;
        })
        .catch(() => "")
        .finally(() => {
          pendingHttpBaseUrl = null;
        });
    }
    pendingHttpBaseUrl
      ?.then((resolved) => setHttpBaseUrl(resolved))
      .catch(() => setHttpBaseUrl(""));
  }, []);

  return httpBaseUrl;
}
