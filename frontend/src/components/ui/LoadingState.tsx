import { Loader2 } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";

export interface LoadingStateProps {
  message?: string;
}

export function LoadingState({ message = "Loading..." }: LoadingStateProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-sm">
        <CardContent className="flex items-center gap-3 pt-6">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          <span className="text-sm text-muted-foreground">{message}</span>
        </CardContent>
      </Card>
    </div>
  );
}
