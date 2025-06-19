<template>
    <div class="dependency-manager h-full bg-base-100 font-system flex flex-col">
        <!-- 标题栏 -->
        <div class="header-bar">
            <div class="header-content">
                <div class="header-left">
                    <div class="header-icon">
                        <v-icon class="w-5 h-5 text-base-content/60" name="md-settings-outlined"></v-icon>
                    </div>
                    <h1 class="header-title">{{ $t('settings.dependency.title') }}</h1>
                </div>

                <div class="header-right">
                    <button @click="validateDependencies" class="header-btn" :disabled="isCheckUpdatesDisabled">
                        <v-icon class="w-4 h-4" name="md-refresh-outlined"></v-icon>
                        <span class="header-btn-text">{{ $t('settings.dependency.validate') }}</span>
                    </button>

                    <button @click="checkUpdates" class="header-btn" :disabled="isCheckUpdatesDisabled">
                        <v-icon class="w-4 h-4" name="md-refresh-outlined"></v-icon>
                        <span class="header-btn-text">{{ $t('settings.dependency.check_updates') }}</span>
                    </button>
                </div>
            </div>
        </div>

        <!-- 主内容区域 -->
        <div class="flex-1 p-4 space-y-4 overflow-y-auto">
            <!-- 依赖卡片 -->
            <div v-for="(dep, key) in dependencies" :key="key" class="card-macos">
                <!-- 卡片头部 -->
                <div class="card-header">
                    <div class="flex items-center space-x-3">
                        <div class="w-3 h-3 rounded-full" :class="{
                            'bg-green-500': dep.available && !dep.needUpdate,
                            'bg-orange-400': dep.available && dep.needUpdate,
                            'bg-red-500': !dep.available
                        }"></div>
                        <h3 class="font-medium text-base-content">{{ dep.name }}</h3>
                        <span v-if="dep.available" class="text-xs px-2 py-1 rounded-full bg-green-100 text-green-700">
                            {{ $t('settings.dependency.installed') }}
                        </span>
                        <span v-else class="text-xs px-2 py-1 rounded-full bg-red-100 text-red-700">
                            {{ $t('settings.dependency.not_installed') }}
                        </span>
                    </div>

                    <div class="flex items-center space-x-2">
                        <!-- 操作按钮区域 -->
                        <template v-if="dep.available">
                            <button v-if="dep.needUpdate && !dep.updating && !dep.installing"
                                @click="showMirrorSelector(key, 'update')" class="btn-macos btn-primary" :disabled="isCheckUpdatesDisabled">
                                <v-icon class="w-4 h-4 mr-2" name="md-upgrade-outlined"></v-icon>
                                {{ $t('settings.dependency.update') }}
                            </button>

                            <button v-else-if="dep.updating" class="btn-macos" disabled>
                                <div
                                    class="w-4 h-4 mr-2 border-2 border-primary border-t-transparent rounded-full animate-spin">
                                </div>
                                {{ $t('settings.dependency.updating') }}
                            </button>

                            <!-- 如果正在安装，也显示安装状态 -->
                            <button v-else-if="dep.installing" class="btn-macos" disabled>
                                <div
                                    class="w-4 h-4 mr-2 border-2 border-primary border-t-transparent rounded-full animate-spin">
                                </div>
                                {{ $t('settings.dependency.installing') }}
                            </button>
                        </template>

                        <template v-else>
                            <button @click="showMirrorSelector(key, 'install')" :disabled="isCheckUpdatesDisabled"
                                class="btn-macos btn-primary">
                                <v-icon v-if="!dep.installing" class="w-4 h-4 mr-2"
                                    name="md-download-outlined"></v-icon>
                                <div v-else
                                    class="w-4 h-4 mr-2 border-2 border-white border-t-transparent rounded-full animate-spin">
                                </div>
                                {{ dep.installing ? $t('settings.dependency.installing') :
                                    $t('settings.dependency.install') }}
                            </button>
                        </template>
                    </div>
                </div>

                <!-- 卡片内容 -->
                <div v-if="dep.available" class="card-content">
                    <!-- 版本信息 -->
                    <div class="config-item">
                        <label class="item-label">{{ $t('settings.dependency.version') }}</label>
                        <div class="item-content">
                            <span class="item-value">{{ dep.version }}</span>
                            <span v-if="dep.needUpdate" class="item-action text-xs text-orange-600">
                                → {{ dep.latestVersion }}
                            </span>
                        </div>
                    </div>

                    <!-- 路径信息 -->
                    <div class="config-item">
                        <label class="item-label">{{ $t('settings.dependency.path') }}</label>
                        <div class="item-content">
                            <span class="item-value" :title="dep.path">{{ dep.path }}</span>
                            <button @click="openDirectory(dep.path)"
                                class="item-action btn btn-sm btn-ghost btn-square">
                                <v-icon class="w-4 h-4 text-base-content/60" name="oi-file-directory"></v-icon>
                            </button>
                        </div>
                    </div>
                </div>

                <!-- 进度显示区域 - 支持安装和更新 -->
                <div v-if="dep.installing || dep.updating" class="card-content border-t border-base-300">
                    <!-- 进度条 -->
                    <div class="space-y-2">
                        <div class="flex items-center justify-between text-sm">
                            <span class="text-base-content/70">{{ $t('settings.dependency.status.progress') }}</span>
                            <span class="text-base-content/70">
                                {{ dep.installing ? (dep.installProgress || $t('settings.dependency.installing')) :
                                    (dep.updateProgress || $t('settings.dependency.updating')) }}
                            </span>
                        </div>

                        <!-- 进度条 -->
                        <div class="w-full bg-base-200 rounded-full h-2">
                            <div class="bg-primary h-2 rounded-full transition-all duration-300"
                                :style="{ width: getProgressPercentage(dep) + '%' }"></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 镜像选择模态框 -->
        <div v-if="showMirrorModal" class="modal-overlay" @click="closeMirrorModal">
            <div class="modal-macos" @click.stop>
                <div class="modal-header">
                    <h3>{{ $t('settings.dependency.select_mirror') }}</h3>
                    <button @click="closeMirrorModal" class="btn-icon">
                        <v-icon class="w-4 h-4" name="md-close"></v-icon>
                    </button>
                </div>

                <div class="modal-content">
                    <div class="space-y-2">
                        <div v-for="mirror in availableMirrors" :key="mirror.name" @click="selectedMirror = mirror.name"
                            class="mirror-option" :class="{ 'selected': selectedMirror === mirror.name }">
                            <div class="flex items-center justify-between">
                                <div>
                                    <div class="font-medium">{{ mirror.name }}</div>
                                    <div class="text-sm text-base-content/60">{{ mirror.description }}</div>
                                </div>
                                <span v-if="mirror.recommended"
                                    class="text-xs px-2 py-1 rounded-full bg-primary/10 text-primary">
                                    {{ $t('settings.dependency.recommended') }}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="modal-footer">
                    <button @click="closeMirrorModal" class="btn-macos">
                        {{ $t('common.cancel') }}
                    </button>
                    <button @click="performAction" class="btn-macos btn-primary" :disabled="!selectedMirror">
                        {{ currentAction === 'install' ? $t('settings.dependency.install') :
                            $t('settings.dependency.update') }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { onMounted, ref, computed } from 'vue'
import { ListMirrors } from 'wailsjs/go/api/DependenciesAPI'
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n'
import useDependenciesStore from '@/stores/dependencies'

export default {
    name: 'Dependency',
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
                await dependenciesStore.checkUpdates()
                $message.success(t('settings.dependency.check_updates_success'))
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
                message.error(t('settings.dependency.load_mirrors_failed'))
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
                await OpenDirectory(path)
            } catch (error) {
                console.error('Failed to open directory:', error)
            }
        }

        const getProgressPercentage = (dep) => {
            // 根据当前状态返回对应的进度百分比
            if (dep.installing) {
                return dep.installProgressPercent || 0
            } else if (dep.updating) {
                return dep.updateProgressPercent || 0
            }
            return 0
        }

        // 组件挂载
        onMounted(async () => {
            await dependenciesStore.loadDependencies()
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
/* macOS 风格样式 */

/* 基础按钮样式 */
%btn-base {
    @apply px-4 py-2 text-sm font-medium rounded-lg border transition-colors duration-150;
    @apply focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary;

    &:disabled {
        @apply opacity-50 cursor-not-allowed;
    }
}

/* 基础头部样式 */
%header-base {
    @apply px-4 py-3 border-base-300 flex items-center justify-between;
}

.btn-macos {
    @extend %btn-base;
    @apply border-base-300 bg-base-100 text-base-content;
    @apply hover:bg-base-200 active:bg-base-300;

    &.btn-primary {
        @apply bg-primary text-primary-content border-primary;
        @apply hover:bg-primary/90 active:bg-primary/80;
    }
}

.btn-icon {
    @apply p-2 rounded-lg border-0 bg-transparent text-base-content/60;
    @apply hover:bg-base-200 hover:text-base-content transition-colors duration-150;
}

.card-macos {
    @apply bg-base-100 border border-base-300 rounded-xl shadow-sm;

    .card-header {
        @extend %header-base;
        @apply border-b;
    }

    .card-content {
        @apply p-4 space-y-4;
    }
}

.header-bar {
    @apply bg-base-100/80 backdrop-blur-sm border-b border-base-200;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.header-content {
    @apply px-6 py-4 flex items-center justify-between;
}

.header-left {
    @apply flex items-center space-x-3;
}

.header-icon {
    @apply w-8 h-8 rounded-lg bg-base-200/50 flex items-center justify-center;
}

.header-title {
    @apply text-lg font-semibold text-base-content tracking-tight;
}

.header-right {
    @apply flex items-center space-x-3;
}

.header-btn {
    @extend %btn-base;
    @apply border-base-300/50 bg-base-100 flex items-center space-x-2;
    @apply hover:bg-base-200 hover:border-base-300 active:bg-base-300;
    @apply transition-all duration-150 ease-out;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);

    &:disabled {
        @apply hover:bg-base-100 hover:border-base-300/50;
    }
}

.header-btn-text {
    @apply hidden sm:inline;
}

.config-item {
    @apply flex items-center py-2 px-1 border-b border-base-200/50 last:border-b-0;
    min-height: 36px;

    .item-label {
        @apply text-sm font-medium text-base-content/80;
        width: 80px;
        flex-shrink: 0;
        text-align: left;
        margin-right: 16px;
    }

    .item-content {
        @apply flex items-center flex-1 min-w-0 justify-end;
        min-height: 24px;

        .item-value {
            @apply text-sm text-base-content truncate;
            text-align: right;
            line-height: 24px;
        }

        .item-action {
            @apply ml-2 flex-shrink-0;

            &.btn {
                min-height: 24px;
                height: 24px;
                padding: 2px;
            }
        }
    }
}

.modal-overlay {
    @apply fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50;
}

.modal-macos {
    @apply bg-base-100 rounded-xl shadow-xl border border-base-300 w-full max-w-md mx-4;

    .modal-header {
        @extend %header-base;
        @apply border-b;
    }

    .modal-content {
        @apply p-4 max-h-96 overflow-y-auto;
    }

    .modal-footer {
        @extend %header-base;
        @apply border-t justify-end space-x-2;
    }
}

.mirror-option {
    @apply p-3 rounded-lg border border-base-300 cursor-pointer transition-colors duration-150;
    @apply hover:bg-base-200;

    &.selected {
        @apply border-primary bg-primary/5;
    }
}
</style>