import { defineStore } from 'pinia'
import { ListDependencies, InstallDependencyWithMirror, UpdateDependencyWithMirror, CheckUpdates, ListMirrors, ValidateDependencies } from 'wailsjs/go/api/DependenciesAPI'
import { useDtStore } from '@/handlers/downtasks'
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
                updating: false,        // 更新状态
                updateProgress: '',     // 更新进度
                updateProgressPercent: 0, // 更新进度百分比
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
            },
            'ffmpeg': {
                // frontend-only properties
                installing: false,
                installProgress: '',
                installProgressPercent: 0,
                updating: false,        // 更新状态
                updateProgress: '',     // 添加更新进度
                updateProgressPercent: 0, // 更新进度百分比
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
            }
        },
        mirrors: {},
        loading: false,
        validating: false,
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
            return !Object.values(state.dependencies).some(dep => dep?.installing || dep?.updating)
        },
    },

    actions: {
        t(key, params = {}) {
            return i18nGlobal.t(key, params)
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
                                needUpdate: info.needUpdate
                            })
                        }
                    })
                } else {
                    throw new Error('Failed to check updates:', response.msg)
                }
            } catch (error) {
                $dialog.error({
                    title: this.t('settings.dependency.check_updates_failed'),
                    content: error.message,
                })
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
                    throw new Error('Failed to validate dependencies:', response.msg)
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

        // 初始化WebSocket监听
        async initWebSocket() {
            if (this._wsInitialized) return

            try {
                const dtStore = useDtStore()
                // 检查 WebSocket 连接状态
                if (!WebSocketService.client || WebSocketService.client.readyState !== WebSocket.OPEN) {
                    await WebSocketService.connect()
                }

                // 格式化百分比
                const formatPercentage = (percentage) => {
                    // 防止 null/undefined，默认为 0
                    const safePercentage = percentage ?? 0;

                    // 只按照百分比形式，直接保留2位小数:0.01 - 100.00
                    return Math.round(safePercentage * 100) / 100;
                }

                // 注册依赖安装进度回调
                const handleDependencyProgress = async (data) => {
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
                                case 'updating':
                                    const updatePercent = formatPercentage(data.percentage || 0);
                                    let updateProgressText = this.t('settings.dependency.status.downloading', { percentage: updatePercent });
                                    this.dependencies[depType].updateProgress = updateProgressText
                                    this.dependencies[depType].updateProgressPercent = updatePercent
                                    break
                                // final status 
                                case 'updateFailed':
                                    this.dependencies[depType].updateProgress = this.t('settings.dependency.status.updateFailed')
                                    // refresh
                                    this.dependencies[depType].updating = false
                                    await this.loadDependencies()
                                    break
                                case 'updateCompleted':
                                    this.dependencies[depType].updateProgress = this.t('settings.dependency.status.updateCompleted')
                                    // refresh
                                    this.dependencies[depType].updating = false
                                    await this.loadDependencies()
                                    break
                                case 'installCancelled':
                                    this.dependencies[depType].installProgress = this.t('settings.dependency.status.installCancelled')
                                    // refresh
                                    this.dependencies[depType].installing = false
                                    await this.loadDependencies()
                                    break
                                case 'updateCancelled':
                                    this.dependencies[depType].updateProgress = this.t('settings.dependency.status.updateCancelled')
                                    // refresh
                                    this.dependencies[depType].updating = false
                                    await this.loadDependencies()
                                    break
                                case 'installFailed':
                                    this.dependencies[depType].installProgress = this.t('settings.dependency.status.installFailed')
                                    // refresh
                                    this.dependencies[depType].installing = false
                                    await this.loadDependencies()
                                    break
                                case 'installCompleted':
                                    this.dependencies[depType].installProgress = this.t('settings.dependency.status.installCompleted')
                                    // refresh
                                    this.dependencies[depType].installing = false
                                    await this.loadDependencies()
                                    break
                                default:
                                    $message.warn(`Unknown dependency installation stage: ${data.stage}`)
                                    break
                            }
                        }
                    }
                }

                dtStore.registerInstallingCallback(handleDependencyProgress)

                this._wsInitialized = true
            } catch (error) {
                console.error('Failed to initialize WebSocket:', error)
                throw error
            }
        },

        async installDependency(type, version, mirror) {
            // 确保WebSocket已初始化
            await this.initWebSocket()

            this.dependencies[type].installing = true
            this.dependencies[type].installProgress = this.t('settings.dependency.installing')

            try {
                const response = await InstallDependencyWithMirror(type, version, mirror)
                if (response.success) {
                    // 不立即重新加载，等待WebSocket事件更新
                } else {
                    this.dependencies[type].installing = false
                    this.dependencies[type].installProgress = this.t('settings.dependency.status.installFailed')
                    throw new Error(response.msg)
                }
            } catch (error) {
                this.dependencies[type].installing = false
                this.dependencies[type].installProgress = this.t('settings.dependency.status.installFailed')
                // dialog
                $dialog.error({
                    title: this.t('settings.dependency.install_failed'),
                    content: error.message,
                })
            }
        },

        async updateDependency(type, mirror) {
            await this.initWebSocket()
            this.dependencies[type].updating = true
            this.dependencies[type].updateProgress = this.t('settings.dependency.updating')

            try {
                const response = await UpdateDependencyWithMirror(type, mirror)
                if (response.success) {
                    // 不立即重新加载，等待WebSocket事件更新
                } else {
                    this.dependencies[type].updating = false
                    this.dependencies[type].updateProgress = this.t('settings.dependency.status.updateFailed')
                    throw new Error(response.msg)
                }
            } catch (error) {
                this.dependencies[type].updating = false
                this.dependencies[type].updateProgress = this.t('settings.dependency.status.updateFailed')
                // dialog
                $dialog.error({
                    title: this.t('settings.dependency.update_failed'),
                    content: error.message,
                })
            }
        }
    }
})

export default useDependenciesStore