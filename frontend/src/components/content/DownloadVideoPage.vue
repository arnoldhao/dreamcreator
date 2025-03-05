<template>
    <div class="flex-1 h-full">
        <div class="download-container rounded-tl-lg h-full w-full">
            <div class="h-full w-full bg-base-200">
                <div class="h-full p-6 overflow-y-auto">
                    <!-- 页面标题 -->
                    <div class="flex justify-between items-center mb-6">
                        <div class="flex items-center">
                            <v-icon name="ri-download-cloud-line" class="w-6 h-6 mr-3 text-primary"></v-icon>
                            <span class="text-xl font-bold">{{ t('video_download.page_title') }}</span>
                        </div>
                        <div class="space-x-3">
                            <button class="btn btn-sm btn-primary btn-outline" @click="refreshInstantData">
                                <v-icon name="hi-refresh" class="h-4 w-4 mr-1"></v-icon>
                                {{ t('video_download.refresh') }}
                            </button>
                        </div>
                    </div>

                    <!-- 主要内容区域 -->
                    <div class="space-y-6 mb-6">
                        <!-- 下载任务卡片 -->
                        <div class="card bg-base-100 shadow-md overflow-hidden">
                            <div class="overflow-x-auto">
                                <table class="table table-zebra w-full">
                                    <!-- 表头 -->
                                    <thead>
                                        <tr class="border-b border-base-300">
                                            <th
                                                class="bg-primary/10 text-left font-bold rounded-tl-2xl rounded-tr-2xl border-r border-base-300/50 px-4">
                                                {{ t('video_download.new_task') }}
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr>
                                            <td class="p-0">
                                                <div class="p-4">
                                                    <!-- URL输入区域 -->
                                                    <div class="flex items-center gap-6">
                                                        <div class="flex-1">
                                                            <div class="flex gap-2">
                                                                <input type="text" v-model="url"
                                                                    :placeholder="t('video_download.url_placeholder')"
                                                                    class="input input-bordered flex-1" />
                                                                <button @click="handleGet"
                                                                    class="btn btn-primary w-[140px]"
                                                                    :disabled="isLoading">
                                                                    <div class="flex items-center justify-center">
                                                                        <v-icon v-if="isLoading" name="ri-loader-2-line"
                                                                            class="animate-spin h-4 w-4 mr-1"></v-icon>
                                                                        <span>{{ isLoading ? t('video_download.parsing')
                                                                            :
                                                                            t('video_download.parse') }}</span>
                                                                    </div>
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>

                                                    <!-- 解析结果区域 -->
                                                    <template v-if="videoData?.info?.title">
                                                        <div class="divider"></div>
                                                        <!-- 视频标题和预览信息 -->
                                                        <div class="flex items-center gap-6 mb-4">
                                                            <div class="flex-1">
                                                                <div class="flex items-center gap-2">
                                                                    <v-icon :name="sourceIcon(videoData.info.source)"
                                                                        class="w-4 h-4 flex-shrink-0" />
                                                                    <div class="text-base">{{ videoData.info.title }}
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>

                                                        <!-- 下载选项 -->
                                                        <div class="flex items-center gap-6">
                                                            <!-- 格式选择 -->
                                                            <div class="flex-1">
                                                                <div class="text-xs text-base-content/70 mb-1">{{
                                                                    t('video_download.format') }}</div>
                                                                <select v-model="selectedFormat"
                                                                    class="select select-bordered w-full">
                                                                    <option disabled value="">{{
                                                                        t('video_download.select_format') }}</option>
                                                                    <option v-for="format in uniqueFormats"
                                                                        :key="format" :value="format">
                                                                        {{ format }}
                                                                    </option>
                                                                </select>
                                                            </div>

                                                            <!-- 质量选择 -->
                                                            <div class="flex-1">
                                                                <div class="text-xs text-base-content/70 mb-1">{{
                                                                    t('video_download.quality') }}</div>
                                                                <select v-model="selectedQuality"
                                                                    class="select select-bordered w-full">
                                                                    <option disabled value="">{{
                                                                        t('video_download.select_quality') }}</option>
                                                                    <option v-for="quality in filteredQualities"
                                                                        :key="quality.id" :value="quality">
                                                                        {{ formatQuality(quality.quality) }} ({{
                                                                            (quality.size / 1024 /
                                                                                1024).toFixed(2) }}MB)
                                                                    </option>
                                                                </select>
                                                            </div>

                                                            <!-- 字幕选择 -->
                                                            <div class="flex-1">
                                                                <div class="text-xs text-base-content/70 mb-1">{{
                                                                    t('video_download.caption') }}</div>
                                                                <select v-model="selectedCaption"
                                                                    class="select select-bordered w-full">
                                                                    <option disabled value="">{{
                                                                        t('video_download.select_caption') }}</option>
                                                                    <option v-for="caption in videoData?.captions || []"
                                                                        :key="caption.id" :value="caption">
                                                                        {{ caption.language }}
                                                                    </option>
                                                                </select>
                                                            </div>

                                                            <!-- 下载按钮 -->
                                                            <div>
                                                                <div class="text-xs text-base-content/70 mb-1">&nbsp;
                                                                </div>
                                                                <button @click="download"
                                                                    class="btn btn-primary w-[140px]"
                                                                    :disabled="requestDownloading || !selectedQuality">
                                                                    <div class="flex items-center justify-center">
                                                                        <v-icon v-if="requestDownloading"
                                                                            name="ri-loader-2-line"
                                                                            class="animate-spin h-4 w-4 mr-1"></v-icon>
                                                                        <span>{{ requestDownloading ?
                                                                            t('video_download.downloading') :
                                                                            t('video_download.download') }}</span>
                                                                    </div>
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </template>
                                                </div>
                                            </td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>

                        <!-- 下载任务列表 -->
                        <div class="card bg-base-100 shadow-md overflow-hidden">
                            <div class="overflow-x-auto">
                                <table class="table table-zebra w-full">
                                    <!-- 表头 -->
                                    <thead>
                                        <tr class="border-b border-base-300">
                                            <th
                                                class="bg-primary/10 min-w-[360px] text-left font-bold rounded-tl-2xl border-r border-base-300/50 px-4">
                                                {{ t('video_download.title') }}</th>
                                            <th
                                                class="bg-primary/10 w-32 text-center font-bold border-r border-base-300/50">
                                                {{
                                                    t('video_download.format') }}</th>
                                            <th
                                                class="bg-primary/10 w-32 text-center font-bold border-r border-base-300/50">
                                                {{
                                                    t('video_download.size') }}</th>
                                            <th
                                                class="bg-primary/10 w-24 text-center font-bold border-r border-base-300/50">
                                                {{
                                                    t('video_download.status') }}</th>
                                            <th class="bg-primary/10 w-32 text-center font-bold rounded-tr-2xl">{{
                                                t('video_download.action') }}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <template v-if="instantData && instantData.length > 0">
                                            <template v-for="item in instantData" :key="item.taskId">
                                                <tr class="hover" @click.stop="toggleExpand(item.taskId)">
                                                    <!-- 标题 -->
                                                    <td class="max-w-[240px]">
                                                        <div class="flex items-center gap-2">
                                                            <v-icon :name="sourceIcon(item.source)"
                                                                class="w-4 h-4 flex-shrink-0" />
                                                            <div class="truncate text-sm" :title="item.title">{{
                                                                item.title }}
                                                            </div>
                                                        </div>
                                                    </td>
                                                    <!-- 格式 -->
                                                    <td class="text-center">
                                                        <div class="badge badge-ghost">{{ item.streams?.[0]?.ext ||
                                                            'N/A' }}
                                                        </div>
                                                    </td>
                                                    <!-- 大小 -->
                                                    <td class="text-center">{{ formatFileSize(item.totalSize) }}</td>
                                                    <!-- 状态 -->
                                                    <td class="text-center">
                                                        <button class="w-4 h-4 cursor-pointer tooltip"
                                                            :data-tip="t('video_download.view_status')"
                                                            @click.stop="showStatusDetail(item)">
                                                            <v-icon
                                                                :name="statusIcon[item.taskStatus] || 'ri-file-unknow-line'"
                                                                class="w-4 h-4" />
                                                        </button>
                                                    </td>
                                                    <!-- 操作 -->
                                                    <td>
                                                        <div class="flex gap-1 justify-end">
                                                            <button class="btn btn-ghost btn-xs tooltip"
                                                                :data-tip="t('video_download.detail')"
                                                                @click.stop="showDetail(item)">
                                                                <v-icon name="ri-information-line" class="w-4 h-4" />
                                                            </button>
                                                            <button class="btn btn-ghost btn-xs tooltip"
                                                                :data-tip="t('video_download.open_folder')"
                                                                @click.stop="openFolder(item.savedPath)">
                                                                <v-icon name="ri-folder-open-line" class="w-4 h-4" />
                                                            </button>
                                                            <button class="btn btn-ghost btn-xs tooltip text-error"
                                                                :data-tip="t('video_download.delete')"
                                                                @click.stop="deleteRecord(item.taskId)">
                                                                <v-icon name="ri-delete-bin-line" class="w-4 h-4" />
                                                            </button>
                                                        </div>
                                                    </td>
                                                </tr>
                                                <!-- 展开的进度详情行 -->
                                                <tr v-if="expandedState.has(item.taskId)" class="bg-base-200/50">
                                                    <td colspan="5" class="p-3 animate-fadeIn">
                                                        <div class="card bg-base-100 shadow-sm">
                                                            <div class="card-body p-4">
                                                                <!-- 下载中状态 -->
                                                                <div v-if="item.isProcessing"
                                                                    class="flex items-center gap-6">
                                                                    <!-- Progress Item -->
                                                                    <div class="flex-1 flex items-center gap-4 px-4">
                                                                        <div class="flex-1">
                                                                            <div
                                                                                class="text-xs text-base-content/70 mb-1">
                                                                                {{
                                                                                    t('video_download.progress') }}
                                                                            </div>
                                                                            <div class="flex items-center gap-3">
                                                                                <div class="flex-1">
                                                                                    <progress
                                                                                        class="progress progress-primary w-full h-2"
                                                                                        :value="item.progress"
                                                                                        max="100">
                                                                                    </progress>
                                                                                </div>
                                                                                <div
                                                                                    class="text-sm font-medium min-w-[48px] text-right">
                                                                                    {{ Number(item.progress).toFixed(2)
                                                                                    }}%
                                                                                </div>
                                                                            </div>
                                                                        </div>
                                                                    </div>

                                                                    <!-- Divider -->
                                                                    <div class="w-px h-8 bg-base-300"></div>

                                                                    <!-- Speed Item -->
                                                                    <div class="px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.speed') }}</div>
                                                                        <div class="flex items-center gap-2">
                                                                            <v-icon name="md-clouddownload"
                                                                                class="w-4 h-4 text-primary" />
                                                                            <span class="text-sm font-medium">{{
                                                                                item.speedString
                                                                                }}</span>
                                                                        </div>
                                                                    </div>

                                                                    <!-- Divider -->
                                                                    <div class="w-px h-8 bg-base-300"></div>

                                                                    <!-- ETA Item -->
                                                                    <div class="px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.eta') }}</div>
                                                                        <div class="flex items-center gap-2">
                                                                            <v-icon name="io-timer-outline"
                                                                                class="w-4 h-4 text-primary" />
                                                                            <span class="text-sm font-medium">{{
                                                                                item.timeRemaining
                                                                                }}</span>
                                                                        </div>
                                                                    </div>
                                                                </div>

                                                                <!-- 非下载中状态 -->
                                                                <div v-else class="flex items-center gap-6">
                                                                    <!-- URL with copy button -->
                                                                    <div class="flex-1 px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.url') }}</div>
                                                                        <div class="flex items-center gap-2">
                                                                            <div class="text-sm truncate flex-1">{{
                                                                                item.url }}
                                                                            </div>
                                                                            <button class="btn btn-ghost btn-xs tooltip"
                                                                                :data-tip="t('video_download.copy_url')"
                                                                                @click.stop="copyText(item.url, 'url')">
                                                                                <v-icon name="md-contentcopy"
                                                                                    class="w-3.5 h-3.5" />
                                                                            </button>
                                                                        </div>
                                                                    </div>

                                                                    <!-- Divider -->
                                                                    <div class="w-px h-8 bg-base-300"></div>

                                                                    <!-- Quality -->
                                                                    <div class="w-48 px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.quality') }}</div>
                                                                        <div class="text-sm font-medium truncate max-w-[200px]"
                                                                            :title="item.streams?.[0]?.quality">
                                                                            {{ item.streams?.[0]?.quality || 'N/A' }}
                                                                        </div>
                                                                    </div>

                                                                    <!-- Divider -->
                                                                    <div class="w-px h-8 bg-base-300"></div>

                                                                    <!-- Average Speed -->
                                                                    <div class="w-36 px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.average_speed') }}
                                                                        </div>
                                                                        <div class="flex items-center gap-2">
                                                                            <v-icon name="md-speed"
                                                                                class="w-3.5 h-3.5 text-primary" />
                                                                            <span class="text-sm font-medium">{{
                                                                                item.averageSpeed || 'N/A'
                                                                            }}</span>
                                                                        </div>
                                                                    </div>

                                                                    <!-- Divider -->
                                                                    <div class="w-px h-8 bg-base-300"></div>

                                                                    <!-- Duration -->
                                                                    <div class="w-32 px-4">
                                                                        <div class="text-xs text-base-content/70 mb-1">
                                                                            {{
                                                                                t('video_download.duration') }}
                                                                        </div>
                                                                        <div class="flex items-center gap-2">
                                                                            <v-icon name="io-timer-outline"
                                                                                class="w-3.5 h-3.5 text-primary" />
                                                                            <span class="text-sm font-medium">{{
                                                                                formatDuration(item.durationSeconds)
                                                                            }}</span>
                                                                        </div>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </td>
                                                </tr>
                                            </template>
                                        </template>
                                        <template v-else>
                                            <tr>
                                                <td colspan="5" class="text-center py-8 text-base-content/50">
                                                    <div class="flex flex-col items-center gap-2">
                                                        <v-icon name="bi-inbox" class="w-8 h-8" />
                                                        {{ t('video_download.no_data') }}
                                                    </div>
                                                </td>
                                            </tr>
                                        </template>
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 详细信息模态框 -->
        <input type="checkbox" id="modal-detail" class="modal-toggle" v-model="detailModal" />
        <div class="modal" @click.self="detailModal = false">
            <div class="modal-box max-w-3xl bg-base-200 py-4">
                <div class="space-y-4">
                    <!-- 基本信息 -->
                    <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.basic_info') }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div class="flex items-center justify-between p-2">
                                <span class="text-base-content/70">{{ t('video_download.title') }}</span>
                                <div class="flex items-center gap-2">
                                    <span class="font-medium text-right truncate max-w-[400px]">{{ detailItem?.title
                                    }}</span>
                                    <button v-if="actionStates.title" class="btn btn-ghost btn-xs tooltip"
                                        :data-tip="t('video_download.copy')"
                                        @click="copyText(detailItem?.title, 'title')">
                                        <v-icon name="md-contentcopy" class="w-3 h-3"></v-icon>
                                    </button>
                                </div>
                            </div>
                            <li class="divider-thin my-1"></li>
                            <div class="flex items-center justify-between p-2">
                                <span class="text-base-content/70">URL</span>
                                <div class="flex items-center gap-2">
                                    <span class="font-medium text-right truncate max-w-[400px]">{{ detailItem?.url
                                    }}</span>
                                    <button v-if="actionStates.url" class="btn btn-ghost btn-xs tooltip"
                                        :data-tip="t('video_download.copy')" @click="copyText(detailItem?.url, 'url')">
                                        <v-icon name="md-contentcopy" class="w-3 h-3"></v-icon>
                                    </button>
                                </div>
                            </div>
                        </div>
                    </ul>

                    <!-- 下载信息 -->
                    <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.download_info') }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div class="grid grid-cols-2 items-center">
                                <div class="flex items-center justify-between p-2 pr-4">
                                    <span class="text-base-content/70">{{ t('video_download.source')
                                        }}</span>
                                    <span>{{ detailItem?.source }}</span>
                                </div>
                                <div class="flex items-center justify-between p-2 pl-4 border-l border-base-300">
                                    <span class="text-base-content/70">{{ t('video_download.format')
                                        }}</span>
                                    <span>{{ detailItem.streams && detailItem.streams.length > 0 ?
                                        detailItem.streams[0].ext
                                        :
                                        'N/A' }}</span>
                                </div>
                            </div>
                            <li class="divider-thin my-1"></li>
                            <div class="grid grid-cols-2 items-center">
                                <div class="flex items-center justify-between p-2 pr-4">
                                    <span class="text-base-content/70">{{ t('video_download.quality')
                                        }}</span>
                                    <span class="font-medium truncate max-w-[200px]"
                                        :title="detailItem.streams && detailItem.streams.length > 0 ? detailItem.streams[0].quality : 'N/A'">{{
                                            detailItem.streams && detailItem.streams.length > 0 ?
                                        detailItem.streams[0].quality :
                                        'N/A' }}</span>
                                </div>
                                <div class="flex items-center justify-between p-2 pl-4 border-l border-base-300">
                                    <span class="text-base-content/70">{{ t('video_download.size') }}</span>
                                    <span>{{ formatFileSize(detailItem?.totalSize) }}</span>
                                </div>
                            </div>
                        </div>
                    </ul>

                    <!-- 状态信息 -->
                    <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.status_info') }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div class="grid grid-cols-2 divide-x">
                                <div class="flex items-center justify-between p-2">
                                    <span class="text-base-content/70">{{ t('video_download.start_time') }}</span>
                                    <span class="font-medium">{{ detailItem?.startTime ? new
                                        Date(detailItem.startTime).toLocaleString() : '-' }}</span>
                                </div>
                                <div class="flex items-center justify-between p-2">
                                    <span class="text-base-content/70 ml-2">{{ t('video_download.end_time') }}</span>
                                    <span class="font-medium">{{ detailItem?.endTime && !isZeroTime(detailItem.endTime)
                                        ? new
                                            Date(detailItem.endTime).toLocaleString() : '-' }}</span>
                                </div>
                            </div>
                            <li class="divider-thin my-1"></li>
                            <div class="grid grid-cols-2 divide-x">
                                <div class="flex items-center justify-between p-2">
                                    <span class="text-base-content/70">{{ t('video_download.average_speed') }}</span>
                                    <span class="font-medium">{{ detailItem?.averageSpeed || '-' }}</span>
                                </div>
                                <div class="flex items-center justify-between p-2">
                                    <span class="text-base-content/70 ml-2">{{ t('video_download.duration') }}</span>
                                    <span class="font-medium">{{ detailItem?.durationSeconds ?
                                        formatDuration(detailItem.durationSeconds) : '-' }}</span>
                                </div>
                            </div>
                            <li class="divider-thin my-1"></li>
                            <div class="grid grid-cols-2 items-center">
                                <div class="flex items-center justify-between p-2 pr-4">
                                    <span class="text-base-content/70">{{ t('video_download.parts_info')
                                        }}</span>
                                    <span>{{ detailItem?.finishedParts || '-' }}/{{ detailItem?.totalParts || '-'
                                        }}</span>
                                </div>
                                <div class="flex items-center justify-between p-2 pl-4 border-l border-base-300">
                                    <span class="text-base-content/70">{{ t('video_download.status')
                                        }}</span>
                                    <span>
                                        <button class="btn btn-ghost btn-xs p-0"
                                            @click.stop="showStatusDetail(detailItem)">
                                            <v-icon :name="statusIcon[detailItem?.taskStatus] || 'ri-file-unknow-line'"
                                                class="w-4 h-4" :title="getTaskStatusText(detailItem?.taskStatus)" />
                                        </button>
                                        {{ getTaskStatusText(detailItem?.taskStatus) }}
                                    </span>
                                </div>
                            </div>
                            <li v-if="detailItem?.savedPath" class="divider-thin my-1"></li>
                            <div v-if="detailItem?.savedPath" class="p-2">
                                <div class="flex items-center justify-between">
                                    <span class="text-base-content/70">{{ t('video_download.saved_path')
                                        }}</span>
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium text-right truncate max-w-[400px]">{{
                                            detailItem?.savedPath
                                        }}</span>
                                        <button v-if="actionStates.folder" class="btn btn-ghost btn-xs tooltip"
                                            :data-tip="t('video_download.open_folder')"
                                            @click="openFolder(detailItem.savedPath)">
                                            <v-icon name="ri-folder-open-line" class="w-3 h-3"></v-icon>
                                        </button>
                                    </div>
                                </div>
                            </div>
                            <li v-if="detailItem?.error" class="divider-thin my-1"></li>
                            <div v-if="detailItem?.error" class="p-2">
                                <div class="flex items-center justify-between">
                                    <span class="text-base-content/70">{{ t('video_download.error')
                                        }}</span>
                                    <span class="font-medium text-error truncate max-w-[400px]">{{
                                        detailItem?.error
                                    }}</span>
                                </div>
                            </div>
                            <li v-if="detailItem?.updatedAt" class="divider-thin my-1"></li>
                            <div v-if="detailItem?.updatedAt" class="flex items-center justify-between p-2">
                                <span class="text-base-content/70">{{ t('video_download.updated_at') }}</span>
                                <span>{{ new Date(detailItem?.updatedAt).toLocaleString() }}</span>
                            </div>
                        </div>
                    </ul>

                    <div class="modal-action">
                        <button class="btn" @click="detailModal = false">{{ t('common.close') }}</button>
                    </div>
                </div>
            </div>
        </div>

        <!-- 状态详情模态框 -->
        <input type="checkbox" id="modal-status" class="modal-toggle" v-model="statusModal" />
        <div class="modal" @click.self="statusModal = false">
            <div class="modal-box max-w-3xl bg-base-200 py-4">
                <div class="space-y-4">
                    <!-- 基本状态信息 -->
                    <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.status_detail') || '状态详情'
                            }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div class="flex items-center justify-between p-2">
                                <span class="text-base-content/70">{{ t('video_download.title') }}</span>
                                <span class="font-medium text-right truncate max-w-[400px]">{{ statusItem?.title
                                }}</span>
                            </div>
                            <li class="divider-thin my-1"></li>
                            <div class="flex items-center justify-between p-2">
                                <span class="text-base-content/70">{{ t('video_download.status') }}</span>
                                <div class="flex items-center gap-2">
                                    <v-icon :name="statusIcon[statusItem?.taskStatus] || 'ri-file-unknow-line'"
                                        class="w-4 h-4" />
                                    <span class="font-medium">{{ getTaskStatusText(statusItem?.taskStatus) }}</span>
                                </div>
                            </div>
                        </div>
                    </ul>

                    <!-- 流信息 -->
                    <ul v-if="statusItem?.streams && statusItem.streams.length > 0"
                        class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.stream_parts') }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div v-for="(stream, index) in statusItem.streams" :key="stream.partId" class="mb-4">
                                <div class="flex items-center justify-between p-2 bg-base-200 rounded-lg">
                                    <span class="font-medium">{{ t('video_download.stream') }} #{{ index + 1 }}</span>
                                    <div class="badge" :class="stream.finalStatus ? 'badge-success' : 'badge-error'">
                                        {{ stream.status }}
                                    </div>
                                </div>
                                <div class="p-2 mt-2">
                                    <div class="grid grid-cols-2 gap-2">
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.name') }}</span>
                                            <span class="font-medium">{{ stream.name }}</span>
                                        </div>
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.quality') }}</span>
                                            <span class="font-medium truncate max-w-[200px]" :title="stream.quality">{{
                                                stream.quality }}</span>
                                        </div>
                                    </div>
                                    <div class="grid grid-cols-2 gap-2">
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.format') }}</span>
                                            <span class="font-medium">{{ stream.ext }}</span>
                                        </div>
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.size') }}</span>
                                            <span class="font-medium">{{ formatFileSize(stream.totalSize) }}</span>
                                        </div>
                                    </div>
                                    <!-- 流信息的消息显示 -->
                                    <div class="p-2 mt-2 bg-base-200 rounded-lg">
                                        <div class="text-base-content/70 mb-1">{{ t('video_download.message') }}</div>
                                        <div class="overflow-y-auto max-h-[150px]">
                                            <p v-if="stream.message" class="text-sm break-all whitespace-normal"
                                                style="word-break: break-word; white-space: pre-line;">{{ stream.message
                                                }}</p>
                                            <p v-else class="text-sm text-base-content/50 italic">{{
                                                t('video_download.no_message') || '无信息' }}</p>
                                        </div>
                                    </div>
                                </div>
                                <li v-if="index < statusItem.streams.length - 1" class="divider-thin my-2"></li>
                            </div>
                        </div>
                    </ul>

                    <!-- 通用部分信息 -->
                    <ul v-if="statusItem?.commonParts && statusItem.commonParts.length > 0"
                        class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                        <div class="flex items-center p-2 pl-4 rounded-lg bg-base-100">
                            <h2 class="font-semibold text-base-content">{{ t('video_download.common_parts') }}</h2>
                        </div>
                        <li class="divider-thin"></li>
                        <div class="p-4 rounded-lg bg-base-100">
                            <div v-for="(part, index) in statusItem.commonParts" :key="part.partId" class="mb-4">
                                <div class="flex items-center justify-between p-2 bg-base-200 rounded-lg">
                                    <span class="font-medium">{{ part.type === 'caption' ? (t('video_download.caption'))
                                        :
                                        part.type }}</span>
                                    <div class="badge" :class="part.finalStatus ? 'badge-success' : 'badge-error'">
                                        {{ part.status }}
                                    </div>
                                </div>
                                <div class="p-2 mt-2">
                                    <div class="grid grid-cols-2 gap-2">
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.name') }}</span>
                                            <span class="font-medium">{{ part.name }}</span>
                                        </div>
                                        <div class="flex items-center justify-between p-2">
                                            <span class="text-base-content/70">{{ t('video_download.format') }}</span>
                                            <span class="font-medium">{{ part.ext }}</span>
                                        </div>
                                    </div>
                                    <!-- 通用部分的消息显示 -->
                                    <div class="p-2 mt-2 bg-base-200 rounded-lg">
                                        <div class="text-base-content/70 mb-1">{{ t('video_download.message') }}</div>
                                        <div class="overflow-y-auto max-h-[150px]">
                                            <p v-if="part.message" class="text-sm break-all whitespace-normal"
                                                style="word-break: break-word; white-space: pre-line;">{{ part.message
                                                }}</p>
                                            <p v-else class="text-sm text-base-content/50 italic">{{
                                                t('video_download.no_message') || '无信息' }}</p>
                                        </div>
                                    </div>
                                </div>
                                <li v-if="index < statusItem.commonParts.length - 1" class="divider-thin my-2"></li>
                            </div>
                        </div>
                    </ul>

                    <div class="modal-action">
                        <button class="btn" @click="statusModal = false">{{ t('common.close') }}</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { GetContent, StartDownload, CheckFFMPEG, DeleteRecord } from 'wailsjs/go/api/DownloadAPI'
