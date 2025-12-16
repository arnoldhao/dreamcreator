<template>
  <div class="gl-root">
    <div class="macos-group gl-header">
      <div class="grow"></div>
    </div>
    <div class="macos-box card-frosted card-translucent">
      <div class="gl-sets">
        <div class="macos-row gl-row" v-for="l in langs" :key="l.code" @click="openInSettings(l.code)">
          <span class="k k-left" :title="l.name || l.code">
            <span class="name one-line">{{ l.name }}</span>
            <!-- <span class="code chip-frosted chip-lg chip-translucent"><span class="chip-label mono">{{ l.code }}</span></span> -->
          </span>
          <span class="v v-actions">
            <div class="ops-actions">
              <button class="btn-chip-icon btn-danger" :data-tooltip="'Delete'" @click.stop="delLang(l)">
                <Icon name="trash" class="w-4 h-4" />
              </button>
            </div>
          </span>
        </div>
        <!-- Add new language as the last list item -->
        <div class="macos-row gl-row gl-row-add" @click="openCreate">
          <div class="add-wrap">
            <div class="btns">
              <button class="btn-chip" :data-tooltip="t('common.add')" @click.stop="openCreate">
                <Icon name="plus" class="w-4 h-4" />
              </button>
              <button class="btn-chip btn-danger" :data-tooltip="t('subtitle.target_languages.restore_defaults')" @click.stop="restoreDefaults">
                <Icon name="refresh" class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="p-2 flex justify-center">
      <div class="stats-pills">
        <span class="stats-pill">{{ langs.length }} {{ t('subtitle.target_languages.title') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events, Window } from '@wailsio/runtime'
import Icon from '@/components/base/Icon.vue'
import { useTargetLanguagesStore } from '@/stores/targetLanguages.js'

const { t } = useI18n()
const tlStore = useTargetLanguagesStore()
const langs = computed(() => tlStore.list || [])

async function loadLangs() {
  try {
    await tlStore.ensureLoaded()
  } catch {}
}

function openSettingsRoute(route) {
  try { Events.Emit('settings:navigate', route) } catch {}
  try {
    const sw = Window.Get('settings')
    try { sw.UnMinimise() } catch {}
    try { sw.Show() } catch {}
    try { sw.Focus() } catch {}
  } catch {}
}

function openInSettings(code) {
  if (!code) return
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'target_languages', llmAssetsId: String(code) })
}

function openCreate() {
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'target_languages', llmAssetsId: 'new' })
}

async function delLang(l) {
  if (!l?.code) return
  const confirmed = window?.$dialog?.confirm
    ? await new Promise((resolve) => {
        window.$dialog.confirm(t('subtitle.target_languages.delete_confirm', { code: l.code }), {
          title: t('common.confirm'),
          positiveText: t('common.delete'),
          negativeText: t('common.cancel'),
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        })
      })
    : window.confirm(t('subtitle.target_languages.delete_confirm', { code: l.code }))
  if (!confirmed) return
  try {
    await tlStore.remove(l.code)
    $message?.success?.(t('common.deleted'))
  } catch (e) {
    console.error('Delete language failed:', e)
    $message?.error?.(t('common.delete_failed'))
  }
}

async function restoreDefaults() {
  const confirmed = window?.$dialog?.confirm
    ? await new Promise((resolve) => {
        window.$dialog.confirm(t('subtitle.target_languages.restore_confirm'), {
          title: t('common.confirm'),
          positiveText: t('common.reset'),
          negativeText: t('common.cancel'),
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        })
      })
    : window.confirm(t('subtitle.target_languages.restore_confirm'))
  if (!confirmed) return
  try {
    await tlStore.resetToDefaults()
    $message?.success?.(t('subtitle.target_languages.restored'))
  } catch (e) {
    console.error('Reset languages failed:', e)
    $message?.error?.(t('common.save_failed'))
  }
}

onMounted(loadLangs)
</script>

<style scoped>
.gl-root { padding: 8px; font-size: var(--fs-base); }
.gl-header { display:flex; align-items:center; gap:8px; }
.gl-sets { display:flex; flex-direction: column; }
.gl-row.gl-row-add { display:flex; justify-content:center; }
.gl-row.gl-row-add .add-wrap { padding: 0; }
.gl-row.gl-row-add .btns { display:flex; gap: 8px; }
.gl-row { cursor: default; }
.gl-row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
.k-left { display:flex; align-items:center; gap: 8px; }
.gl-row .k-left { min-width: 0; }
.gl-row .k-left .name { display:inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.v-actions { display:flex; align-items:center; gap: 8px; justify-content: flex-end; }
.ops-actions { display:flex; align-items:center; gap: 6px; }
.grow { flex: 1 1 auto; }
.gl-header + .macos-box { margin-top: 8px; }
.add-inline { display:flex; align-items:center; gap: 8px; width: 100%; }
.input-xs { height: 28px; padding: 4px 8px; font-size: var(--fs-sub); }
.code .chip-label { font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace); }

/* Bottom stats styles to align with other inspectors */
.meta-group { display:flex; align-items:center; gap: 10px; font-size: 12px; color: var(--macos-text-secondary); }
.meta-group .item { display: inline-flex; align-items: baseline; gap: 4px; }
.meta-group .item .num { font-weight: 600; color: var(--macos-text-primary); line-height: 1; }
.meta-group .item .label { line-height: 1; }
.divider-v { width:1px; height: 16px; background: var(--macos-divider-weak); opacity: 0.8; }

/* Inspector bottom stats: align style with subtitle editor bottom pill */
.stats-pills { display:flex; align-items:center; gap: 8px; }
.stats-pill { display:inline-flex; align-items:center; height: 22px; padding: 0 10px; border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.22);
  background: color-mix(in oklab, var(--macos-surface) 78%, transparent);
  color: var(--macos-text-secondary); font-size: var(--fs-sub);
  -webkit-backdrop-filter: var(--macos-surface-blur); backdrop-filter: var(--macos-surface-blur);
  box-shadow: var(--macos-shadow-1);
}
</style>
