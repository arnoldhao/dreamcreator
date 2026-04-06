import * as React from "react";
import { Call } from "@wailsio/runtime";
import {
  type AttachmentAdapter,
  type CompleteAttachment,
  type PendingAttachment,
  type ThreadHistoryAdapter,
  type ThreadMessage,
  RuntimeAdapterProvider,
  useAssistantApi,
  type unstable_RemoteThreadListAdapter as RemoteThreadListAdapter,
} from "@assistant-ui/react";
import { createAssistantStream, type AssistantStream } from "assistant-stream";

import {
  useThreadStore,
  type ThreadSummary,
  type ThreadStatus,
  type ThreadTitleChangedBy,
} from "@/shared/store/threads";
import { toUIMessage, type UIMessagePart } from "./custom-chat-adapter";
import {
  clearThreadStreamCursor,
  registerThreadAlias,
  removeThreadAlias,
  resolveRemoteThreadId,
} from "./thread-identities";

const DEFAULT_THREAD_TITLE = "New Chat";
const DEFAULT_ATTACHMENT_MEDIA_TYPE = "application/octet-stream";

type ThreadDTO = {
  id?: string;
  workspaceName?: string;
  assistantId?: string;
  title?: string;
  titleIsDefault?: boolean;
  titleChangedBy?: string;
  status?: ThreadStatus;
  createdAt?: string;
  updatedAt?: string;
  lastInteractiveAt?: string;
  deletedAt?: string;
  purgeAfter?: string;
};

type RemoteThreadMetadata = {
  status: "regular" | "archived";
  remoteId: string;
  title?: string;
};

type RemoteThreadListResponse = {
  threads: RemoteThreadMetadata[];
};

type NewThreadResponseDTO = {
  threadId?: string;
  workspaceName?: string;
  assistantId?: string;
};

type ThreadMessageDTO = {
  id?: string;
  kind?: string;
  role?: string;
  content?: string;
  partsJson?: string;
  partsVersion?: number;
  createdAt?: string;
};

type GenerateThreadTitleRequestDTO = {
  threadId: string;
  fallbackTitle?: string;
};

type GenerateThreadTitleResponseDTO = {
  threadId?: string;
  title?: string;
  titleIsDefault?: boolean;
  titleChangedBy?: string;
};

const normalizeTitleChangedBy = (value: unknown): ThreadTitleChangedBy | undefined => {
  if (typeof value !== "string") {
    return undefined;
  }
  const normalized = value.trim().toLowerCase();
  if (normalized === "user" || normalized === "summary") {
    return normalized;
  }
  return undefined;
};

const normalizeAttachmentType = (file: File): "image" | "document" | "file" => {
  const contentType = file.type.toLowerCase();
  if (contentType.startsWith("image/")) {
    return "image";
  }
  if (
    contentType.startsWith("text/") ||
    contentType.includes("json") ||
    contentType.includes("xml") ||
    contentType.includes("pdf")
  ) {
    return "document";
  }
  return "file";
};

const createAttachmentID = () => {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const toFileDataURL = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = (error) => reject(error);
    reader.readAsDataURL(file);
  });

const useAttachmentAdapter = (): AttachmentAdapter =>
  React.useMemo<AttachmentAdapter>(() => {
    return {
      accept: "*",
      async add({ file }): Promise<PendingAttachment> {
        const contentType = file.type || DEFAULT_ATTACHMENT_MEDIA_TYPE;
        return {
          id: createAttachmentID(),
          type: normalizeAttachmentType(file),
          name: file.name || "attachment",
          contentType,
          file,
          status: { type: "requires-action", reason: "composer-send" },
        };
      },
      async remove() {
        return;
      },
      async send(attachment): Promise<CompleteAttachment> {
        const file = attachment.file;
        const name = attachment.name || file?.name || "attachment";
        const contentType = attachment.contentType || file?.type || DEFAULT_ATTACHMENT_MEDIA_TYPE;
        if (attachment.type === "image" && file) {
          return {
            ...attachment,
            name,
            contentType,
            status: { type: "complete" },
            content: [{ type: "image", image: await toFileDataURL(file), filename: name }],
          };
        }
        const dataURL = file ? await toFileDataURL(file) : "";
        const data = dataURL.includes(",") ? dataURL.split(",")[1] ?? "" : dataURL;
        return {
          ...attachment,
          name,
          contentType,
          status: { type: "complete" },
          content: [{ type: "file", filename: name, data, mimeType: contentType }],
        };
      },
    };
  }, []);

