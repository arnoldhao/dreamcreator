<script setup lang="ts">
import type { DropdownMenuItemProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { DropdownMenuItem } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<DropdownMenuItemProps & { class?: HTMLAttributes["class"] }>()
</script>

<template>
  <DropdownMenuItem
    v-bind="props"
    :class="
      cn(
        'relative flex cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors',
        'focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        '[&>svg]:size-4 [&>svg]:shrink-0 [&>svg]:text-muted-foreground',
        props.class,
      )
    "
  >
    <slot />
  </DropdownMenuItem>
</template>
