<template>
  <div class="chat-message-list">
    <ChatMessageBubble
      v-for="msg in messages"
      :key="msg.id"
      :message="msg"
      :app-label="appLabel"
      :provider-label="providerLabel"
      :app-icon="appIcon"
      :provider-icon="providerIcon"
    />
    <ChatMessageBubble
      v-if="pendingMessage"
      :message="pendingMessage"
      :app-label="appLabel"
      :provider-label="providerLabel"
      :app-icon="appIcon"
      :provider-icon="providerIcon"
      :is-pending="true"
      :pending-content="pendingWaitingLabel"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import ChatMessageBubble from '@/components/chat/ChatMessageBubble.vue'

const props = defineProps({
  messages: { type: Array, default: () => [] },
  appLabel: { type: String, required: true },
  providerLabel: { type: String, required: true },
  appIcon: { type: String, default: 'leaf' },
  providerIcon: { type: String, default: 'terminal' },
  pendingRequestId: { type: String, default: '' },
  pendingWaitText: { type: String, default: '' },
  pendingRunningLabel: { type: String, default: '' },
  pendingWaitingLabel: { type: String, default: '' },
})

const pendingMessage = computed(() => {
  if (!props.pendingRequestId || !props.pendingWaitText) return null
  return {
    id: props.pendingRequestId,
    role: 'provider',
    roleClass: 'from-provider',
    kindLabel: props.pendingRunningLabel,
    timeText: props.pendingWaitText,
    content: '',
    blocks: [],
  }
})
</script>

<style scoped>
.chat-message-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>

