import { Box } from "lucide-react";

import { cn } from "@/lib/utils";
import type { Assistant } from "@/shared/store/assistant";

import { buildAssetPreviewUrl } from "./avatarUtils";
import { useHttpBaseUrl } from "./useHttpBaseUrl";
import { Avatar3DViewer } from "./Avatar3DViewer";

interface Assistant3DAvatarProps {
  assistant: Assistant;
  className?: string;
  iconClassName?: string;
  avatarPathOverride?: string | null;
  motionPathOverride?: string | null;
}

export function Assistant3DAvatar({
  assistant,
  className,
  iconClassName,
  avatarPathOverride,
  motionPathOverride,
}: Assistant3DAvatarProps) {
  const httpBaseUrl = useHttpBaseUrl();
  const avatarPath =
    avatarPathOverride !== undefined ? avatarPathOverride?.trim() : assistant.avatar?.avatar3d?.path?.trim();
  const avatarSrc = buildAssetPreviewUrl(httpBaseUrl, avatarPath);
  const motionPath =
    motionPathOverride !== undefined ? motionPathOverride?.trim() : assistant.avatar?.motion?.path?.trim();
  const motionSrc = buildAssetPreviewUrl(httpBaseUrl, motionPath);

  if (!avatarSrc) {
    return (
      <div className={cn("flex items-center justify-center rounded-2xl bg-muted text-muted-foreground", className)}>
        <Box className={cn("h-4 w-4", iconClassName)} />
      </div>
    );
  }

  return (
    <div className={cn("overflow-hidden rounded-2xl", className)}>
      <Avatar3DViewer src={avatarSrc} motionSrc={motionSrc || undefined} />
    </div>
  );
}