const normalizeThreadDTO = (item: ThreadDTO): ThreadSummary => {
  const id = typeof item.id === "string" ? item.id : "";
  return {
    id,
    workspaceName: typeof item.workspaceName === "string" ? item.workspaceName : "",
    assistantId: typeof item.assistantId === "string" ? item.assistantId : "",
    title: typeof item.title === "string" && item.title.trim() ? item.title : DEFAULT_THREAD_TITLE,
    titleIsDefault: item.titleIsDefault !== false,
    titleChangedBy: normalizeTitleChangedBy(item.titleChangedBy),
    status: item.status === "archived" ? "archived" : "regular",
    createdAt: typeof item.createdAt === "string" ? item.createdAt : "",
    updatedAt: typeof item.updatedAt === "string" ? item.updatedAt : "",
    lastInteractiveAt:
      typeof item.lastInteractiveAt === "string"
        ? item.lastInteractiveAt
        : typeof item.updatedAt === "string"
          ? item.updatedAt
          : "",
    deletedAt: typeof item.deletedAt === "string" ? item.deletedAt : "",
    purgeAfter: typeof item.purgeAfter === "string" ? item.purgeAfter : "",
  };
};

type StoredMessagePart = {
  type?: string;
  parentId?: string;
  text?: string;
  state?: string;
  toolCallId?: string;
  toolName?: string;
  input?: unknown;
  output?: unknown;
  errorText?: string;
  data?: unknown;
};

const normalizePartParentId = (part: StoredMessagePart) => {
  const parentId = typeof part.parentId === "string" ? part.parentId.trim() : "";
  return parentId || undefined;
};

const parseAttachmentPayload = (raw: unknown) => {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return {
      filename: "",
      mimeType: "",
      data: "",
      image: "",
      path: "",
    };
  }
  const payload = raw as Record<string, unknown>;
  return {
    filename: typeof payload.filename === "string" ? payload.filename.trim() : "",
    mimeType: typeof payload.mimeType === "string" ? payload.mimeType.trim() : "",
    data: typeof payload.data === "string" ? payload.data : "",
    image: typeof payload.image === "string" ? payload.image : "",
    path: typeof payload.path === "string" ? payload.path.trim() : "",
  };
};

const parseDataUrlMime = (value: string) => {
  if (!value.startsWith("data:")) {
    return "";
  }
  const comma = value.indexOf(",");
  const head = comma >= 0 ? value.slice(5, comma) : value.slice(5);
  const semicolon = head.indexOf(";");
  const mime = semicolon >= 0 ? head.slice(0, semicolon) : head;
  return mime.trim();
};

const normalizeAttachmentTypeFromMime = (mimeType: string): "image" | "document" | "file" => {
  const normalized = mimeType.toLowerCase();
  if (normalized.startsWith("image/")) {
    return "image";
  }
  if (
    normalized.startsWith("text/") ||
    normalized.includes("json") ||
    normalized.includes("xml") ||
    normalized.includes("pdf")
  ) {
    return "document";
  }
  return "file";
};

const buildStoredAttachmentId = (messageId: string, index: number) =>
  `${messageId}-attachment-${index}`;

const extractUserMessageContent = (parts: StoredMessagePart[], messageId: string) => {
  const content: Array<any> = [];
  const attachments: CompleteAttachment[] = [];
  let attachmentIndex = 0;
  let hasText = false;

  for (const part of parts) {
    const partType = typeof part.type === "string" ? part.type : "";
    if (!partType) {
      continue;
    }
    if (partType === "text") {
      const text = typeof part.text === "string" ? part.text : "";
      content.push({ type: "text", text });
      if (text.trim()) {
        hasText = true;
      }
      continue;
    }
    if (partType === "image") {
      const payload = parseAttachmentPayload(part.data);
      const image = payload.image || payload.path;
      if (!image) {
        continue;
      }
      const filename = payload.filename || "image";
      const mimeType = payload.mimeType || parseDataUrlMime(image) || "image/png";
      attachments.push({
        id: buildStoredAttachmentId(messageId, attachmentIndex),
        type: "image",
        name: filename,
        contentType: mimeType,
        status: { type: "complete" },
        content: [
          {
            type: "image",
            image,
            ...(filename ? { filename } : {}),
          },
        ],
      });
      attachmentIndex += 1;
      continue;
    }
    if (partType === "file") {
      const payload = parseAttachmentPayload(part.data);
      const filename = payload.filename || "attachment";
      const mimeType = payload.mimeType || DEFAULT_ATTACHMENT_MEDIA_TYPE;
      attachments.push({
        id: buildStoredAttachmentId(messageId, attachmentIndex),
        type: normalizeAttachmentTypeFromMime(mimeType),
        name: filename,
        contentType: mimeType,
        status: { type: "complete" },
        content: [
          {
            type: "file",
            data: payload.data || "",
            mimeType,
            ...(filename ? { filename } : {}),
          },
        ],
      });
      attachmentIndex += 1;
      continue;
    }
  }

  return { content, attachments, hasText };
};