import useDownloadStore from '@/stores/download'
import { storeToRefs } from "pinia";
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n';
const downloadStore = useDownloadStore()
const { instantData } = storeToRefs(downloadStore)
const { t } = useI18n();

const url = ref('')
const isLoading = ref(false)
const videoData = ref([])
const selectedCaption = ref({})
const selectedFormat = ref('')
const selectedQuality = ref(null)

const errorModal = ref(null)
const errorInfo = ref('')
const detailModal = ref(false)
const detailItem = ref({})
const actionStates = ref({
    title: true,
    url: true,
    folder: true
})

const statusModal = ref(false)
const statusItem = ref({})

// 展开状态管理
const expandedState = ref(new Map()) // taskId -> { type: 'auto' | 'manual' }

// Calculate all unique formats
const uniqueFormats = computed(() => {
    if (!videoData.value.qualities) return []
    return [...new Set(videoData.value.qualities.map(q => q.format))]
})

// Filter qualities based on selected format
const filteredQualities = computed(() => {
    if (!selectedFormat.value || !videoData.value.qualities) return []
    return videoData.value.qualities.filter(q => q.format === selectedFormat.value)
})

// 监听数据变化
watch(() => instantData.value, (newData) => {
    newData.forEach(item => {
        const currentState = expandedState.value.get(item.taskId)
        if (item.isProcessing) {
            // 如果是下载中且没有手动展开，设置为自动展开
            if (!currentState || currentState.type === 'auto') {
                expandedState.value.set(item.taskId, { type: 'auto' })
            }
        } else {
            // 如果不是下载中且是自动展开的，则移除
            if (currentState?.type === 'auto') {
                expandedState.value.delete(item.taskId)
            }
        }
    })
}, { deep: true })

