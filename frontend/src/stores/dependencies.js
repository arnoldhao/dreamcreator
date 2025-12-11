import { defineStore } from 'pinia'
import { ListDependencies, InstallDependencyWithMirror, UpdateDependencyWithMirror, CheckUpdates, ListMirrors, ValidateDependencies, RepairDependency, QuickValidateDependencies } from 'wailsjs/go/api/DependenciesAPI'
import { useDtStore } from '@/stores/downloadTasks'
import { i18nGlobal } from '@/utils/i18n.js'
import WebSocketService from '@/services/websocket'

const useDependenciesStore = defineStore('dependencies', {
        state: () => ({
            dependencies: {
                'yt-dlp': {
                    // frontend-only properties
                    installing: false, // 是否在安装
                    installProgress: '', // 安装进度
                    installProgressPercent: 0, // 安装进度百分比
                    currentAction: '', // 'install' | 'update' | ''
                    installed: false, // 是否安装
                // backend properties
                type: 'yt-dlp',
                name: 'YT-DLP', // 添加显示名称
                available: false,
                path: '',
                execPath: '',
                version: '',
                latestVersion: '',
                needUpdate: false,
                lastCheckAttempted: false,
                lastCheckSuccess: true,
                lastCheckError: '',
                lastCheckErrorCode: '',
            },
                'deno': {
                    // frontend-only properties
                    installing: false,
                    installProgress: '',
                    installProgressPercent: 0,
                    currentAction: '',
                    installed: false,
                // backend properties
                type: 'deno',
                name: 'Deno',
                available: false,
                path: '',
                execPath: '',
                version: '',
                latestVersion: '',
                needUpdate: false,
                lastCheckAttempted: false,
                lastCheckSuccess: true,
                lastCheckError: '',
                lastCheckErrorCode: '',
            },
                'ffmpeg': {
                    // frontend-only properties
                    installing: false,
                    installProgress: '',
                    installProgressPercent: 0,
                    currentAction: '',
                    installed: false,
                // backend properties
                type: 'ffmpeg',
                name: 'FFmpeg', // 添加显示名称
                available: false,
                path: '',
                execPath: '',
                version: '',
                latestVersion: '',
                needUpdate: false,
                lastCheckAttempted: false,
                lastCheckSuccess: true,
                lastCheckError: '',
                lastCheckErrorCode: '',
            }
        },
        mirrors: {},
        loading: false,
        validating: false,
        _wsListenersSetup: false,
        _dependencyProgressHandler: null, // 保存回调引用用于清理
    }),

    getters: {
        getDependency: (state) => (type) => {
            return state.dependencies[type] || null
        },

        isInstalled: (state) => (type) => {
            return state.dependencies[type]?.available || false
        },

        needsUpdate: (state) => (type) => {
            return state.dependencies[type]?.needUpdate || false
        },
        allowCheckUpdates: (state) => {
            // 如果正在加载，不允许检查更新
            if (state.loading) return false

            // 如果正在验证，不允许检查更新
            if (state.validating) return false

            // 如果任何依赖正在安装或更新，不允许检查更新
            return !Object.values(state.dependencies).some(dep => dep?.installing)
        },
    },

    actions: {
        t(key, params = {}) {
            return i18nGlobal.t(key, params)
        },

        // 重命名并简化WebSocket监听器设置
        setupWebSocketListeners() {
            if (this._wsListenersSetup) return;

            const dtStore = useDtStore();

            // 创建回调函数并保存引用
            this._dependencyProgressHandler = this.createDependencyProgressHandler();
            dtStore.registerInstallingCallback(this._dependencyProgressHandler);

            this._wsListenersSetup = true;
        },

        // 提取回调处理逻辑
        createDependencyProgressHandler() {
            return async (data) => {
                const formatPercentage = (percentage) => {
                    // 防止 null/undefined，默认为 0
                    const safePercentage = percentage ?? 0;
        
                    // 只按照百分比形式，直接保留2位小数:0.01 - 100.00
                    return Math.round(safePercentage * 100) / 100;
                };
                if (data && data.type) {
                    const depType = data.type.toLowerCase()
                    if (this.dependencies[depType]) {
                        // 更新安装进度
                        switch (data.stage) {
                            case 'preparing':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.preparing')
                                break
                            case 'downloading':
                                const percent = formatPercentage(data.percentage || 0);
                                let progressText = this.t('settings.dependency.status.downloading', { percentage: percent });
                                this.dependencies[depType].installProgress = progressText
                                this.dependencies[depType].installProgressPercent = percent
                                break
                            case 'extracting':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.extracting')
                                this.dependencies[depType].installProgressPercent = formatPercentage(data.percentage || 0);
                                break
                            case 'validating':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.validating')
                                this.dependencies[depType].installProgressPercent = formatPercentage(data.percentage || 0);
                                break
                            case 'cleaning':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.cleaning')
                                this.dependencies[depType].installProgressPercent = formatPercentage(data.percentage || 0);
                                break
                            case 'installCancelled':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.installCancelled')
                                // refresh
                                this.dependencies[depType].installing = false
                                this.dependencies[depType].currentAction = ''
                                await this.loadDependencies()
                                break
                            case 'installFailed':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.installFailed')
                                // refresh
                                this.dependencies[depType].installing = false
                                this.dependencies[depType].currentAction = ''
                                await this.loadDependencies()
                                break
                            case 'installCompleted':
                                this.dependencies[depType].installProgress = this.t('settings.dependency.status.installCompleted')
                                // refresh
                                this.dependencies[depType].installing = false
                                // toast success based on action
                                const action = this.dependencies[depType].currentAction
                                if (action === 'update') {
                                    $message.success(this.t('settings.dependency.update_success'))
                                } else {
                                    $message.success(this.t('settings.dependency.install_success'))
                                }
                                this.dependencies[depType].currentAction = ''
                                await this.loadDependencies()
                                break
                            default:
                                $message.warn(`Unknown dependency installation stage: ${data.stage}`)
                                break
                        }
                    }
                } else {
                    $message.warn(`Unknown dependency type: ${data.type}`)
                }
            };
        },

        // 添加清理方法
        cleanup() {
            if (this._dependencyProgressHandler) {
                const dtStore = useDtStore();
                dtStore.unregisterInstallingCallback(this._dependencyProgressHandler);
                this._dependencyProgressHandler = null;
                this._wsListenersSetup = false;
            }
        },

        async installDependency(type, version, mirror) {
            await WebSocketService.ensureConnected()

            this.dependencies[type].installing = true;
            this.dependencies[type].installProgress = this.t('settings.dependency.installing');
            this.dependencies[type].currentAction = 'install';

            try {
                const response = await InstallDependencyWithMirror(type, version, mirror);
                if (!response.success) {
                    throw new Error(response.msg);
                }
            } catch (error) {
                this.dependencies[type].installing = false;
                this.dependencies[type].installProgress = this.t('settings.dependency.status.installFailed');
                $dialog.error({
                    title: this.t('settings.dependency.install_failed'),
                    content: error.message,
                });
                throw error;
            }
        },

        async updateDependency(type, mirror) {
            await WebSocketService.ensureConnected()

            this.dependencies[type].installing = true
            this.dependencies[type].installProgress = this.t('settings.dependency.updating')
            this.dependencies[type].currentAction = 'update'

            try {
                const response = await UpdateDependencyWithMirror(type, mirror)
                if (!response.success) {
                    throw new Error(response.msg);
                }
            } catch (error) {
                this.dependencies[type].installing = false
                this.dependencies[type].installProgress = this.t('settings.dependency.status.updateFailed')
                // dialog
                $dialog.error({
                    title: this.t('settings.dependency.update_failed'),
                    content: error.message,
                })
            }
        },

        async repairDependency(type) {
            await WebSocketService.ensureConnected()

            this.dependencies[type].installing = true
            try {
                const resp = await RepairDependency(type)
                if (resp.success) {
                    $message.success(this.t('settings.dependency.repair_success'))
                } else {
                    $message.error(this.t('settings.dependency.repair_failed'))
                }
            } catch (error) {
                $message.error(this.t('settings.dependency.repair_failed'))
            } finally {
                this.dependencies[type].installing = false
                await this.loadDependencies()
            }
        },

        async loadDependencies() {
            this.loading = true
            try {
                const response = await ListDependencies()
                if (response.success && response.data) {
                    const parsedData = JSON.parse(response.data)
                    // 更新依赖状态
                    Object.entries(parsedData).forEach(([type, info]) => {
                        if (this.dependencies[type]) {
                            Object.assign(this.dependencies[type], {
                                available: info.available,
                                path: info.path,
                                execPath: info.execPath,
                                version: info.version,
                                latestVersion: info.latestVersion,
                                needUpdate: info.needUpdate,
                                lastCheckAttempted: !!info.lastCheckAttempted,
                                lastCheckSuccess: !!info.lastCheckSuccess,
                                lastCheckError: info.lastCheckError || '',
                                lastCheckErrorCode: info.lastCheckErrorCode || '',
                                installed: info.available
                            })
                        }
                    })
                } else {
                    throw new Error('Failed to load dependencies:', response.msg)
                }
            } catch (error) {
                $message.error(error.message)
            } finally {
                this.loading = false
            }
        },

        async checkUpdates() {
            this.loading = true
            try {
                const response = await CheckUpdates()
                if (response.success && response.data) {
                    const parsedData = JSON.parse(response.data)
                    Object.entries(parsedData).forEach(([type, info]) => {
                        if (this.dependencies[type]) {
                            Object.assign(this.dependencies[type], {
                                latestVersion: info.latestVersion,
                                needUpdate: info.needUpdate,
                                lastCheckAttempted: !!info.lastCheckAttempted,
                                lastCheckSuccess: !!info.lastCheckSuccess,
                                lastCheckError: info.lastCheckError || '',
                                lastCheckErrorCode: info.lastCheckErrorCode || '',
                            })
                        }
                    })
                    const hasFailures = Object.values(this.dependencies).some(dep => dep.lastCheckAttempted && !dep.lastCheckSuccess)
                    return { hasFailures }
                } else {
                    throw new Error('Failed to check updates:', response.msg)
                }
            } catch (error) {
                $dialog.error({
                    title: this.t('settings.dependency.check_updates_failed'),
                    content: error.message,
                })
                return { hasFailures: true }
            } finally {
                this.loading = false
            }
        },

        async validateDependencies() {
            this.validating = true
            try {
                const response = await ValidateDependencies()
                if (response.success) {
                    $message.success(this.t('settings.dependency.validate_success'))
                    return true
                } else {
                    throw new Error('Failed to validate dependencies:' + response.msg)
                }
            } catch (error) {
                $dialog.error({
                    title: this.t('settings.dependency.validate_failed'),
                    content: error.message,
                })
                return false
            } finally {
                await this.loadDependencies()
                this.validating = false
            }
        },

        async quickValidateDependencies(showToast = false) {
            try {
                const response = await QuickValidateDependencies()
                if (!response.success) throw new Error(response.msg || 'Quick validate failed')
                // merge into store
                const parsed = JSON.parse(response.data || '{}')
                Object.entries(parsed).forEach(([type, info]) => {
                    if (this.dependencies[type]) {
                        Object.assign(this.dependencies[type], {
                            available: info.available,
                            path: info.path,
                            execPath: info.execPath,
                            version: info.version,
                            latestVersion: info.latestVersion,
                            needUpdate: info.needUpdate,
                            lastCheckAttempted: !!info.lastCheckAttempted,
                            lastCheckSuccess: !!info.lastCheckSuccess,
                            lastCheckError: info.lastCheckError || '',
                            lastCheckErrorCode: info.lastCheckErrorCode || '',
                            installed: info.available
                        })
                    }
                })
                if (showToast) {
                    $message.success(this.t('settings.dependency.quick_validate_success'))
                }
                return true
            } catch (e) {
                $message.error(e.message || 'Quick validate failed')
                return false
            }
        },

        async loadMirrors(type) {
            try {
                const response = await ListMirrors(type)
                if (response.success && response.data) {
                    this.mirrors[type] = response.data
                } else {
                    throw new Error('Failed to load mirrors:', response.msg)
                }
            } catch (error) {
                $message.error(error.message)
            }
        },
    }
})

export default useDependenciesStore
