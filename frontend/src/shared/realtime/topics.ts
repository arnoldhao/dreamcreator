export const REALTIME_TOPICS = {
  system: {
    hello: "system.hello",
    fonts: "system.fonts",
  },
  debug: {
    echo: "debug.echo",
  },
  chat: {
    threadUpdated: "chat.thread.updated",
  },
  library: {
    operation: "library.operation",
    file: "library.file",
    history: "library.history",
    workspace: "library.workspace",
    workspaceProject: "library.workspace_project",
  },
  notices: {
    created: "notice.created",
    updated: "notice.updated",
    unread: "notice.unread",
  },
} as const

type ExtractValues<T> = T extends string ? T : { [K in keyof T]: ExtractValues<T[K]> }[keyof T]

export type RealtimeTopic = ExtractValues<typeof REALTIME_TOPICS> | (string & {})

export const DEFAULT_DEBUG_TOPICS: RealtimeTopic[] = [REALTIME_TOPICS.system.hello, REALTIME_TOPICS.debug.echo]
