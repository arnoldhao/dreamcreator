<script setup lang="ts">
import type { ToggleGroupRootEmits, ToggleGroupRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { ToggleGroupRoot, useForwardPropsEmits } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<ToggleGroupRootProps & { class?: HTMLAttributes["class"] }>()
const emits = defineEmits<ToggleGroupRootEmits>()

const delegatedProps = reactiveOmit(props, "class")
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ToggleGroupRoot
    v-bind="forwarded"
    :class="cn(
      'inline-flex items-center justify-center gap-1 rounded-lg border border-input bg-muted/40 p-1 text-muted-foreground',
      props.class,
    )"
  >
    <slot />
  </ToggleGroupRoot>
</template>