const parsePartsJson = (raw: string | undefined) => {
  if (!raw || !raw.trim()) {
    return [] as StoredMessagePart[];
  }
  try {
    const decoded = JSON.parse(raw);
    if (!Array.isArray(decoded)) {
      return [] as StoredMessagePart[];
    }
    return decoded as StoredMessagePart[];
  } catch {
    return [] as StoredMessagePart[];
  }
};

const parsePartDataRecord = (value: unknown): Record<string, unknown> | null => {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  if (typeof value !== "string") {
    return null;
  }
  const trimmed = value.trim();
  if (!(trimmed.startsWith("{") || trimmed.startsWith("["))) {
    return null;
  }
  try {
    const decoded = JSON.parse(trimmed);
    if (!decoded || typeof decoded !== "object" || Array.isArray(decoded)) {
      return null;
    }
    return decoded as Record<string, unknown>;
  } catch {
    return null;
  }
};

const extractStoredRuntimeErrorText = (parts: StoredMessagePart[]) => {
  for (const part of parts) {
    const partType = typeof part.type === "string" ? part.type : "";
    if (partType !== "data" || !part.data) {
      continue;
    }
    const payload = parsePartDataRecord(part.data);
    if (!payload) {
      continue;
    }
    const name = typeof payload.name === "string" ? payload.name.trim().toLowerCase() : "";
    if (name !== "runtime_error") {
      continue;
    }
    const data = payload.data;
    if (data && typeof data === "object" && !Array.isArray(data)) {
      const detail = data as Record<string, unknown>;
      const message = typeof detail.message === "string" ? detail.message.trim() : "";
      if (message) {
        return message;
      }
      const fallbackDetail = typeof detail.detail === "string" ? detail.detail.trim() : "";
      if (fallbackDetail) {
        return fallbackDetail;
      }
    }
    const text = typeof part.text === "string" ? part.text.trim() : "";
    if (text) {
      return text;
    }
    return "llm request failed";
  }
  return "";
};

const normalizeStoredSourcePart = (part: StoredMessagePart) => {
  const payload =
    part.data && typeof part.data === "object" && !Array.isArray(part.data)
      ? (part.data as Record<string, unknown>)
      : null;
  const urlFromData = payload && typeof payload.url === "string" ? payload.url.trim() : "";
  const urlFromText = typeof part.text === "string" ? part.text.trim() : "";
  const url = urlFromData || urlFromText;
  if (!url) {
    return null;
  }
  const idFromData = payload && typeof payload.id === "string" ? payload.id.trim() : "";
  const title = payload && typeof payload.title === "string" ? payload.title.trim() : "";
  return {
    type: "source",
    sourceType: "url" as const,
    id: idFromData || url,
    url,
    ...(normalizePartParentId(part) ? { parentId: normalizePartParentId(part) } : {}),
    ...(title ? { title } : {}),
  };
};

