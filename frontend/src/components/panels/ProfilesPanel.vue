<template>
  <div class="pf-root">
    <div class="macos-group pf-header"><div class="grow"></div></div>
    <div class="macos-box card-frosted card-translucent">
      <div class="pf-rows">
        <div
          v-for="p in profiles"
          :key="p.id"
          class="macos-row pf-row"
          @click="openInSettings(p.id)"
        >
          <span class="k k-left" :title="p.name || ('Profile ' + p.id.slice(0,6))">
            <span class="name one-line">{{ p.name || ('Profile ' + p.id.slice(0,6)) }}</span>
            <span class="meta-group small pf-meta" :title="`T=${p.temperature ?? 0}, TopP=${p.top_p ?? 1}, ${p.json_mode ? 'JSON' : 'Text'}`" aria-label="profile meta">
              <span class="item">T=<span class="val">{{ p.temperature ?? 0 }}</span></span>
              <span class="divider-v"></span>
              <span class="item">TopP=<span class="val">{{ p.top_p ?? 1 }}</span></span>
              <span class="divider-v"></span>
              <span class="item"><span class="val">{{ p.json_mode ? 'JSON' : 'Text' }}</span></span>
            </span>
          </span>
          <span class="v v-actions">
            <div class="ops-actions">
              <button class="btn-chip-icon btn-danger" :data-tooltip="$t('common.delete')" data-tip-pos="top" @click.stop="onDelete(p)"><Icon name="trash" class="w-4 h-4"/></button>
            </div>
          </span>
        </div>
        <!-- Add new profile as the last list item -->
        <div class="macos-row pf-row pf-row-add" @click="newProfile()">
          <div class="add-wrap">
            <button class="btn-chip" :data-tooltip="$t('profiles.add')" @click.stop="newProfile()">
              <Icon name="plus" class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
    <div v-if="!profiles.length" class="empty muted">{{ $t('profiles.empty') }}</div>
    <!-- bottom stats: align style with subtitle editor bottom pill -->
    <div class="p-2 flex justify-center">
      <div class="stats-pills">
        <span class="stats-pill">{{ profiles.length }} {{ $t('profiles.inspector_title') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Events, Window } from '@wailsio/runtime'
import Icon from '@/components/base/Icon.vue'
import { listGlobalProfiles, deleteGlobalProfile } from '@/services/llmProviderService.js'

const profiles = ref([])

async function load(){
  try { profiles.value = await listGlobalProfiles() } catch { profiles.value = [] }
}

function openSettingsRoute(route){
  try { Events.Emit('settings:navigate', route) } catch {}
  try {
    const sw = Window.Get('settings')
    try { sw.UnMinimise() } catch {}
    try { sw.Show() } catch {}
    try { sw.Focus() } catch {}
  } catch {}
}

function openInSettings(id){
  if (!id) return
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'profiles', llmAssetsId: String(id) })
}

function newProfile(){
  openSettingsRoute({ section: 'llm_assets', llmAssetsKind: 'profiles', llmAssetsId: 'new' })
}

async function onDelete(p){
  if (!p?.id) return
  const baseName = p.name || ('Profile ' + p.id.slice(0,6))
  const showName = baseName.length > 120 ? baseName.slice(0,120) + '…' : baseName
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
    await deleteGlobalProfile(p.id)
    await load()
    window.$message?.success?.(window.$t ? window.$t('common.deleted') : 'Deleted')
  } catch(e){ window.$message?.error?.(e?.message || 'Delete failed') }
}

onMounted(() => { load() })
</script>

<style scoped>
.pf-root { padding: 8px; font-size: var(--fs-base); }
.pf-header { display:flex; align-items:center; gap:8px; }
.pf-header + .macos-box { margin-top: 8px; }
.pf-rows { display: flex; flex-direction: column; }
.pf-row { cursor: pointer; grid-template-columns: 1fr auto; }
/* subtle hover highlight to match GlossaryPanel */
.pf-row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
.pf-row + .pf-row::before { content: ''; position: absolute; top: 0; left: 10px; right: 10px; height: 1px; background: var(--macos-divider-weak); }

.k-left { display:flex; align-items:center; gap: 8px; min-width: 0; }
.k-left .name { display:inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-weight: 500; }
/* meta pills: keep visible, allow name to shrink first */
.pf-meta { flex-shrink: 0; }

.v-actions { display:flex; align-items:center; gap: 8px; justify-content: flex-end; }
.ops-actions { display:flex; align-items:center; gap: 6px; }

/* Add-row placement centered */
.pf-row.pf-row-add .add-wrap { grid-column: 1 / -1; justify-self: center; padding: 0; }

.empty { padding: 12px; text-align: center; color: var(--macos-text-tertiary); }
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
