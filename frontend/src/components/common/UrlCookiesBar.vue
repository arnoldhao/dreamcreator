<template>
  <div class="url-row" :class="variant" @keydown.stop>
    <div class="search-field">
      <Icon name="search" class="icon" />
      <input
        type="text"
        :placeholder="placeholder"
        class="input-macos"
        :value="url"
        @input="$emit('update:url', $event.target.value)"
        ref="inputRef"
      />
      <!-- paste button removed per UX: use keyboard shortcut instead -->
    </div>
    <transition name="slide-in-right">
      <div v-if="trailingVisible" class="trailing-controls">
        <select v-if="showSelect" class="select-macos" :disabled="isLoadingBrowsers" :value="browser" @change="$emit('update:browser', $event.target.value)">
          <option v-for="opt in browserOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button v-if="showParse" class="btn-glass btn-primary" :disabled="parseDisabled" @click="$emit('parse')">
          <slot name="parse-icon">
            <Icon v-if="parsing" name="spinner" class="animate-spin w-4 h-4 mr-2" />
          </slot>
          <span>{{ parsing ? parsingText : parseText }}</span>
        </button>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, defineProps } from 'vue'

const props = defineProps({
  url: String,
  browser: String,
  browserOptions: { type: Array, default: () => [] },
  isLoadingBrowsers: Boolean,
  trailingVisible: Boolean,
  showParse: Boolean,
  parseDisabled: Boolean,
  parsing: Boolean,
  showSelect: { type: Boolean, default: true },
  placeholder: { type: String, default: '' },
  parseText: { type: String, default: 'Parse' },
  parsingText: { type: String, default: 'Parsing…' },
  variant: { type: String, default: 'compact' } // 'hero' | 'compact'
})

const inputRef = ref(null)

defineExpose({ focus: () => inputRef.value?.focus?.() })
</script>

<style scoped>
.url-row { display:flex; align-items:center; gap: 8px; }
.search-field { position: relative; flex: 1 1 auto; min-width: 0; }
.search-field .input-macos { width: 100%; padding-left: 28px; height: 28px; }
.search-field .icon { position: absolute; left: 8px; top: 50%; transform: translateY(-50%); width: 14px; height: 14px; color: var(--macos-text-tertiary); }
.trailing-controls { display:inline-flex; align-items:center; gap: 8px; }
.trailing-controls .select-macos { width: 140px; }

/* size variants */
.url-row.hero .search-field .input-macos { height: 34px; font-size: var(--fs-title); }
.url-row.compact .search-field .input-macos { height: 28px; font-size: var(--fs-base); }
</style>