const storedPartsToThreadContent = (parts: StoredMessagePart[]) => {
  const content: Array<any> = [];
  for (const part of parts) {
    const partType = typeof part.type === "string" ? part.type : "";
    if (!partType) {
      continue;
    }
    if (partType === "text") {
      content.push({
        type: "text",
        text: typeof part.text === "string" ? part.text : "",
        ...(normalizePartParentId(part) ? { parentId: normalizePartParentId(part) } : {}),
      });
      continue;
    }
    if (partType === "reasoning") {
      content.push({
        type: "reasoning",
        text: typeof part.text === "string" ? part.text : "",
        ...(normalizePartParentId(part) ? { parentId: normalizePartParentId(part) } : {}),
      });
      continue;
    }
    if (partType === "source") {
      const sourcePart = normalizeStoredSourcePart(part);
      if (sourcePart) {
        content.push(sourcePart);
      }
      continue;
    }
    if (partType === "tool-call") {
      const toolName = typeof part.toolName === "string" ? part.toolName.trim() : "";
      if (!toolName) {
        continue;
      }
      const args = part.input ?? {};
      const argsText = typeof args === "string" ? args : JSON.stringify(args ?? {});
      const state = typeof part.state === "string" ? part.state : "";
      const isError = state === "output-error" || state === "input-error";
      content.push({
        type: "tool-call",
        toolCallId: typeof part.toolCallId === "string" ? part.toolCallId : "",
        toolName,
        args,
        argsText,
        result: part.output ?? (isError ? { error: part.errorText ?? "tool error" } : undefined),
        isError,
        ...(normalizePartParentId(part) ? { parentId: normalizePartParentId(part) } : {}),
      });
      continue;
    }
    if (partType === "data") {
      const payload = parsePartDataRecord(part.data);
      const dataName = payload && typeof payload.name === "string" ? payload.name.trim() : "";
      if (!dataName) {
        continue;
      }
      const parentId = normalizePartParentId(part);
      content.push({
        type: "data",
        name: dataName,
        data: payload ? payload.data ?? null : null,
        ...(parentId ? { parentId } : {}),
      });
      continue;
    }
    if (partType === "image") {
      const payload = parseAttachmentPayload(part.data);
      const image = payload.image || payload.path;
      if (!image) {
        continue;
      }
      const filename = payload.filename || undefined;
      content.push({
        type: "image",
        image,
        ...(filename ? { filename } : {}),
      });
      continue;
    }
    if (partType === "file") {
      const payload = parseAttachmentPayload(part.data);
      const data = payload.data;
      const mimeType = payload.mimeType || DEFAULT_ATTACHMENT_MEDIA_TYPE;
      if (!data && !payload.path) {
        continue;
      }
      const filename = payload.filename || undefined;
      content.push({
        type: "file",
        data: data || "",
        mimeType,
        ...(filename ? { filename } : {}),
      });
      continue;
    }
  }
  return content;
};

const joinUIMessageText = (parts: UIMessagePart[]) => {
  let text = "";
  for (const part of parts) {
    if (part.type === "text" && typeof part.text === "string" && part.text) {
      text += part.text;
    }
  }
  return text.trim();
};

const toThreadMessage = (item: ThreadMessageDTO): ThreadMessage => {
  const createdAt = item.createdAt ? new Date(item.createdAt) : new Date();
  const id = typeof item.id === "string" && item.id ? item.id : `${createdAt.getTime()}`;
  const contentText = typeof item.content === "string" ? item.content : "";
  const storedParts = parsePartsJson(item.partsJson);
  const role = typeof item.role === "string" ? item.role : "user";
  const kind = typeof item.kind === "string" ? item.kind.trim().toLowerCase() : "chat";

  if (kind === "notice") {
    const parts = storedPartsToThreadContent(storedParts);
    const noticeText =
      (parts.find((part) => part?.type === "text" && typeof part.text === "string") as { text?: string } | undefined)
        ?.text ?? contentText;
    return {
      id,
      role: "system",
      createdAt,
      content: [{ type: "text", text: noticeText }],
      metadata: { custom: { kind: "notice" } },
    };
  }

  if (role === "assistant") {
    const parts = storedPartsToThreadContent(storedParts);
    const assistantContent = parts.length > 0 ? parts : [{ type: "text", text: contentText }];
    const runtimeErrorText = extractStoredRuntimeErrorText(storedParts);
    return {
      id,
      role: "assistant",
      createdAt,
      content: assistantContent,
      status: runtimeErrorText
        ? { type: "incomplete", reason: "error", error: runtimeErrorText }
        : { type: "complete", reason: "unknown" },
      metadata: {
        unstable_state: null,
        unstable_annotations: [],
        unstable_data: [],
        steps: [],
        custom: {},
      },
    };
  }

  if (role === "system") {
    const parts = storedPartsToThreadContent(storedParts);
    const firstTextPart = parts.find(
      (part) => part && part.type === "text" && typeof part.text === "string"
    ) as { text?: string } | undefined;
    const systemText = firstTextPart?.text ?? contentText;
    return {
      id,
      role: "system",
      createdAt,
      content: [{ type: "text", text: systemText }],
      metadata: { custom: {} },
    };
  }

  const { content: userContent, attachments, hasText } = extractUserMessageContent(storedParts, id);
  const trimmedContent = contentText.trim();
  if (!hasText && trimmedContent) {
    userContent.push({ type: "text", text: trimmedContent });
  }

  return {
    id,
    role: "user",
    createdAt,
    attachments,
    content: userContent,
    metadata: { custom: {} },
  };
};

