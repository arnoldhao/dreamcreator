export type SettingsRouteId =
  | "gateway"
  | "tools"
  | "skills"
  | "general"
  | "memory"
  | "integration"
  | "connectors"
  | "external-tools"
  | "usage"
  | "debug"
  | "about"
  | "provider"
  | "link";

export interface SettingsRoute {
  id: SettingsRouteId;
  label: string;
  path: string;
}

export const SETTINGS_ROUTES: SettingsRoute[] = [
  { id: "gateway", label: "Gateway", path: "/settings/gateway" },
  { id: "general", label: "General", path: "/settings/general" },
  { id: "provider", label: "Provider", path: "/settings/provider" },
  { id: "tools", label: "Tools", path: "/settings/tools" },
  { id: "skills", label: "Skills", path: "/settings/skills" },
  { id: "memory", label: "Memory", path: "/settings/memory" },
  { id: "integration", label: "Integration", path: "/settings/integration" },
  { id: "connectors", label: "Connectors", path: "/settings/connectors" },
  { id: "external-tools", label: "External Tools", path: "/settings/external-tools" },
  { id: "usage", label: "Usage", path: "/settings/usage" },
  { id: "debug", label: "Debug", path: "/settings/debug" },
  { id: "link", label: "Link", path: "/settings/link" },
  { id: "about", label: "About", path: "/settings/about" },
];
