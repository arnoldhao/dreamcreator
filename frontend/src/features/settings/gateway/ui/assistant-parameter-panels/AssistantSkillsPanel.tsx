import { Input } from "@/shared/ui/input";
import { Separator } from "@/shared/ui/separator";
import { Switch } from "@/shared/ui/switch";
import type { AssistantSkills } from "@/shared/store/assistant";

import { fieldNumberClassName, panelClassName, rowClassName } from "./constants";

type Translate = (key: string) => string;

interface AssistantSkillsPanelProps {
  t: Translate;
  skills: AssistantSkills;
  onEnabledChange: (enabled: boolean) => void;
  onLimitChange: (field: "maxSkillsInPrompt" | "maxPromptChars", value: number) => void;
}

const parseNumberInput = (value: string) => {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }
  return Math.floor(parsed);
};

export function AssistantSkillsPanel({
  t,
  skills,
  onEnabledChange,
  onLimitChange,
}: AssistantSkillsPanelProps) {
  const skillsEnabled = (skills.mode?.trim().toLowerCase() ?? "on") !== "off";
  const maxSkillsInPrompt = skills.maxSkillsInPrompt ?? 150;
  const maxPromptChars = skills.maxPromptChars ?? 30000;

  return (
    <div className={panelClassName}>
      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.skills.injectionEnabled")}
        </div>
        <Switch checked={skillsEnabled} onCheckedChange={onEnabledChange} />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.skills.maxSkillsInPrompt")}
        </div>
        <Input
          type="number"
          min={0}
          value={maxSkillsInPrompt}
          onChange={(event) => onLimitChange("maxSkillsInPrompt", parseNumberInput(event.target.value))}
          size="compact"
          className={fieldNumberClassName}
        />
      </div>

      <Separator />

      <div className={rowClassName}>
        <div className="text-sm font-medium text-muted-foreground">
          {t("settings.gateway.skills.maxPromptChars")}
        </div>
        <Input
          type="number"
          min={0}
          value={maxPromptChars}
          onChange={(event) => onLimitChange("maxPromptChars", parseNumberInput(event.target.value))}
          size="compact"
          className={fieldNumberClassName}
        />
      </div>

    </div>
  );
}
