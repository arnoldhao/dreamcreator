import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import type { Assistant, AssistantIdentity } from "@/shared/store/assistant";

import { AssistantEmojiPicker } from "../AssistantEmojiPicker";
import { fieldClassName, fieldSelectClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

interface AssistantIdentityPanelProps {
  t: Translate;
  assistant: Assistant;
  identity: AssistantIdentity;
  roleOptions: string[];
  defaultRole: string;
  onUpdateIdentity: (next: AssistantIdentity, commit?: boolean) => void;
}

export function AssistantIdentityPanel({
  t,
  assistant,
  identity,
  roleOptions,
  defaultRole,
  onUpdateIdentity,
}: AssistantIdentityPanelProps) {
  return (
    <div className={panelClassName}>
      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.identity.name")}
        </div>
        <Input
          value={identity.name ?? ""}
          onChange={(event) => onUpdateIdentity({ ...identity, name: event.target.value })}
          onBlur={() => onUpdateIdentity(identity, true)}
          size="compact"
          className={fieldClassName}
        />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.identity.emoji")}
        </div>
        <AssistantEmojiPicker assistant={assistant} emojiClassName="text-base" />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.identity.role")}
        </div>
        {roleOptions.length > 0 ? (
          <Select
            value={identity.role?.trim() || defaultRole || roleOptions[0]}
            onChange={(event) => onUpdateIdentity({ ...identity, role: event.target.value }, true)}
            className={fieldSelectClassName}
          >
            {roleOptions.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </Select>
        ) : (
          <Input
            value={identity.role ?? ""}
            onChange={(event) => onUpdateIdentity({ ...identity, role: event.target.value })}
            onBlur={() => onUpdateIdentity(identity, true)}
            size="compact"
            className={fieldClassName}
          />
        )}
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.identity.creature")}
        </div>
        <Input
          value={identity.creature ?? ""}
          onChange={(event) => onUpdateIdentity({ ...identity, creature: event.target.value })}
          onBlur={() => onUpdateIdentity(identity, true)}
          size="compact"
          className={fieldClassName}
        />
      </div>
    </div>
  );
}
