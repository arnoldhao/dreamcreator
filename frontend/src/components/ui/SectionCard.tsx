import type { ReactNode } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

export interface SectionCardProps {
  title?: string;
  description?: string;
  children: ReactNode;
  contentClassName?: string;
}

export function SectionCard({ title, description, children, contentClassName }: SectionCardProps) {
  return (
    <Card>
      {title ? (
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          {description ? <CardDescription>{description}</CardDescription> : null}
        </CardHeader>
      ) : null}
      <CardContent className={cn("space-y-3 p-3", contentClassName)}>{children}</CardContent>
    </Card>
  );
}
