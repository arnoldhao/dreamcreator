<script setup lang="ts">
import type { TooltipContentEmits, TooltipContentProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { TooltipContent, TooltipPortal, useForwardPropsEmits } from "reka-ui"
import { cn } from "@/lib/utils"

defineOptions({
  inheritAttrs: false,
})

const props = withDefaults(defineProps<TooltipContentProps & { class?: HTMLAttributes["class"] }>(), {
  sideOffset: 4,
})

const emits = defineEmits<TooltipContentEmits>()

const delegatedProps = reactiveOmit(props, "class")

const forwarded = useForwardPropsEmits(delegatedProps, emits)

const isSettingsWindow = (() => {
  try {
    return new URLSearchParams(window.location.search).get("window") === "settings"
  } catch {
    return false
  }
})()
</script>

<template>
  <TooltipPortal>
    <TooltipContent
      v-bind="{ ...forwarded, ...$attrs }"
      :class="cn(
        isSettingsWindow ? 'dc-shadcn settings-window' : '',
        'z-50 overflow-hidden rounded-md border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
        isSettingsWindow ? 'px-2 py-1 text-xs shadow-lg' : '',
        props.class,
      )"
    >
      <slot />
    </TooltipContent>
  </TooltipPortal>
</template>