async function handleGet() {
    isLoading.value = true
    try {
        const { data, success, msg } = await GetContent(url.value)
        if (!success) {
            $dialog.error(msg)
            return
        }

        // Parse video data
        const responseData = JSON.parse(data)
        if (!responseData) {
            $message.error(t('video_download.no_data'))
            return
        }

        let parsedData = responseData.videos[0]

        // Extract basic information
        const videoInfo = {
            id: parsedData.id || '',
            source: parsedData.source || '',
            title: parsedData.title || '',
            url: parsedData.url || '',
            site: parsedData.site || ''
        }

        // Extract all available video qualities
        const qualities = []
        if (parsedData.streams && typeof parsedData.streams === 'object') {
            Object.entries(parsedData.streams).forEach(([id, stream]) => {
                if (stream && stream.parts && stream.parts[0]) {
                    qualities.push({
                        id,
                        quality: stream.quality || '',
                        size: stream.size || 0,
                        format: stream.ext || '',
                        parts: stream.parts.length || 0,
                    })
                }
            })

            // Sort qualities by quality
            qualities.sort((a, b) => {
                const getQualityNumber = (quality) =>
                    parseInt(quality.match(/\d+/)?.[0] || '0')
                return getQualityNumber(b.quality) - getQualityNumber(a.quality)
            })
        }

        // select the first format
        if (qualities.length > 0) {
            const formats = [...new Set(qualities.map(q => q.format))]
            if (formats.length > 0) {
                selectedFormat.value = formats[0]
                // 自动选择当前格式下文件大小最大的质量选项
                const qualitiesForFormat = qualities
                    .filter(q => q.format === formats[0])
                    .sort((a, b) => b.size - a.size)  // 按文件大小降序排序
                if (qualitiesForFormat.length > 0) {
                    selectedQuality.value = qualitiesForFormat[0]  // 选择最大的
                }
            }
        }

        const captions = []
        if (parsedData.captions && typeof parsedData.captions === 'object') {
            Object.entries(parsedData.captions).forEach(([id, caption]) => {
                if (caption && caption.url) {
                    captions.push({
                        id,
                        language: id,
                    })
                }
            })
        } else {
            $message.warning(t('video_download.no_caption'))
        }

        // Set video data
        videoData.value = {
            info: videoInfo,
            qualities: qualities,
            captions: captions,
        }
    } catch (error) {
        videoData.value = {
            info: null,
            qualities: [],
            captions: []
        }
    } finally {
        isLoading.value = false
        requestDownloading.value = false
    }
}

