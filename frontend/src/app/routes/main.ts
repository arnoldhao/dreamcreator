import type { SettingsRouteId } from "./settings";

export type MainPageId = "library" | "cron";
export type MainRouteId = MainPageId | "chat";

export interface MainRoute {
  id: MainRouteId;
  label: string;
  path: string;
  kind: "page" | "chat";
  hotkey?: string;
}

export interface SidebarFooterMenuItem {
  id: SettingsRouteId;
  label: string;
  path: string;
}

export interface SidebarFooterMenu {
  label: string;
  items: SidebarFooterMenuItem[];
}

export interface SidebarFooterConfig {
  menu: SidebarFooterMenu;
  updateAction: {
    id: "update";
    label: string;
  };
}

export const MAIN_NAV_ROUTES: MainRoute[] = [
  { id: "chat", label: "Chat", path: "/chat", kind: "chat", hotkey: "g c" },
  { id: "library", label: "Library", path: "/library", kind: "page", hotkey: "g l" },
  { id: "cron", label: "Cron", path: "/cron", kind: "page", hotkey: "g g" },
];

export const MAIN_CHAT_ROUTE: MainRoute = {
  id: "chat",
  label: "Chat",
  path: "/chat",
  kind: "chat",
  hotkey: "g c",
};

export const MAIN_SIDEBAR_FOOTER: SidebarFooterConfig = {
  menu: {
    label: "Settings",
    items: [
      { id: "general", label: "Settings", path: "/settings/general" },
    ],
  },
  updateAction: {
    id: "update",
    label: "Update",
  },
};
