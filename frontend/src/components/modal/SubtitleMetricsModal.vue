<template>
  <div v-if="show" class="macos-modal" role="dialog" aria-modal="true">
    <div class="modal-card card-frosted card-translucent" tabindex="-1">
      <!-- Header: close on left, title on right (two-layer structure) -->
      <div class="modal-header">
        <ModalTrafficLights @close="$emit('close')" />
        <div class="right">
          <div class="title-text">{{ $t('subtitle.list.current_metrics') }}</div>
          <div class="title-extra" v-if="standardName || standardDesc">
            <span class="chip-frosted chip-sm chip-translucent strong-badge"><span class="chip-label">{{ standardName || '-' }}</span></span>
            <span class="desc">{{ standardDesc }}</span>
          </div>
        </div>
      </div>

      <!-- Body: group box + rows (unified background, outline, inner separators) -->
      <div class="modal-body">
        <!-- Current standard summary moved to header (kept body clean) -->

        <div class="rows">
          <div class="row">
            <div class="k"><span class="chip-frosted chip-sm chip-translucent abbr-pill"><span class="chip-label">{{ $t('subtitle.list.cps') }}</span></span></div>
            <div class="v">
              <div class="row-title">{{ $t('subtitle.list.cps_fullname') }}</div>
              <div class="row-desc">{{ $t('subtitle.list.cps_desc') }}</div>
            </div>
          </div>
          <div class="row">
            <div class="k"><span class="chip-frosted chip-sm chip-translucent abbr-pill"><span class="chip-label">{{ $t('subtitle.list.wpm') }}</span></span></div>
            <div class="v">
              <div class="row-title">{{ $t('subtitle.list.wpm_fullname') }}</div>
              <div class="row-desc">{{ $t('subtitle.list.wpm_desc') }}</div>
            </div>
          </div>
          <div class="row">
            <div class="k"><span class="chip-frosted chip-sm chip-translucent abbr-pill"><span class="chip-label">{{ $t('subtitle.list.cpl') }}</span></span></div>
            <div class="v">
              <div class="row-title">{{ $t('subtitle.list.cpl_fullname') }}</div>
              <div class="row-desc">{{ $t('subtitle.list.cpl_desc') }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  show: { type: Boolean, default: false },
  standardName: { type: String, default: '' },
  standardDesc: { type: String, default: '' },
})
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
</script>

<style scoped>
.macos-modal { position: fixed; inset: 0; z-index: 2000; display:flex; align-items:center; justify-content:center; padding: 16px; background: rgba(0,0,0,0.2); backdrop-filter: blur(8px); }
.modal-card { width: min(780px, 96vw); max-height: 80vh; border-radius: 12px; display:flex; flex-direction: column; overflow: hidden; }
/* Always-on active frosted look */
.modal-card.card-frosted.card-translucent { background: color-mix(in oklab, var(--macos-surface) 76%, transparent); border: 1px solid rgba(255,255,255,0.28); box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0,0,0,0.24); }
.modal-header { display:flex; align-items:center; justify-content: space-between; padding: 8px 10px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.modal-header .left { display:flex; align-items:center; gap:6px; }
.modal-header .right { display:flex; align-items:center; gap: 10px; min-width: 0; }
.modal-header .right .title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); }
.modal-header .right .title-extra { display:flex; align-items:center; gap: 8px; min-width: 0; }
.modal-header .right .title-extra .desc { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.modal-body { padding: 12px; overflow: auto; background: var(--macos-surface); }
/* traffic lights */
/* no traffic lights for sheet-like modal */

/* body summary removed (moved to header) */

.rows { border: none; border-radius: 10px; background: transparent; }
.row { display: grid; grid-template-columns: 110px 1fr; align-items: center; gap: 10px; padding: 8px 10px; position: relative; }
.row + .row::before { content: ''; position: absolute; top: 0; left: 10px; right: 10px; height: 1px; background: var(--macos-divider-weak); }
.row-title { font-size: var(--fs-sub); font-weight: 600; color: var(--macos-text-primary); }
.row-desc { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-top: 2px; }
.abbr-pill { display:inline-block; font-size: var(--fs-sub); font-weight: 600; }
</style>
