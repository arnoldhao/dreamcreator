<script setup lang="ts">
import type { DialogContentEmits, DialogContentProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { computed } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { X } from "lucide-vue-next"
import {
  DialogClose,
  DialogContent,
  DialogPortal,
  injectDialogRootContext,
  useForwardPropsEmits,
} from "reka-ui"
import { cn } from "@/lib/utils"

interface DialogContentPropsWithClass extends DialogContentProps {
  class?: HTMLAttributes["class"]
}

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<DialogContentPropsWithClass>()
const emits = defineEmits<DialogContentEmits>()

const delegatedProps = reactiveOmit(props, "class")
const forwarded = useForwardPropsEmits(delegatedProps, emits)

const rootContext = injectDialogRootContext()
const isOpen = computed(() => !!rootContext?.open?.value)

const windowMode = (() => {
  try {
    return new URLSearchParams(window.location.search).get("window") || "main"
  } catch {
    return "main"
  }
})()

const useMaiaTokens = windowMode === "settings" || windowMode === "main"
</script>

<template>
  <DialogPortal>
    <template v-if="isOpen">
      <!-- Avoid reka-ui DialogOverlay to prevent body scroll-lock (which can cause layout shift/jitter). -->
      <DialogClose as-child>
        <div class="fixed inset-0 z-50 bg-black/80" />
      </DialogClose>

      <DialogContent
        v-bind="{ ...forwarded, ...$attrs }"
        :class="
          cn(
            // Portals render outside the window root; apply Maia shadcn tokens without inheriting
            // window-level blur/radius rules from `.settings-window`/`.main-window`.
            useMaiaTokens ? 'dc-shadcn dc-shadcn-portal-glass' : 'dc-shadcn',
            'fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-card p-4 shadow-lg',
            'sm:rounded-lg',
            props.class,
          )
        "
      >
        <slot />

        <DialogClose
          class="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none"
        >
          <X class="h-3.5 w-3.5 text-muted-foreground" />
        </DialogClose>
      </DialogContent>
    </template>
  </DialogPortal>
</template>