const requestDownloading = ref(false)
async function download() {
    try {
        requestDownloading.value = true  // 设置提交状态

        // check ffmpeg
        const { success, msg } = await CheckFFMPEG()
        if (!success) {
            $message.error(msg)
            requestDownloading.value = false
            return
        }

        // set total
        let total = 0
        if (selectedQuality.value.id) {
            total++
        }
        if (selectedCaption.value.id) {
            total++
        }

        let reqBody = {
            taskId: "",
            contentId: videoData.value.info.id,
            total: total,
            stream: selectedQuality.value.id,
            captions: selectedCaption.value.id ? [selectedCaption.value.id] : [],
            danmakus: "",
        }

        const { success: createSuccess, data: createData, msg: createMsg } = await StartDownload(reqBody)
        if (!createSuccess) {
            $message.error(createMsg)
            isSubmitting.value = false
            return
        }

        // 刷新下载列表
        await refreshInstantData()

        requestDownloading.value = false  // 重置提交状态
    } catch (error) {
        $message.error(t('video_download.download_failed'))
        requestDownloading.value = false
    }
}

async function deleteRecord(taskId) {
    if (!taskId) return
    const { success, msg } = await DeleteRecord(taskId)
    if (success) {
        $message.success(t('video_download.delete_success'))
        downloadStore.setInstantData()
    } else {
        $message.error(msg)
    }
}

