import * as React from "react";
import type { PropsWithChildren } from "react";
import { Call } from "@wailsio/runtime";
import {
  AssistantRuntimeProvider as BaseAssistantRuntimeProvider,
  useLocalRuntime,
  unstable_useRemoteThreadListRuntime,
  type AssistantRuntime,
} from "@assistant-ui/react";

import { useAssistants } from "@/shared/query/assistant";
import { REALTIME_TOPICS, registerTopic, subscribeGatewayEvents } from "@/shared/realtime";
import { useChatRuntimeStore } from "@/shared/store/chat-runtime";
import { useThreadStore } from "@/shared/store/threads";
import { useThreadListAdapter } from "./thread-list-adapter";
import { chatFeatureFlags } from "./feature-flags";
import { createChatModelAdapter } from "./custom-chat-adapter";

const useThreadRuntime = (chatApi: string): AssistantRuntime => {
  const adapter = React.useMemo(() => createChatModelAdapter(chatApi), [chatApi]);
  return useLocalRuntime(adapter, { maxSteps: 5 });
};

const useAssistantRuntime = (
  httpBaseUrl: string,
  assistantId: string,
  onAssistantResolved?: (assistantId: string) => void
) => {
  const chatApi = React.useMemo(() => httpBaseUrl, [httpBaseUrl]);
  const adapter = useThreadListAdapter({ httpBaseUrl, assistantId, onAssistantResolved });

  return unstable_useRemoteThreadListRuntime({
    runtimeHook: function RuntimeHook() {
      return useThreadRuntime(chatApi);
    },
    adapter,
  });
};

function AssistantRuntimeMount({
  children,
  httpBaseUrl,
  assistantId,
  onAssistantResolved,
}: PropsWithChildren<{
  httpBaseUrl: string;
  assistantId: string;
  onAssistantResolved?: (assistantId: string) => void;
}>) {
  const runtime = useAssistantRuntime(httpBaseUrl, assistantId, onAssistantResolved);
  const threadMap = useThreadStore((state) => state.threads);
  const knownThreadIDsRef = React.useRef<Set<string>>(new Set());
  const refreshTimerRef = React.useRef<number | null>(null);
  const refreshInFlightRef = React.useRef(false);
  const refreshQueuedRef = React.useRef(false);

  React.useEffect(() => {
    knownThreadIDsRef.current = new Set(Object.keys(threadMap));
  }, [threadMap]);

  const refreshThreadList = React.useCallback(() => {
    const core = (runtime as unknown as { threads?: { _core?: { __internal_load?: () => void } } })
      .threads?._core;
    const loader = core?.__internal_load;
    if (typeof loader === "function") {
      loader.call(core);
    }
  }, [runtime]);

  const flushThreadListRefresh = React.useCallback(() => {
    if (refreshInFlightRef.current) {
      refreshQueuedRef.current = true;
      return;
    }
    refreshInFlightRef.current = true;
    refreshThreadList();
    window.setTimeout(() => {
      refreshInFlightRef.current = false;
      if (!refreshQueuedRef.current) {
        return;
      }
      refreshQueuedRef.current = false;
      flushThreadListRefresh();
    }, 240);
  }, [refreshThreadList]);

  const requestThreadListRefresh = React.useCallback(() => {
    if (!chatFeatureFlags.threadAutoRefreshEnabled) {
      return;
    }
    if (refreshTimerRef.current !== null) {
      return;
    }
    refreshTimerRef.current = window.setTimeout(() => {
      refreshTimerRef.current = null;
      flushThreadListRefresh();
    }, 120);
  }, [flushThreadListRefresh]);

  React.useEffect(() => {
    if (!chatFeatureFlags.threadAutoRefreshEnabled) {
      return;
    }
    const offThreadUpdated = registerTopic(REALTIME_TOPICS.chat.threadUpdated, (event) => {
      if ((event?.type ?? "").trim().toLowerCase() === "resync-required") {
        requestThreadListRefresh();
        return;
      }
      if (event?.replay) {
        requestThreadListRefresh();
        return;
      }
      const payload = event?.payload as
        | { threadId?: string; change?: string; reason?: string }
        | null;
      const threadID = (payload?.threadId ?? "").trim();
      if (!threadID) {
        return;
      }
      const change = (payload?.change ?? "").trim().toLowerCase();
      if (change === "purge") {
        requestThreadListRefresh();
        return;
      }
      if (change !== "upsert") {
        return;
      }
      if (!knownThreadIDsRef.current.has(threadID)) {
        requestThreadListRefresh();
        return;
      }
      requestThreadListRefresh();
    });
    return () => {
      offThreadUpdated();
    };
  }, [requestThreadListRefresh]);

  React.useEffect(() => {
    if (!chatFeatureFlags.threadAutoRefreshEnabled) {
      return;
    }
    const unknownSessionIDs = new Set<string>();
    const unsubscribe = subscribeGatewayEvents((event) => {
      const sessionID = (event.sessionId ?? "").trim();
      if (!sessionID) {
        return;
      }
      if (knownThreadIDsRef.current.has(sessionID) || unknownSessionIDs.has(sessionID)) {
        return;
      }
      unknownSessionIDs.add(sessionID);
      requestThreadListRefresh();
    });
    return () => {
      unsubscribe();
    };
  }, [requestThreadListRefresh]);

  React.useEffect(() => {
    return () => {
      if (refreshTimerRef.current !== null) {
        window.clearTimeout(refreshTimerRef.current);
      }
    };
  }, []);

  return <BaseAssistantRuntimeProvider runtime={runtime}>{children}</BaseAssistantRuntimeProvider>;
}

export function AssistantUIRuntimeProvider({ children }: PropsWithChildren) {
  const [httpBaseUrl, setHttpBaseUrl] = React.useState("");
  const { data: assistants = [] } = useAssistants(false);
  const selectedAssistantId = useChatRuntimeStore((state) => state.assistantId);
  const applyDefaults = useChatRuntimeStore((state) => state.applyDefaults);
  const setAssistantId = useChatRuntimeStore((state) => state.setAssistantId);

  const defaultAssistantId = React.useMemo(() => {
    if (assistants.length === 0) {
      return "";
    }
    return assistants.find((assistant) => assistant.isDefault)?.id ?? assistants[0]?.id ?? "";
  }, [assistants]);

  React.useEffect(() => {
    applyDefaults({
      assistantId: defaultAssistantId,
    });
  }, [assistants, applyDefaults, defaultAssistantId]);

  React.useEffect(() => {
    let active = true;
    Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.HTTPBaseURL")
      .then((url) => {
        if (!active) {
          return;
        }
        const resolved = typeof url === "string" ? url : String(url ?? "");
        setHttpBaseUrl(resolved.trim());
      })
      .catch(() => {
        if (active) {
          setHttpBaseUrl("");
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const runtimeMountKey = httpBaseUrl || "__empty_http_base_url__";
  const assistantId = selectedAssistantId || defaultAssistantId;

  return (
    <AssistantRuntimeMount
      key={runtimeMountKey}
      httpBaseUrl={httpBaseUrl}
      assistantId={assistantId}
      onAssistantResolved={setAssistantId}
    >
      {children}
    </AssistantRuntimeMount>
  );
}
