import { Captions, Clapperboard } from "lucide-react"

import { useI18n } from "@/shared/i18n"
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import type { LibraryWorkspaceEditor } from "../../model/workspaceStore"

type ModeSwitcherProps = {
  value: LibraryWorkspaceEditor
  onChange: (value: LibraryWorkspaceEditor) => void
}

export function ModeSwitcher({ value, onChange }: ModeSwitcherProps) {
  const { t } = useI18n()
  const options: Array<{
    value: LibraryWorkspaceEditor
    label: string
    icon: typeof Clapperboard
  }> = [
    { value: "video", label: t("library.workspace.header.videoEditing"), icon: Clapperboard },
    { value: "subtitle", label: t("library.workspace.header.subtitleEditing"), icon: Captions },
  ]

  return (
    <Tabs
      value={value}
      onValueChange={(nextValue) => onChange(nextValue as LibraryWorkspaceEditor)}
      className="w-auto"
    >
      <TabsList className="w-fit">
        {options.map((option) => {
          const Icon = option.icon
          return (
            <TabsTrigger key={option.value} value={option.value}>
              <Icon className="h-3.5 w-3.5" />
              <span>{option.label}</span>
            </TabsTrigger>
          )
        })}
      </TabsList>
    </Tabs>
  )
}
