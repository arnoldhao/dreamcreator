import * as React from "react";
import { Sparkles } from "lucide-react";

import { cn } from "@/lib/utils";

export interface ProductModeGlyphProps extends React.HTMLAttributes<HTMLDivElement> {
  iconClassName?: string;
  auraClassName?: string;
}

export function ProductModeGlyph({ className, iconClassName, auraClassName, ...props }: ProductModeGlyphProps) {
  return (
    <div
      className={cn("relative flex items-center justify-center", className)}
      aria-hidden="true"
      {...props}
    >
      <div
        className={cn(
          "absolute inset-[12%] rounded-full bg-[conic-gradient(from_220deg,_rgba(255,115,0,0.9),_rgba(20,184,166,0.9),_rgba(59,130,246,0.92),_rgba(245,158,11,0.88),_rgba(255,115,0,0.9))] blur-md opacity-90",
          auraClassName
        )}
      />
      <div className="relative flex h-full w-full items-center justify-center rounded-[inherit] border border-white/35 bg-[radial-gradient(circle_at_30%_30%,rgba(255,255,255,0.95),rgba(255,255,255,0.6)_36%,rgba(241,245,249,0.14)_100%)] shadow-[inset_0_1px_0_rgba(255,255,255,0.6),0_10px_30px_-16px_rgba(14,165,233,0.8)] dark:border-white/15 dark:bg-[radial-gradient(circle_at_30%_30%,rgba(255,255,255,0.28),rgba(15,23,42,0.88)_55%,rgba(2,6,23,0.95)_100%)]">
        <Sparkles className={cn("h-[46%] w-[46%] text-slate-950 dark:text-white", iconClassName)} />
      </div>
    </div>
  );
}
