import type { SettingsRouteId } from "@/app/routes/settings";

export type SettingsSectionId = Extract<
  SettingsRouteId,
  | "gateway"
  | "tools"
  | "skills"
  | "general"
  | "provider"
  | "memory"
  | "integration"
  | "connectors"
  | "external-tools"
  | "usage"
  | "debug"
  | "about"
>;

const STORAGE_KEY = "dreamcreator:settings-section";
const GATEWAY_TARGET_KEY = "dreamcreator:settings-gateway-target";

export type GatewayView = "status" | "details" | "character" | "assistant";
export type GatewayPanelTab = "assistant" | "parameters";
export type GatewayCharacterTab = "avatar" | "motion";
export type GatewayParameterTab =
  | "models"
  | "identity"
  | "soul"
  | "user"
  | "tools"
  | "skills"
  | "memory"
  | "workspace";

export type PendingGatewayTarget = {
  view?: GatewayView;
  panelTab?: GatewayPanelTab;
  parameterTab?: GatewayParameterTab;
  characterTab?: GatewayCharacterTab;
};

function toSettingsSection(value: string | null): SettingsSectionId | null {
  if (value === "appearance") {
    return "general";
  }
  if (
    value === "gateway" ||
    value === "tools" ||
    value === "skills" ||
    value === "general" ||
    value === "provider" ||
    value === "memory" ||
    value === "integration" ||
    value === "connectors" ||
    value === "external-tools" ||
    value === "usage" ||
    value === "about" ||
    value === "debug"
  ) {
    return value;
  }
  return null;
}

export function isSettingsSection(value: SettingsRouteId | string | null): value is SettingsSectionId {
  return (
    value === "gateway" ||
    value === "tools" ||
    value === "skills" ||
    value === "general" ||
    value === "provider" ||
    value === "memory" ||
    value === "integration" ||
    value === "connectors" ||
    value === "external-tools" ||
    value === "usage" ||
    value === "about" ||
    value === "debug"
  );
}

export function setPendingSettingsSection(section: SettingsSectionId) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, section);
  } catch {
    // ignore storage errors
  }
}

export function consumePendingSettingsSection(): SettingsSectionId | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    const section = toSettingsSection(stored);
    if (stored) {
      window.localStorage.removeItem(STORAGE_KEY);
    }
    return section;
  } catch {
    return null;
  }
}

export function listenPendingSettingsSection(onSection: (section: SettingsSectionId) => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  const handler = (event: StorageEvent) => {
    if (event.key !== STORAGE_KEY) {
      return;
    }
    const section = toSettingsSection(event.newValue);
    if (section) {
      onSection(section);
      try {
        window.localStorage.removeItem(STORAGE_KEY);
      } catch {
        // ignore storage errors
      }
    }
  };

  window.addEventListener("storage", handler);
  return () => window.removeEventListener("storage", handler);
}

export function setPendingGatewayTarget(target: PendingGatewayTarget) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(GATEWAY_TARGET_KEY, JSON.stringify(target));
  } catch {
    // ignore storage errors
  }
}

export function consumePendingGatewayTarget(): PendingGatewayTarget | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const stored = window.localStorage.getItem(GATEWAY_TARGET_KEY);
    if (stored) {
      window.localStorage.removeItem(GATEWAY_TARGET_KEY);
    }
    if (!stored) {
      return null;
    }
    const parsed = JSON.parse(stored) as PendingGatewayTarget;
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}
