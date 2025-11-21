<template>
  <div class="chat-row" :class="message.roleClass">
    <div class="avatar">
      <Icon v-if="message.role === 'app'" :name="appIcon" class="w-4 h-4" />
      <Icon v-else :name="providerIcon" class="w-4 h-4" />
    </div>
    <div class="bubble" :class="{ 'bubble-pending': isPending }">
      <div class="bubble-header">
        <span class="who">
          {{ message.role === 'app' ? appLabel : providerLabel }}
        </span>
        <span v-if="message.kindLabel" class="kind">{{ message.kindLabel }}</span>
        <span v-if="message.timeText" class="time mono">{{ message.timeText }}</span>
      </div>
      <div class="content">
        <template v-if="!isPending">
          <template v-if="message.blocks && message.blocks.length">
            <template v-for="(blk, idx) in message.blocks" :key="idx">
              <p v-if="blk.type === 'text'" class="text-block">{{ blk.text }}</p>
              <pre v-else-if="blk.type === 'code'" class="code-block"><code>{{ blk.text }}</code></pre>
              <pre v-else-if="blk.type === 'json'" class="json-block mono"><code>{{ formatJsonBlock(blk) }}</code></pre>
              <div v-else-if="blk.type === 'markdown'" class="markdown-block" v-html="blk.html"></div>
              <p v-else class="text-block">{{ blk.text }}</p>
            </template>
          </template>
          <pre v-else class="plain-content">{{ message.content }}</pre>
        </template>
        <pre v-else class="plain-content pending-content mono">
{{ pendingContent }}
        </pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import Icon from '@/components/base/Icon.vue'

const props = defineProps({
  message: { type: Object, required: true },
  appLabel: { type: String, required: true },
  providerLabel: { type: String, required: true },
  appIcon: { type: String, default: 'leaf' },
  providerIcon: { type: String, default: 'terminal' },
  isPending: { type: Boolean, default: false },
  pendingContent: { type: String, default: '' },
})

function formatJsonBlock(blk) {
  try {
    if (blk && blk.value !== undefined) {
      return JSON.stringify(blk.value, null, 2)
    }
    if (blk && typeof blk.text === 'string') return blk.text
    if (blk && typeof blk.raw === 'string') return blk.raw
  } catch {
    // ignore
  }
  return ''
}
</script>

<style scoped>
.chat-row {
  display: flex;
  gap: 8px;
}
.chat-row.from-app {
  flex-direction: row-reverse;
}
.avatar {
  flex: 0 0 auto;
  width: 24px;
  height: 24px;
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in oklab, var(--macos-surface) 80%, transparent);
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.18);
  color: var(--macos-text-primary);
}
.bubble {
  max-width: 78%;
  border-radius: 12px;
  padding: 8px 10px;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.2);
  -webkit-user-select: text;
  -moz-user-select: text;
  user-select: text;
}
.chat-row.from-app .bubble {
  background: color-mix(in oklab, var(--macos-blue) 24%, var(--macos-surface) 80%);
}
.chat-row.from-provider .bubble {
  min-width: 220px;
}
.bubble-pending {
  opacity: 0.85;
  width: 220px; /* fixed width to avoid jitter when waiting time changes */
}
.bubble-header {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  margin-bottom: 4px;
}
.bubble-header .who {
  flex: 1 1 auto;
  min-width: 0;
  font-size: 11px;
  font-weight: 600;
  color: var(--macos-text-primary);
}
.bubble-header .kind {
  flex: 0 0 70px;
  font-size: 11px;
  color: var(--macos-text-secondary);
  text-align: center;
}
.bubble-header .time {
  flex: 0 0 64px;
  font-size: 10px;
  color: var(--macos-text-secondary);
  text-align: right;
}
.content {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--macos-text-primary);
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.text-block,
.plain-content {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
.pending-content {
  font-size: 11px;
  color: var(--macos-text-secondary);
}
.markdown-block {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
}
.markdown-block h1,
.markdown-block h2,
.markdown-block h3,
.markdown-block h4,
.markdown-block h5,
.markdown-block h6 {
  margin: 0.4em 0 0.25em;
  font-weight: 600;
}
.markdown-block ul,
.markdown-block ol {
  margin: 0.25em 0;
  padding-left: 1.3em;
}
.markdown-block li {
  margin: 0.1em 0;
}
.code-block,
.json-block {
  margin: 0;
  padding: 6px 8px;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 11px;
}
.code-block {
  background: color-mix(in oklab, var(--macos-surface) 78%, #000 14%);
}
.json-block {
  background: color-mix(in oklab, var(--macos-surface) 82%, var(--macos-blue) 8%);
}
</style>
