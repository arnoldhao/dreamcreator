<template>
  <aside class="dc-main-ribbon" :style="asideStyle">
    <!-- Main source list -->
    <div class="pl-2 pr-2 pt-1 source-group">
      <div
        v-for="(m, i) in navStore.menuOptions"
        :key="i"
        class="source-chip"
        :class="{ 'ribbon-active active': props.value === m.key }"
        @click="emit('update:value', m.key)"
      >
        <span class="icon-cell"><Icon :name="m.icon" class="source-row-icon" /></span>
        <span class="label-cell"><span class="source-row-label truncate">{{ $t(m.label) }}</span></span>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import useNavStore from 'stores/nav.js'

const navStore = useNavStore()

const props = defineProps({
  value: {
    type: String,
    default: 'download',
  },
  width: {
    type: Number,
    default: 260,
  },
})

const emit = defineEmits(['update:value'])

const widthPx = computed(() => `${props.width}px`)
const asideStyle = computed(() => ({
  width: widthPx.value,
  minWidth: widthPx.value,
}))

</script>

<style lang="scss" scoped>
.dc-main-ribbon { height: 100%; background: transparent; }

.ribbon-header {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 6px 0 6px;
}

/* Light + frosted tweaks for items now live in global styles */

/* Popover 样式已全局化：.macos-popover/.popover-item 等在 styles/macos-components.scss */
</style>
