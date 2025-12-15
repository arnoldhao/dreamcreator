import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ListSubtitles } from 'bindings/dreamcreator/backend/api/subtitlesapi'

export const useSubtitleStore = defineStore('subtitle', () => {
    // 状态
    const projects = ref([])
    const currentProject = ref(null)
    const currentLanguage = ref(null)
    const isLoading = ref(false)
    const error = ref(null)
    const lastRefreshTime = ref(0)
    // Pending project to open when Subtitle view mounts
    const pendingOpenProjectId = ref(null)

    // Actions
    const fetchProjects = async (options = {}) => {
        const { force = false, showLoading = true } = options

        // 防抖：如果距离上次刷新不到 1 秒且不是强制刷新，则跳过
        const now = Date.now()
        if (!force && now - lastRefreshTime.value < 1000) {
            return projects.value
        }

        try {
            if (showLoading) isLoading.value = true
            error.value = null

            const response = await ListSubtitles()
            if (response.success) {
                const data = response.data
                const projectsData = Array.isArray(data) ? data : JSON.parse(data || '[]')
                projects.value = projectsData
                lastRefreshTime.value = now

                // 同步更新当前项目数据
                if (currentProject.value) {
                    const updatedCurrentProject = projectsData.find(p => p.id === currentProject.value.id)
                    if (updatedCurrentProject) {
                        // 找到对应项目，更新为最新数据
                        currentProject.value = updatedCurrentProject
                    } else {
                        // 项目不存在，清空当前项目
                        currentProject.value = null
                        currentLanguage.value = null
                    }
                }

                return projectsData
            } else {
                throw new Error(response.msg)
            }
        } catch (err) {
            error.value = err.message
            throw err
        } finally {
            if (showLoading) isLoading.value = false
        }
    }

    const setCurrentProject = (project) => {
        currentProject.value = project
        if (!project) {
            currentLanguage.value = null
        }
    }

    const setPendingOpenProjectId = (id) => { pendingOpenProjectId.value = id }

    const updateProject = (updatedProject) => {
        const index = projects.value.findIndex(p => p.id === updatedProject.id)
        if (index !== -1) {
            projects.value[index] = updatedProject
            if (currentProject.value?.id === updatedProject.id) {
                currentProject.value = updatedProject
            }
        }
    }

    const refreshProjects = async () => {
        return await fetchProjects({ force: true })
    }

    return {
        // 状态
        projects,
        currentProject,
        currentLanguage,
        isLoading,
        error,
        pendingOpenProjectId,

        // Actions
        fetchProjects,
        setCurrentProject,
        updateProject,
        refreshProjects,
        setPendingOpenProjectId
    }
})
