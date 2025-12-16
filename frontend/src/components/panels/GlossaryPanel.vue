<template>
  <div class="gl-root">
    <div class="macos-group gl-header">
      <div class="grow"></div>
    </div>
    
    <div class="macos-box card-frosted card-translucent">
      <div class="gl-sets">
        <div class="macos-row gl-row" v-for="s in glSets" :key="s.id" @click="openInSettings(s.id)">
          <span class="k k-left" :title="s.name">
            <span class="name one-line">{{ s.name }}</span>
            <span class="count chip-frosted chip-lg chip-translucent"><span class="chip-label">{{ glCounts[s.id] || 0 }}</span></span>
          </span>
          <span class="v v-actions">
            <div class="ops-actions">
              <button class="btn-chip-icon btn-danger" :data-tooltip="'Delete'" @click.stop="delSet(s)">
                <Icon name="trash" class="w-4 h-4" />
              </button>
            </div>
          </span>
        </div>
        <!-- Add new set as the last list item -->
        <div class="macos-row gl-row gl-row-add" @click="openCreateSet()">
          <div class="add-wrap">
            <button class="btn-chip" :data-tooltip="t('common.add')" @click.stop="openCreateSet()">
              <Icon name="plus" class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="p-2 flex justify-center">
      <div class="stats-pills">
        <span class="stats-pill">{{ glSets.length }} {{ t('glossary.sets') }}</span>
        <span class="stats-pill">{{ glTotalTerms }} {{ t('glossary.terms') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events, Window } from '@wailsio/runtime'
import Icon from '@/components/base/Icon.vue'
import { subtitleService } from '@/services/subtitleService.js'

const glSets = ref([])
const glCurrentSet = ref('')
const glCounts = ref({})
const glTotalTerms = ref(0)

const { t } = useI18n()

async function loadSets() {
  try {
    const sets = await subtitleService.listGlossarySets()
    glSets.value = Array.isArray(sets) ? sets : []
    if (!glCurrentSet.value && glSets.value.length) glCurrentSet.value = glSets.value[0].id
    const map = {}; let total = 0
    await Promise.all(glSets.value.map(async s => {
      try { const list = await subtitleService.listGlossaryBySet(s.id); map[s.id] = Array.isArray(list) ? list.length : 0; total += map[s.id] } catch { map[s.id] = 0 }
    }))
    glCounts.value = map
    glTotalTerms.value = total
  } catch { glSets.value = [] }
}
async function loadEntries() {
  if (!glCurrentSet.value) { return }
  try {
    const list = await subtitleService.listGlossaryBySet(glCurrentSet.value)
    const m = { ...glCounts.value }; m[glCurrentSet.value] = Array.isArray(list) ? list.length : 0; glCounts.value = m
    glTotalTerms.value = Object.values(glCounts.value).reduce((a,b)=>a+(b||0),0)
  } catch { }
}
function pickSet(id) { glCurrentSet.value = id; loadEntries() }

function openSettingsRoute(route) {
  try { Events.Emit('settings:navigate', route) } catch {}
  try {
    const sw = Window.Get('settings')
    try { sw.UnMinimise() } catch {}
    try { sw.Show() } catch {}
    try { sw.Focus() } catch {}
  } catch {}
}

function openInSettings(setId) {
  if (!setId) return
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'glossary', llmAssetsId: String(setId) })
}

function openCreateSet() {
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'glossary', llmAssetsId: 'new' })
}

async function delSet(set) {
  if (!set?.id) return
  const name = set.name || 'this set'
  const showName = name.length > 120 ? name.slice(0, 120) + '…' : name
  const confirmed = window?.$dialog?.confirm
    ? await new Promise((resolve) => {
        window.$dialog.confirm(`Delete ${showName}?`, {
          title: 'Confirm',
          positiveText: 'Delete',
          negativeText: 'Cancel',
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        })
      })
    : window.confirm(`Delete ${showName}?`)
  if (!confirmed) return
  try {
    await subtitleService.deleteGlossarySet(set.id)
    glSets.value = glSets.value.filter(x => x.id !== set.id)
    const m = { ...glCounts.value }; delete m[set.id]; glCounts.value = m
    if (glCurrentSet.value === set.id) { glCurrentSet.value = glSets.value[0]?.id || '' }
    await loadEntries()
    $message?.success?.('Deleted')
  } catch (e) {
    $message?.error?.('Delete set failed')
  }
}

onMounted(() => { loadSets().then(loadEntries) })
</script>

<style scoped>
.gl-root { padding: 8px; font-size: var(--fs-base); }
.gl-header { display:flex; align-items:center; gap:8px; }
.gl-sets { display:flex; flex-direction: column; }
.gl-row.gl-row-add .add-wrap { grid-column: 1 / -1; justify-self: center; padding: 0; }
.gl-row { cursor: pointer; }
/* hover 交互：行高亮 */
.gl-row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
/* 选中高亮已移除：点击行直接打开 Modal，无选中状态 */
.k-left { display:flex; align-items:center; gap: 8px; }
.gl-row .k-left { min-width: 0; }
.gl-row .k-left .name { display:inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.v-actions { display:flex; align-items:center; gap: 8px; justify-content: flex-end; }
.ops-actions { display:flex; align-items:center; gap: 6px; }
.grow { flex: 1 1 auto; }
/* add spacing between header and sets box */
.gl-header + .macos-box { margin-top: 8px; }

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
