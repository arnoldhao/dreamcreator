<script setup lang="ts">
import type { ToggleGroupItemProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { ToggleGroupItem } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<ToggleGroupItemProps & { class?: HTMLAttributes["class"] }>()
</script>

<template>
  <ToggleGroupItem
    v-bind="props"
    :class="cn(
      'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-normal ring-offset-background transition-colors',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
      'data-[state=on]:bg-popover data-[state=on]:text-foreground data-[state=on]:shadow-sm',
      props.class,
    )"
  >
    <slot />
  </ToggleGroupItem>
</template>

