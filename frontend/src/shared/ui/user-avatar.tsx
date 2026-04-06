import * as React from "react";

import type { CurrentUserProfile } from "@/shared/query/system";
import { cn } from "@/lib/utils";

export interface UserAvatarProps extends React.HTMLAttributes<HTMLDivElement> {
  profile?: CurrentUserProfile | null;
  imageClassName?: string;
  fallbackClassName?: string;
}

export function UserAvatar({ profile, className, imageClassName, fallbackClassName, ...props }: UserAvatarProps) {
  const avatarSrc = resolveUserAvatarSrc(profile);
  const initials = resolveUserInitials(profile);
  const label = resolveUserDisplayName(profile);

  return (
    <div
      className={cn(
        "relative flex shrink-0 items-center justify-center overflow-hidden rounded-xl bg-muted text-muted-foreground dark:bg-muted/80 dark:text-foreground/80",
        className
      )}
      aria-label={label}
      {...props}
    >
      {avatarSrc ? (
        <img
          src={avatarSrc}
          alt={label}
          className={cn("h-full w-full object-cover", imageClassName)}
        />
      ) : (
        <span className={cn("text-sm font-semibold tracking-[0.16em]", fallbackClassName)}>{initials}</span>
      )}
    </div>
  );
}

export function resolveUserAvatarSrc(profile?: CurrentUserProfile | null) {
  const avatarBase64 = profile?.avatarBase64?.trim() ?? "";
  if (!avatarBase64) {
    return "";
  }
  const avatarMime = profile?.avatarMime?.trim() || "image/png";
  return `data:${avatarMime};base64,${avatarBase64}`;
}

export function resolveUserDisplayName(profile?: CurrentUserProfile | null) {
  return profile?.displayName?.trim() || profile?.username?.trim() || "Desktop User";
}

export function resolveUserSubtitle(profile?: CurrentUserProfile | null) {
  const username = profile?.username?.trim() ?? "";
  const displayName = profile?.displayName?.trim() ?? "";
  if (username && displayName && username !== displayName) {
    return username;
  }
  return "";
}

export function resolveUserInitials(profile?: CurrentUserProfile | null) {
  const value = profile?.initials?.trim();
  if (value) {
    return value;
  }
  const source = resolveUserDisplayName(profile);
  const segments = source.split(/\s+/).filter(Boolean);
  if (segments.length > 1) {
    return segments
      .slice(0, 2)
      .map((segment) => segment[0] ?? "")
      .join("")
      .toUpperCase();
  }
  return source.slice(0, 2).toUpperCase();
}
