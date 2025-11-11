<template>
  <div class="sr-right">
    <div class="sr-card p-0">
      <div class="sr-card-body">
        <div v-for="(dep, key) in dependencies" :key="key">
          <!-- 主行：名称 + 状态 + 操作 -->
          <div class="sr-row dep-row">
            <div class="dep-left">
              <span class="dep-dot" :class="{
                ok: dep.available && !dep.needUpdate,
                warn: dep.available && dep.needUpdate,
                err: !dep.available
              }"></span>
              <span class="dep-name">{{ dep.name }}</span>
              <span v-if="dep.available" class="dep-badge dep-ok">{{ $t('settings.dependency.installed') }}</span>
              <span v-else class="dep-badge dep-miss">{{ $t('settings.dependency.not_installed') }}</span>
              <span v-if="dep.needUpdate" class="dep-badge dep-warn">{{ $t('settings.dependency.update') }}</span>
              <!-- Show last check failure badge when an error exists -->
              <button
                v-if="dep.lastCheckAttempted && !dep.lastCheckSuccess"
                type="button"
                class="dep-badge dep-err"
                @click="showLastCheckError(key)"
                :title="$t('settings.dependency.check_updates_failed')"
              >
                {{ $t('settings.dependency.check_updates_failed') }}
              </button>
            </div>
            <div class="sr-control control-short dep-actions">
              <template v-if="dep.available">
                <button
                  v-if="dep.needUpdate && !dep.installing"
                  @click="showMirrorSelector(key, 'update')"
                  class="btn-chip-ghost btn-primary"
                >
                  <Icon name="arrow-left-right" class="w-4 h-4 mr-1" /> {{ $t('settings.dependency.update') }}
                </button>
                <button v-else-if="dep.installing" class="btn-chip-ghost" disabled>
                  <div class="btn-spinner mr-2"></div>{{ $t('settings.dependency.installing') }}
                </button>
              </template>
              <template v-else>
                <button class="btn-chip-ghost" @click="repairDependency(key)" :disabled="isCheckUpdatesDisabled">
                  <Icon name="download" class="w-4 h-4 mr-1" /> {{ $t('settings.dependency.repair') }}
                </button>
                <button class="btn-chip-ghost" @click="showMirrorSelector(key, 'install')" :disabled="isCheckUpdatesDisabled">
                  <Icon name="download" class="w-4 h-4 mr-1" /> {{ $t('settings.dependency.install') }}
                </button>
              </template>
            </div>
          </div>

          <!-- 元信息行：版本与路径（可打开） -->
          <div class="sr-row dep-meta dep-row-sm" v-if="dep.available">
            <!-- 左侧：版本信息（左对齐） -->
            <div class="dep-meta-left">
              <span class="meta-item">{{ $t('settings.dependency.version') }}: {{ dep.version || '-' }}<template v-if="dep.needUpdate"> → {{ dep.latestVersion }}</template></span>
            </div>
            <!-- 右侧：路径（右对齐） + 打开 -->
            <div class="sr-control control-short dep-meta-right">
              <span class="meta-item path" :title="dep.path">{{ dep.path }}</span>
              <button class="icon-chip-ghost" type="button" :disabled="!dep.path" @click="openDirectory(dep.path)" title="Open">
                <Icon name="folder" class="w-4 h-4" />
              </button>
            </div>
          </div>

          <!-- 进度行 -->
          <div class="sr-row dep-progress" v-if="dep.installing">
            <div class="dep-progress-bar"><div class="dep-progress-fill" :style="{ width: getProgressPercentage(dep) + '%' }"></div></div>
            <div class="sr-control control-short"><span class="meta-item">{{ dep.installProgress }}</span></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 镜像选择模态框 -->
    <div v-if="showMirrorModal" class="macos-modal" @click.self="closeMirrorModal">
      <div class="modal-card mirror-modal card-frosted card-translucent" role="dialog" aria-modal="true">
        <div class="modal-header sheet">
          <ModalTrafficLights @close="closeMirrorModal" />
          <div class="title-area">
            <div class="title-text">{{ $t('settings.dependency.select_mirror') }}</div>
          </div>
        </div>

        <div class="modal-body">
          <div class="mirror-list">
            <div
              v-for="mirror in availableMirrors"
              :key="mirror.name"
              class="mirror-option"
              :class="{ selected: selectedMirror === mirror.name }"
              @click="selectedMirror = mirror.name"
            >
              <div class="mirror-meta">
                <div class="mirror-text">
                  <div class="mirror-name">{{ mirror.name }}</div>
                  <div class="mirror-desc text-secondary">{{ mirror.description }}</div>
                </div>
                <span v-if="mirror.recommended" class="recommend-badge">
                  {{ $t('settings.dependency.recommended') }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn-chip" type="button" @click="closeMirrorModal">
            {{ $t('common.cancel') }}
          </button>
          <button
            :class="['btn-chip', { 'btn-primary': currentAction === 'update' }]"
            type="button"
            @click="performAction"
            :disabled="!selectedMirror"
          >
            {{ currentAction === 'install' ? $t('settings.dependency.install') : $t('settings.dependency.update') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { ListMirrors } from 'wailsjs/go/api/DependenciesAPI'
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n'
import useDependenciesStore from '@/stores/dependencies'
import eventBus from '@/utils/eventBus.js'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'

export default {
    name: 'Dependency',
    components: {
        ModalTrafficLights
    },
    setup() {
        const { t } = useI18n()
        const dependenciesStore = useDependenciesStore()

        // 响应式数据
        const dependencies = computed(() => dependenciesStore.dependencies)
        const showMirrorModal = ref(false)
        const availableMirrors = ref([])
        const selectedMirror = ref('')
        const currentDepType = ref('')
        const currentAction = ref('') // 'install' or 'update'
        // 更新相关
        const isChecking = ref(false)
        const allowCheckUpdates = computed(() => dependenciesStore.allowCheckUpdates)
        // 组合禁用条件
        const isCheckUpdatesDisabled = computed(() => {
            return isChecking.value || !allowCheckUpdates.value
        })
        // 验证相关
        const isValidating = computed(() => dependenciesStore.validating)

        // 验证依赖
        const validateDependencies = async () => {
            await dependenciesStore.validateDependencies()
        }

        // 检查更新
        const checkUpdates = async () => {
            isChecking.value = true
            try {
                const result = await dependenciesStore.checkUpdates()
                if (result && result.hasFailures) {
                    $message.warning(t('settings.dependency.check_updates_partial'))
                } else {
                    $message.success(t('settings.dependency.check_updates_success'))
                }
            } catch (error) {
                console.error('Check updates failed:', error)
                $message.error(t('settings.dependency.check_updates_failed'))
            } finally {
                isChecking.value = false
            }
        }

        // 显示镜像选择器
        const showMirrorSelector = async (depType, action) => {
            try {
                const response = await ListMirrors(depType)
                if (response.success) {
                    const mirrors = JSON.parse(response.data)
                    availableMirrors.value = mirrors

                    // 选择推荐的镜像
                    const recommended = mirrors.find(m => m.recommended)
                    selectedMirror.value = recommended ? recommended.name : (mirrors[0]?.name || '')

                    currentDepType.value = depType
                    currentAction.value = action
                    showMirrorModal.value = true
                }
            } catch (error) {
                console.error('Failed to load mirrors:', error)
                $message?.error?.(t('settings.dependency.load_mirrors_failed'))
            }
        }

        // 执行操作
        const performAction = () => {
            if (currentAction.value === 'install') {
                dependenciesStore.installDependency(
                    currentDepType.value,
                    'latest',
                    selectedMirror.value
                )

            } else if (currentAction.value === 'update') {
                dependenciesStore.updateDependency(
                    currentDepType.value,
                    selectedMirror.value
                )
            }

            // 操作完成后关闭模态框
            closeMirrorModal()
        }

        const repairDependency = async (type) => {
            await dependenciesStore.repairDependency(type)
        }

        const showLastCheckError = (type) => {
            const dep = dependencies.value[type]
            if (!dep) return
            const headerBase = t('settings.dependency.check_updates_failed')
            const codeKey = `settings.dependency.error.${dep.lastCheckErrorCode || 'unknown'}`
            const codeText = t(codeKey)
            const header = `${headerBase}: ${codeText}`
            const details = (dep.lastCheckError || '').trim()
            const content = details ? `${header}\n\n${details}` : header
            $dialog.error({
                title: t('settings.dependency.check_updates_failed'),
                content,
            })
        }

        // 关闭镜像选择模态框
        const closeMirrorModal = () => {
            showMirrorModal.value = false
            selectedMirror.value = ''
            currentDepType.value = ''
            currentAction.value = ''
            availableMirrors.value = []
        }

        // 打开目录
        const openDirectory = async (path) => {
            try {
                const p = (path || '').trim()
                if (!p) return
                await OpenDirectory(p)
            } catch (error) {
                console.error('Failed to open directory:', error)
            }
        }

        const getProgressPercentage = (dep) => {
            // 根据当前状态返回对应的进度百分比
            if (dep.installing) {
                return dep.installProgressPercent || 0
            } 
            return 0
        }

        // 组件挂载
        const onQuickValidate = () => dependenciesStore.quickValidateDependencies(true)

        onMounted(async () => {
            // 进入依赖页先做一次快速校验，体验更好；需要深度校验可用工具栏按钮触发
            await dependenciesStore.quickValidateDependencies(false)
            // toolbar actions
            eventBus.on('dependency:validate', validateDependencies)
            eventBus.on('dependency:quick-validate', onQuickValidate)
            eventBus.on('dependency:check-updates', checkUpdates)
        })

        onUnmounted(() => {
            eventBus.off('dependency:validate', validateDependencies)
            eventBus.off('dependency:quick-validate', onQuickValidate)
            eventBus.off('dependency:check-updates', checkUpdates)
        })

        return {
            dependencies,
            showMirrorModal,
            availableMirrors,
            selectedMirror,
            currentDepType,
            currentAction,
            isCheckUpdatesDisabled,
            isValidating,
            validateDependencies,
            checkUpdates,
            showMirrorSelector,
            repairDependency,
            showLastCheckError,
            closeMirrorModal,
            performAction,
            openDirectory,
            getProgressPercentage,
            t
        }
    }
}
</script>

<style lang="scss" scoped>
.sr-right { font-size: var(--fs-base); background: var(--macos-background); padding: 12px; min-height: 100%; }
.sr-card { border: 1px solid var(--macos-separator); border-radius: 10px; background: color-mix(in oklab, var(--macos-background) 97%, var(--macos-text-secondary) 3%); }
/* list rows inside card */
.sr-card .sr-row { display: grid; grid-template-columns: 1fr 160px; align-items: center; gap: 12px; padding: 8px 6px; min-height: 36px; border-bottom: 1px solid var(--macos-divider-weak); margin: 0 8px; }
.sr-card .sr-row:last-child { border-bottom: none; }
.dep-left { display: flex; align-items: center; gap: 8px; min-width: 0; }
.dep-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--macos-divider-weak); }
.dep-dot.ok { background: #30d158; }
.dep-dot.warn { background: #ff9f0a; }
.dep-dot.err { background: #ff453a; }
.dep-name { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); margin-right: 4px; }
  .dep-badge { font-size: var(--fs-sub); padding: 2px 8px; border-radius: 999px; border: 1px solid var(--macos-separator); background: transparent; color: var(--macos-text-secondary); box-shadow: none; }
.dep-badge.dep-ok { border-color: #30d158; }
.dep-badge.dep-warn { border-color: #ff9f0a; }
.dep-badge.dep-miss { border-color: #ff453a; }
.dep-badge.dep-err { border-color: #ff453a; color: #ff453a; cursor: pointer; background: transparent; }
.dep-badge.dep-err:hover { background: color-mix(in oklab, #ff453a 12%, transparent); }
/* removed sr-icon-btn (use icon-chip-ghost) */
.dep-meta-left { display: flex; align-items: center; gap: 12px; min-width: 0; }
.meta-item { font-size: var(--fs-sub); color: var(--macos-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.meta-item.path { max-width: 60ch; }
.dep-meta-right { display: inline-flex; align-items: center; gap: 8px; width: 160px; justify-content: flex-end; min-width: 0; }
.dep-meta-right .meta-item.path { flex: 1; text-align: right; }
.dep-row-sm { min-height: 30px; padding: 6px 6px; }
.dep-progress-bar { height: 2px; width: 100%; background: var(--macos-divider-weak); border-radius: 999px; overflow: hidden; }
.dep-progress-fill { height: 100%; background: var(--macos-blue); }
/* subtle text */
.text-secondary { color: var(--macos-text-secondary); }
/* Modal (macOS look) */
/* use global .macos-modal overlay */
.macos-modal { animation: fadeIn 0.18s ease-out; }
.mirror-modal { width: 440px; max-width: calc(100% - 40px); background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 12px; box-shadow: var(--macos-shadow-3); overflow: hidden; display: flex; flex-direction: column; }
.modal-header.sheet { display: flex; align-items: center; justify-content: flex-start; gap: 12px; padding: 10px 12px; background: var(--macos-background-secondary); border-bottom: 1px solid var(--macos-separator); }
.title-area { flex: 1; display: flex; align-items: center; justify-content: flex-end; min-width: 0; }
.title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); }
.modal-body { padding: 12px; max-height: 60vh; overflow-y: auto; }
.modal-footer { display: flex; align-items: center; justify-content: flex-end; gap: 8px; padding: 10px 12px; background: var(--macos-background-secondary); border-top: 1px solid var(--macos-separator); }
.mirror-list { display: flex; flex-direction: column; gap: 8px; }
.mirror-option { padding: 10px 12px; border: 1px solid var(--macos-separator); border-radius: 10px; background: var(--macos-background); cursor: pointer; transition: background .16s ease, border-color .16s ease, box-shadow .16s ease; }
.mirror-option:hover { background: color-mix(in oklab, var(--macos-background-secondary) 82%, transparent); }
.mirror-option.selected { border-color: var(--macos-blue); box-shadow: 0 0 0 1px color-mix(in oklab, var(--macos-blue) 30%, transparent); background: color-mix(in oklab, var(--macos-blue) 14%, transparent); }
.mirror-option:focus-visible { outline: none; box-shadow: 0 0 0 2px color-mix(in oklab, var(--macos-blue) 42%, transparent); }
.mirror-meta { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.mirror-text { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.mirror-name { font-weight: 600; color: var(--macos-text-primary); }
.mirror-desc { font-size: var(--fs-sub); }
.recommend-badge { font-size: var(--fs-sub); padding: 2px 8px; border-radius: 999px; background: color-mix(in oklab, var(--macos-blue) 18%, transparent); color: var(--macos-blue); }
/* spinner tuned for button visibility */
.btn-spinner { width: 14px; height: 14px; border: 2px solid transparent; border-top-color: currentColor; border-right-color: currentColor; border-radius: 50%; animation: spin .8s linear infinite; }

/* Dependency actions: keep buttons on one line and match macOS style */
.dep-row { grid-template-columns: 1fr auto !important; }
.dep-actions { width: auto !important; display: inline-flex; gap: 8px; justify-content: flex-end; }
.dep-actions .btn-chip-ghost { height: 28px; padding: 0 10px; }
</style>