const buildExportedRepository = (messages: ThreadMessage[]) => {
  const items = messages.map((message, index) => ({
    parentId: index > 0 ? messages[index - 1]?.id ?? null : null,
    message,
  }));
  return {
    headId: messages.length > 0 ? messages[messages.length - 1]?.id ?? null : null,
    messages: items,
  };
};

const useThreadHistoryAdapter = (httpBaseUrl: string): ThreadHistoryAdapter => {
  const api = useAssistantApi();

  return React.useMemo<ThreadHistoryAdapter>(() => {
    return {
      load: async () => {
        const { remoteId } = api.threadListItem().getState();
        if (!remoteId) {
          return { headId: null, messages: [] };
        }
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.ListMessages",
          remoteId,
          200
        );
        const raw = (result as ThreadMessageDTO[]) ?? [];
        const messages = raw.map(toThreadMessage);
        return {
          ...buildExportedRepository(messages),
          unstable_resume: false,
        };
      },
      resume: async function* (options) {
        void options;
        return;
      },
      append: async ({ message }) => {
        if (message.role !== "user") {
          return;
        }
        const { remoteId, id } = api.threadListItem().getState();
        const threadId = remoteId || id;
        if (!threadId) {
          return;
        }
        const uiMessage = toUIMessage(message);
        if (!uiMessage) {
          return;
        }
        const content = joinUIMessageText(uiMessage.parts);
        try {
          await Call.ByName("dreamcreator/internal/presentation/wails.ThreadHandler.AppendMessage", {
            id: message.id,
            threadId,
            role: uiMessage.role,
            content,
            parts: uiMessage.parts,
          });
        } catch {
          // Best effort: message persistence should not block UI.
        }
      },
    };
  }, [api, httpBaseUrl]);
};

export type ThreadListAdapterOptions = {
  httpBaseUrl: string;
  assistantId: string;
  onAssistantResolved?: (assistantId: string) => void;
};