const sourceIcon = (source) => {
    const icons = {
        "bilibili": "ri-bilibili-line",
        "youtube": "ri-youtube-line"
    }
    return icons[source] || "ri-video-line"  // 如果没有匹配的图标，返回一个通用的视频图标
}

const statusIcon = {
    0: "md-task", // 0: 任务已创建 MdTask
    1: "md-downloading",  // 1: 等待下载
    2: "md-downloading",  // 2: 正在下载
    3: "md-pause",  // 3: 已暂停 MdPause
    4: "bi-puzzle",  // 4: 正在合并分片 BiPuzzle
    5: "bi-puzzle-fill", // 5: 合并成功  BiPuzzleFill
    6: "co-puzzle", // 6: 合并失败 CoPuzzle
    7: "md-downloaddone", // 7: 下载完成 MdDownloaddone
    8: "md-runningwitherrors", // 8: 下载失败 MdRunningwitherrors
    9: "md-filedownloadoff", // 9: 部分分下下载成功 MdFiledownloadoff
    10: "md-filedownloadoff", // 10: 部分分片下载失败 MdFiledownloadoff
    11: "md-freecancellation", // 11: 已取消 MdFreecancellation 
    12: "ri-file-unknow-line", // 12: 未知状态
}

const getTaskStatusText = (status) => {
    const statusMap = {
        0: "task_created",
        1: "waiting_download",
        2: "downloading",
        3: "paused",
        4: "merging",
        5: "merge_success",
        6: "merge_failed",
        7: "download_completed",
        8: "download_failed",
        9: "partial_success",
        10: "partial_failed",
        11: "cancelled",
        12: "unknown_status",
    }
    const key = statusMap[status]
    return key ? t(`video_download.status_${key}`) : t('video_download.unknown')
}

