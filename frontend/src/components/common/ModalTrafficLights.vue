<template>
  <div class="modal-traffic" aria-label="Modal controls">
    <button class="tl tl-close" aria-label="Close" @click.stop="$emit('close')">
      <span class="dot dot-red"></span>
    </button>
    <div class="tl tl-min disabled" aria-hidden="true">
      <span class="dot dot-yellow disabled"></span>
    </div>
    <div class="tl tl-max disabled" aria-hidden="true">
      <span class="dot dot-green disabled"></span>
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>

<style scoped>
.modal-traffic { display: flex; align-items: center; gap: 6px; }
.tl { border: 0; padding: 0; background: transparent; display: inline-flex; align-items: center; justify-content: center; width: 14px; height: 14px; }
.dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; position: relative; box-shadow: inset 0 0 0 1px rgba(0,0,0,0.12); }
.dot-red { background: #ff5f56; }
.dot-yellow { background: #d9d9d9; }
.dot-green { background: #d9d9d9; }
.tl-close:hover .dot { box-shadow: inset 0 0 0 1px rgba(0,0,0,0.18), inset 0 1px 0 rgba(255,255,255,0.35); }
.disabled { opacity: .6; cursor: default; }
</style>