export const useThreadListAdapter = ({
  httpBaseUrl,
  assistantId,
  onAssistantResolved,
}: ThreadListAdapterOptions): RemoteThreadListAdapter => {
  const setThreads = useThreadStore((state) => state.setThreads);
  const upsertThread = useThreadStore((state) => state.upsertThread);
  const patchThread = useThreadStore((state) => state.patchThread);
  const removeThread = useThreadStore((state) => state.removeThread);

  const unstable_Provider = React.useCallback<React.FC<React.PropsWithChildren>>(
    function Provider({ children }) {
      const history = useThreadHistoryAdapter(httpBaseUrl);
      const attachments = useAttachmentAdapter();
      const adapters = React.useMemo(() => ({ history, attachments }), [attachments, history]);
      return <RuntimeAdapterProvider adapters={adapters}>{children}</RuntimeAdapterProvider>;
    },
    [httpBaseUrl]
  );

  return React.useMemo<RemoteThreadListAdapter>(() => {
    return {
      list: async (): Promise<RemoteThreadListResponse> => {
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.ListThreads",
          false
        );
        const items = (result as ThreadDTO[]) ?? [];
        const threads = (items ?? []).map(normalizeThreadDTO).filter((thread) => thread.id);
        setThreads(threads);
        return {
          threads: threads.map((thread) => ({
            status: thread.status,
            remoteId: thread.id,
            title: thread.title,
          })),
        };
      },
      initialize: async (
        localId: string
      ): Promise<{ remoteId: string; externalId: string | undefined }> => {
        if (!assistantId) {
          throw new Error("assistant id is required");
        }
        const payload = (await Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.NewThread",
          {
            assistantId,
            title: "",
            isDefaultTitle: true,
          }
        )) as NewThreadResponseDTO;
        const payloadThreadId = payload.threadId ?? "";
        if (!payloadThreadId) {
          throw new Error("thread id is missing");
        }
        const resolvedAssistantId = (payload.assistantId ?? assistantId).trim();
        if (resolvedAssistantId) {
          onAssistantResolved?.(resolvedAssistantId);
        }
        registerThreadAlias(localId, payloadThreadId);
        const thread: ThreadSummary = {
          id: payloadThreadId,
          workspaceName: payload.workspaceName ?? "",
          assistantId: resolvedAssistantId,
          title: DEFAULT_THREAD_TITLE,
          titleIsDefault: true,
          titleChangedBy: undefined,
          status: "regular",
          createdAt: "",
          updatedAt: "",
          lastInteractiveAt: "",
          deletedAt: "",
          purgeAfter: "",
        };
        upsertThread(thread);
        return { remoteId: payloadThreadId, externalId: undefined };
      },
      rename: async (remoteId, newTitle) => {
        await Call.ByName("dreamcreator/internal/presentation/wails.ThreadHandler.RenameThread", {
          threadId: remoteId,
          title: newTitle,
        });
        patchThread(remoteId, { title: newTitle, titleIsDefault: false, titleChangedBy: "user" });
      },
      archive: async (remoteId) => {
        await Call.ByName("dreamcreator/internal/presentation/wails.ThreadHandler.SetThreadStatus", {
          threadId: remoteId,
          status: "archived",
        });
        patchThread(remoteId, { status: "archived" });
      },
      unarchive: async (remoteId) => {
        await Call.ByName("dreamcreator/internal/presentation/wails.ThreadHandler.SetThreadStatus", {
          threadId: remoteId,
          status: "regular",
        });
        patchThread(remoteId, { status: "regular" });
      },
      delete: async (remoteId) => {
        await Call.ByName("dreamcreator/internal/presentation/wails.ThreadHandler.SoftDeleteThread", remoteId);
        removeThreadAlias(remoteId);
        clearThreadStreamCursor(remoteId);
        removeThread(remoteId);
      },
      fetch: async (threadId: string): Promise<RemoteThreadMetadata> => {
        const remoteId = resolveRemoteThreadId(threadId);
        if (!remoteId) {
          throw new Error("thread id is required");
        }
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.ThreadHandler.ListThreads",
          true
        );
        const items = (result as ThreadDTO[]) ?? [];
        const item = items.find((thread) => thread.id === remoteId) ?? {};
        const thread = normalizeThreadDTO(item ?? {});
        if (!thread.id) {
          throw new Error("thread id is missing");
        }
        if (thread.id) {
          upsertThread(thread);
        }
        return {
          status: thread.status,
          remoteId: thread.id,
          title: thread.title,
        };
      },
      generateTitle: async (remoteId: string, _messages?: readonly ThreadMessage[]): Promise<AssistantStream> => {
        return createAssistantStream(async (controller) => {
          const resolvedRemoteId = resolveRemoteThreadId(remoteId);
          if (!resolvedRemoteId) {
            controller.close();
            return;
          }

          try {
            const response = (await Call.ByName(
              "dreamcreator/internal/presentation/wails.ThreadHandler.GenerateThreadTitle",
              {
                threadId: resolvedRemoteId,
              } as GenerateThreadTitleRequestDTO
            )) as GenerateThreadTitleResponseDTO;
            const title = typeof response.title === "string" ? response.title.trim() : "";
            const patch: Partial<ThreadSummary> = {};
            if (title) {
              patch.title = title;
              controller.appendText(title);
            }
            if (typeof response.titleIsDefault === "boolean") {
              patch.titleIsDefault = response.titleIsDefault;
            }
            const titleChangedBy = normalizeTitleChangedBy(response.titleChangedBy);
            if (titleChangedBy) {
              patch.titleChangedBy = titleChangedBy;
            }
            if (Object.keys(patch).length > 0) {
              patchThread(resolvedRemoteId, patch);
            }
          } catch {}
          controller.close();
        });
      },
      unstable_Provider,
    };
  }, [assistantId, httpBaseUrl, onAssistantResolved, patchThread, removeThread, setThreads, upsertThread, unstable_Provider]);
};

export const resolveThreadMeta = (threadId: string) => {
  if (!threadId) {
    return null;
  }
  const remoteId = resolveRemoteThreadId(threadId);
  return useThreadStore.getState().threads[remoteId] ?? null;
};
