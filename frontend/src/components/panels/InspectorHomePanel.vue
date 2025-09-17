<template>
  <div class="insp-empty">
    <div class="emoji" aria-hidden="true">{{ emoji }}</div>
    <div class="title">{{ $t('inspector.empty.detail_title') }}</div>
    <div class="hint">{{ $t('inspector.empty.detail_hint') }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
defineEmits(['open-modal'])

const emojiSet = ['🗂️','📁','📂','🗃️','🗄️','🧾','📝']
const emoji = ref('🗂️')

function pickRandom() {
  const i = Math.floor(Math.random() * emojiSet.length)
  emoji.value = emojiSet[i]
}

onMounted(() => {
  // 每次打开该面板（挂载）时挑选一次
  pickRandom()
})
</script>

<style scoped>
.insp-empty { height: 100%; min-height: 100%; padding: 24px 12px; text-align: center; color: var(--macos-text-secondary); display:flex; flex-direction:column; align-items:center; justify-content:center; gap: 8px; }
.insp-empty .emoji { font-size: 28px; line-height: 1; animation: insp-bob 3s ease-in-out infinite; }
.insp-empty .title { font-weight: 700; font-size: var(--fs-base); color: var(--macos-text-primary); }
.insp-empty .hint { font-size: var(--fs-sub); color: var(--macos-text-secondary); }

@keyframes insp-bob {
  0% { transform: translateY(0); opacity: 0.95; }
  50% { transform: translateY(-3px); opacity: 1; }
  100% { transform: translateY(0); opacity: 0.95; }
}
</style>
