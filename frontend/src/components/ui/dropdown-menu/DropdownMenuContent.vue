<script setup lang="ts">
import type { DropdownMenuContentEmits, DropdownMenuContentProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import {
  DropdownMenuContent,
  DropdownMenuPortal,
  useForwardPropsEmits,
} from "reka-ui"
import { cn } from "@/lib/utils"

const props = withDefaults(defineProps<DropdownMenuContentProps & { class?: HTMLAttributes["class"] }>(), {
  sideOffset: 6,
})
const emits = defineEmits<DropdownMenuContentEmits>()

const delegatedProps = reactiveOmit(props, "class")
const forwarded = useForwardPropsEmits(delegatedProps, emits)

const windowMode = (() => {
  try {
    return new URLSearchParams(window.location.search).get("window") || "main"
  } catch {
    return "main"
  }
})()

// Only apply shadcn tokens for portals (avoid inheriting window-level blur/radius rules).
const shadcnPortalClass = (windowMode === "settings" || windowMode === "main")
  ? "dc-shadcn dc-shadcn-portal-glass"
  : "dc-shadcn"
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuContent
      v-bind="{ ...forwarded, ...$attrs }"
      :style="{ maxHeight: 'min(var(--reka-dropdown-menu-content-available-height), 50vh)' }"
      :class="
        cn(
          shadcnPortalClass,
          'z-50 min-w-[10rem] overflow-y-auto overflow-x-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md',
          props.class,
        )
      "
    >
      <slot />
    </DropdownMenuContent>
  </DropdownMenuPortal>
</template>