// 监听数据变化
watch(() => instantData.value, (newData) => {
    newData.forEach(item => {
        const currentState = expandedState.value.get(item.taskId)
        if (item.isProcessing) {
            // 如果是下载中且没有手动展开，设置为自动展开
            if (!currentState || currentState.type === 'auto') {
                expandedState.value.set(item.taskId, { type: 'auto' })
            }
        } else {
            // 如果不是下载中且是自动展开的，则移除
            if (currentState?.type === 'auto') {
                expandedState.value.delete(item.taskId)
            }
        }
    })
}, { deep: true })

const toggleExpand = (taskId) => {
    const currentState = expandedState.value.get(taskId)
    if (currentState) {
        expandedState.value.delete(taskId)
    } else {
        expandedState.value.set(taskId, { type: 'manual' })
    }
}

const formatDuration = (duration) => {
    if (!duration) return 'N/A'
    const totalSeconds = Math.floor(duration / 1000 / 1000)
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

const formatQuality = (quality) => {
    if (!quality) return ''
    // Split by semicolon, take the first part
    const beforeSemicolon = quality.split(';')[0]
    // Split by space, take the first part
    const beforeSpace = beforeSemicolon.split(' ')[0]
    return beforeSpace.split('/')[0]
}

const refreshInstantData = () => {
    downloadStore.setInstantData()
}

onMounted(() => {
    refreshInstantData()
})

onUnmounted(() => {
    document.removeEventListener('click', handleGlobalClick)
})

function handleGlobalClick(event) {
    // check if the click is outside the tasks list
    const tasksElement = document.querySelector('.tasks-list')
    if (!tasksElement?.contains(event.target)) {
    }
}


const formatFileSize = (size) => {
    if (!size) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let index = 0
    let fileSize = Number(size)

    while (fileSize >= 1024 && index < units.length - 1) {
        fileSize /= 1024
        index++
    }

    return `${fileSize.toFixed(2)} ${units[index]}`
}

const openFolder = (savedPath) => {
    OpenDirectory(savedPath)
}

function downloadFailed(total, finished, status) {
    if (total === finished) {
        if (String(status).includes('Failed')) {
            return true
        }
    }
    return false
}

const showErrorModal = (error) => {
    errorInfo.value = error
    errorModal.value = true
}

const showDetail = (item) => {
    detailItem.value = item
    detailModal.value = true
    actionStates.value = {
        title: true,
        url: true,
        folder: true
    }
}

const showStatusDetail = (item) => {
    statusItem.value = item
    statusModal.value = true
}

const copyText = (text, type) => {
    if (text) {
        navigator.clipboard.writeText(text)
        // actionStates.value[type] = false
        $message.success(t('video_download.copy_success'))
    }
}

const copyItemInfo = () => {
    const info = `${t('video_download.title')}: ${detailItem.value.title}
${t('video_download.url')}: ${detailItem.value.url}
${t('video_download.format')}: ${detailItem.value.streams && detailItem.value.streams.length > 0 ? detailItem.value.streams[0].ext : 'N/A'}
${t('video_download.quality')}: ${detailItem.value.streams && detailItem.value.streams.length > 0 ? detailItem.value.streams[0].quality : 'N/A'}
${t('video_download.status')}: ${detailItem.value.status}
${t('video_download.progress')}: ${Number(detailItem.value.progress).toFixed(2)}%
${t('video_download.finished')}: ${detailItem.value.finished}
${t('video_download.total')}: ${detailItem.value.total}
${t('video_download.size')}: ${formatFileSize(detailItem.value.size)}
${t('video_download.saved_path')}: ${detailItem.value.savedPath}
${detailItem.value.error ? `${t('video_download.error_info')}: ${detailItem.value.error}` : ''}`

    navigator.clipboard.writeText(info)
}

watch(detailModal, (newVal) => {
    if (newVal) {
        actionStates.value = {
            title: true,
            url: true,
            folder: true
        }
    }
})

const isZeroTime = (time) => {
    return time === '0001-01-01T00:00:00Z'
}
</script>

<style scoped>
.download-container {
    height: 100%;
    width: 100%;
    min-width: 500px;
    flex-shrink: 0;
    overflow: hidden;
}

/* Add smooth transitions */
.card {
    transition: all 0.2s ease-in-out;
}

/* Add responsive table styles */
@media (max-width: 768px) {
    .table {
        font-size: 0.85rem;
    }
}

/* Improve table header contrast */
.table-header {
    color: hsl(var(--p));
    border-bottom: 2px solid rgba(147, 51, 234, 0.3);
}

/* Fix dropdown menu positioning */
.dropdown-menu-fixed {
    position: fixed;
    margin-top: 0;
}

/* Make sure the dropdown has high z-index and doesn't get clipped */
.card-body {
    overflow: visible !important;
}

.table-container {
    overflow: visible;
}

.card {
    overflow: visible !important;
}

/* Ensure table parent has proper overflow handling */
.overflow-x-auto {
    overflow-x: auto;
    overflow-y: visible !important;
}

.animate-fadeIn {
    animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(-4px);
    }

    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>