import type * as React from "react";

import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";

import {
  ToolIOCard,
  ToolInputExampleBlock,
  ToolMethodSelectorBlock,
  ToolOutputExampleBlock,
} from "./tool-detail-layout";

type ToolMethodIOPanelProps = {
  rowClassName: string;
  labelClassName: string;
  controlClassName: string;
  methodLabel: React.ReactNode;
  actionValue: string;
  actions: string[];
  onActionChange: (value: string) => void;
  emptyMethodLabel: string;
  inputTitle: string;
  outputTitle: string;
  inputPayload: string;
  outputPayload: string;
};

export function ToolMethodIOPanel({
  rowClassName,
  labelClassName,
  controlClassName,
  methodLabel,
  actionValue,
  actions,
  onActionChange,
  emptyMethodLabel,
  inputTitle,
  outputTitle,
  inputPayload,
  outputPayload,
}: ToolMethodIOPanelProps) {
  return (
    <ToolIOCard>
      <ToolMethodSelectorBlock
        rowClassName={rowClassName}
        label={methodLabel}
        control={
          <Select
            value={actionValue}
            onChange={(event) => onActionChange(event.target.value)}
            className={controlClassName}
            disabled={actions.length === 0}
          >
            {actions.length === 0 ? (
              <option value="">{emptyMethodLabel}</option>
            ) : (
              actions.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))
            )}
          </Select>
        }
      />
      <Separator />
      <ToolInputExampleBlock
        title={inputTitle}
        labelClassName={labelClassName}
        payload={inputPayload}
      />
      <Separator />
      <ToolOutputExampleBlock
        title={outputTitle}
        labelClassName={labelClassName}
        payload={outputPayload}
      />
    </ToolIOCard>
  );
}
