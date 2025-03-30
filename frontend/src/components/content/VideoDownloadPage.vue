<template>
  <div class="flex flex-col h-full bg-base-200">
    <!-- top toolbar -->
    <div class="flex justify-between items-center p-3 border-b border-base-200">
      <div class="flex items-center">
        <v-icon name="ri-download-cloud-line" class="w-5 h-5 mr-2 text-primary"></v-icon>
        <span class="text-lg font-bold"> {{ $t('download.title') }}</span>
      </div>
      <div class="flex items-center space-x-2">
        <button class="btn btn-xs btn-primary" @click="openDownloadModal">
          <v-icon name="ri-add-line" class="h-3.5 w-3.5 mr-1"></v-icon>
          {{ $t('download.new_task') }}
        </button>
      </div>
    </div>

    <!-- main content area -->
    <div class="flex-1 overflow-auto p-4">
      <!-- active task area -->
      <div v-if="activeTask" class="mb-6 animate-fadeIn">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-base font-semibold"> {{ $t('download.brief') }}</h2>
        </div>

        <div class="card bg-base-100 shadow-md border border-base-200">
          <div class="card-body p-3">
            <!-- task brief -->
            <div class="flex items-center mb-3">
              <div class="w-14 h-9 bg-base-200 rounded overflow-hidden flex-shrink-0 mr-3">
                <template v-if="!thumbnailLoadError && activeTask.thumbnail">
                  <img :src="activeTask.thumbnail" class="w-full h-full object-cover" :alt="$t('download.thumbnail')"
                    @error="handleThumbnailError" />
                </template>
                <div v-else
                  class="w-full h-full flex flex-col items-center justify-center text-base-content/30 bg-base-200">
                  <v-icon name="ri-video-line" class="w-5 h-5 mb-1"></v-icon>
                </div>
              </div>
              <div class="flex-1">
                <div class="flex items-center">
                  <div class="font-medium text-sm truncate mr-2">{{ activeTask.title }}</div>
                  <div class="badge badge-sm" :class="statusBadgeClass(activeTask.stage)">
                    {{ statusText(activeTask.stage) }}
                  </div>
                </div>
                <div class="text-xs text-base-content/70 mt-0.5">
                  {{ formatDuration(activeTask.duration) }} · {{ formatFileSize(activeTask.fileSize) }} · {{
                    activeTask.extractor }} · {{
                    activeTask.uploader }}
                </div>
              </div>
              <div class="flex items-center space-x-2">
                <button
                  v-if="activeTask.stage === 'downloading' || activeTask.stage === 'translating' || activeTask.stage === 'embedding'"
                  class="btn btn-xs btn-warning" @click="pauseTask(activeTask.id)">
                  <v-icon name="ri-pause-line" class="w-3 h-3 mr-1"></v-icon>
                  {{ $t('download.pause') }}
                </button>
                <button v-if="activeTask.stage === 'paused'" class="btn btn-xs btn-primary"
                  @click="resumeTask(activeTask.id)">
                  <v-icon name="ri-play-line" class="w-3 h-3 mr-1"></v-icon>
                  {{ $t('download.resume') }}
                </button>
                <button class="btn btn-xs btn-outline" @click="showTaskDetail(activeTask)">
                  <v-icon name="ri-information-line" class="w-3 h-3 mr-1"></v-icon>
                  {{ $t('download.detail') }}
                </button>
              </div>
            </div>

            <!-- progress information area -->
            <div class="mb-4">
              <!-- progress bar part - keep width 100% of container -->
              <div class="mb-3">
                <div class="flex justify-between items-center mb-2">
                  <div class="text-sm font-medium">{{ $t('download.progress') }}</div>
                  <div class="text-sm">{{ activeTask.percentage.toFixed(2) }}%</div>
                </div>
                <div class="w-full h-2 bg-base-200 rounded-full overflow-hidden">
                  <div class="h-full bg-primary transition-all duration-300"
                    :style="{ width: `${activeTask.percentage}%` }"></div>
                </div>
              </div>
            </div>

            <!-- workflow stages -->
            <div class="bg-base-200 p-2 rounded-lg">
              <div class="flex items-center justify-between">
                <!-- left workflow -->
                <div class="flex items-center space-x-0.5">
                  <!-- only show applicable stages -->
                  <template v-for="(stage, index) in filteredStages(activeTask)" :key="stage.id">
                    <!-- stage icon and text -->
                    <div class="flex items-center"
                      :class="{ 'text-primary font-medium': activeTask.stage === stage.id }">
                      <div class="w-5 h-5 rounded-full flex items-center justify-center" :class="{
                        'bg-primary text-white': activeTask.stage === stage.id,
                        'bg-success text-white': isStageCompleted(stage.id),
                        'bg-base-300 text-base-content/70': !isStageCompleted(stage.id) && activeTask.stage !== stage.id
                      }">
                        <v-icon :name="stage.icon" class="w-3.5 h-3.5"></v-icon>
                      </div>
                      <span class="ml-1 text-xs">{{ stage.label }}</span>
                    </div>

                    <!-- connection line -->
                    <div v-if="index < filteredStages(activeTask).length - 1" class="h-0.5 w-6" :class="{
                      'bg-success': isStageCompleted(stage.id),
                      'bg-base-300': !isStageCompleted(stage.id)
                    }"></div>
                  </template>
                </div>

                <!-- download information -->
                <div v-if="activeTask.downloadSpeed || activeTask.estimatedTime"
                  class="text-xs text-base-content/80 flex items-center pl-2 ml-2">
                  <div class="flex text-xs text-base-content/70 space-x-6">
                    <div v-if="activeTask.speed" class="flex items-center">
                      <v-icon name="ri-download-line" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.speed') }}:</span> {{ activeTask.speed }}
                    </div>
                    <div v-if="activeTask.stage === 'initializing'" class="flex items-center">
                      <v-icon name="co-infinity" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.initializing') }}</span>
                    </div>
                    <div v-if="activeTask.stage === 'completed'" class="flex items-center">
                      <v-icon name="ri-check-line" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.completed') }}</span>
                    </div>
                    <div v-if="activeTask.stage === 'paused'" class="flex items-center">
                      <v-icon name="ri-pause-line" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.paused') }}</span>
                    </div>
                    <div
                      v-if="activeTask.stage === 'downloading' || activeTask.stage === 'translating' || activeTask.stage === 'embedding'"
                      class="flex items-center">
                      <div v-if="activeTask.stageInfo" class="flex items-center">
                        <v-icon name="ri-file-type-line" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                        <span class="font-medium">{{ $t('download.file_type') }}:</span>
                        <span class="ml-1 badge badge-xs" :class="{
                          'badge-primary': activeTask.stageInfo.includes('video'),
                          'badge-secondary': activeTask.stageInfo.includes('audio'),
                          'badge-accent': activeTask.stageInfo.includes('subtitle')
                        }">{{ activeTask.stageInfo }}</span>
                      </div>
                      <div v-if="activeTask.estimatedTime" class="flex items-center ml-2">
                        <v-icon name="ri-time-line" class="w-3 h-3 mr-1 text-primary/70"></v-icon>
                        <span class="font-medium">{{ $t('download.estimated_time') }}:</span> {{
                          activeTask.estimatedTime }}
                      </div>
                    </div>
                    <div v-if="activeTask.stage === 'failed'" class="flex items-center">
                      <v-icon name="ri-error-warning-line" class="w-3.5 h-3.5 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.failed') }}</span>
                    </div>
                    <div v-if="activeTask.stage === 'cancelled'" class="flex items-center">
                      <v-icon name="md-freecabcellation" class="w-3.5 h-3.5 mr-1 text-primary/70"></v-icon>
                      <span class="font-medium">{{ $t('download.cancelled') }}</span>
                    </div>
                  </div>
                </div>

                <!-- current stage info -->
                <div v-if="currentStageInfo"
                  class="text-xs text-base-content/80 flex items-center border-l border-base-300 pl-2 ml-2">
                  <div class="w-4 h-4 rounded-full bg-primary/10 flex items-center justify-center mr-2">
                    <v-icon :name="currentStageInfo.icon" class="w-3.5 h-3.5 text-primary"></v-icon>
                  </div>
                  <span>{{ currentStageInfo.description }}</span>
                </div>
              </div>
            </div>

          </div>
        </div>
      </div>

      <!-- task list -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <!-- tasks title-->
          <h2 class="text-base font-semibold">{{ $t('download.tasks') }}</h2>

          <!-- right aligned elements -->
          <div class="flex items-center space-x-2">
            <!-- tasks count-->
            <span class="text-xs text-base-content/70">{{ filteredTasks.length }} / {{ tasks.length }} {{
              $t('download.tasks')
            }}</span>

            <!-- filter button -->
            <div class="dropdown dropdown-end">
              <label tabindex="0" class="btn btn-xs btn-outline">
                <v-icon name="ri-filter-3-line" class="h-3.5 w-3.5 mr-1"></v-icon>
                {{filterOptions.find(option => option.id === filterStage)?.label}}
              </label>
              <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52 text-xs">
                <li v-for="option in filterOptions" :key="option.id">
                  <a @click="toggleFilter(option.id)">{{ option.label }}</a>
                </li>
              </ul>
            </div>

            <!-- refresh button -->
            <button class="btn btn-xs btn-outline" @click="refreshTasks()">
              <v-icon name="ri-refresh-line" class="h-3.5 w-3.5 mr-1"></v-icon>
              {{ $t('download.refresh') }}
            </button>
          </div>
        </div>

        <div v-if="filteredTasks.length > 0" class="space-y-2.5">
          <div v-for="task in filteredTasks" :key="task.id"
            class="card bg-base-100 shadow-md border border-base-200 hover:shadow-lg transition-shadow duration-200"
            :class="{ 'border-primary/30 bg-primary/5': task.id === activeTask?.id }">
            <div class="card-body p-2.5 cursor-pointer" @click="switchActiveTask(task)">
              <div class="flex items-center">
                <!-- task information -->
                <div class="w-12 h-8 bg-base-200 rounded-md overflow-hidden flex-shrink-0 mr-2.5">
                  <template v-if="!task.thumbnailLoadError && task.thumbnail">
                    <img :src="task.thumbnail" class="w-full h-full object-cover" :alt="$t('download.thumbnail')"
                      @error="handleTaskThumbnailError" />
                  </template>
                  <div v-else
                    class="w-full h-full flex flex-col items-center justify-center text-base-content/30 bg-base-200">
                    <v-icon name="ri-video-line" class="w-5 h-5 mb-1"></v-icon>
                  </div>
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center">
                    <div class="font-medium text-sm truncate mr-2">{{ task.title }}</div>
                    <div class="badge badge-sm" :class="statusBadgeClass(task.stage)">
                      {{ statusText(task.stage) }}
                    </div>
                  </div>
                  <div class="text-xs text-base-content/70 mt-0.5">
                    {{ formatDuration(task.duration) }} · {{ formatFileSize(task.fileSize) }} · {{ task.extractor }} ·{{
                      task.uploader }}
                  </div>
                </div>

                <!-- simplified pipeline - only show applicable stage icons -->
                <div class="hidden md:flex items-center space-x-1 mx-3">
                  <div v-for="stage in filteredStages(task)" :key="`${task.id}-${stage.id}`" class="flex items-center">
                    <div class="w-4 h-4 rounded-full flex items-center justify-center text-xs" :class="{
                      'bg-primary/20 text-primary': task.stage === stage.id,
                      'bg-success/20 text-success': isStageCompleted(stage.id),
                      'bg-base-200 text-base-content/50': !isStageCompleted(stage.id) && task.stage !== stage.id
                    }">
                      <v-icon :name="stage.icon" class="w-3 h-3"></v-icon>
                    </div>

                    <div v-if="filteredStages(task).indexOf(stage) < filteredStages(task).length - 1"
                      class="w-3 h-1 mx-px" :class="{
                        'bg-success': isStageCompleted(stage.id),
                        'bg-base-200': !isStageCompleted(stage.id)
                      }"></div>
                  </div>
                </div>

                <!-- progress bar -->
                <div class="w-20 flex-shrink-0 mr-3">
                  <div class="h-1.5 w-full bg-base-200 rounded-full overflow-hidden">
                    <div class="h-full transition-all duration-300" :class="statusBarClass(task.stage)"
                      :style="{ width: `${task.percentage.toFixed(1)}%` }"></div>
                  </div>
                </div>

                <!-- action buttons -->
                <div class="flex items-center space-x-1">
                  <button v-if="task.stage === 'completed'" class="btn btn-xs btn-outline"
                    @click.stop="openDirectory(task.outputDir)" title="{{ $t('download.open_folder') }}">
                    <v-icon name="ri-folder-open-line" class="w-3 h-3"></v-icon>
                  </button>
                  <button v-if="task.stage === 'failed'" class="btn btn-xs btn-error btn-outline"
                    @click.stop="showError(task)" title="{{ $t('download.view_error') }}">
                    <v-icon name="ri-error-warning-line" class="w-3 h-3"></v-icon>
                  </button>
                  <button v-if="task.stage === 'paused'" class="btn btn-xs btn-primary btn-outline"
                    @click.stop="resumeTask(task.id)" title="{{ $t('download.start_continue') }}">
                    <v-icon name="ri-play-line" class="w-3 h-3"></v-icon>
                  </button>
                  <button v-if="task.stage === 'downloading'" class="btn btn-xs btn-warning btn-outline"
                    @click.stop="pauseTask(task.id)" title="{{ $t('download.pause') }}">
                    <v-icon name="ri-pause-line" class="w-3 h-3"></v-icon>
                  </button>
                  <button class="btn btn-xs btn-outline" @click="showTaskDetail(task)"
                    title="{{ $t('download.detail') }}">
                    <v-icon name="ri-information-line" class="w-3 h-3"></v-icon>
                  </button>
                  <button class="btn btn-xs btn-error btn-outline" @click="deleteTask(task.id)"
                    title="{{ $t('download.delete') }}">
                    <v-icon name="ri-delete-bin-line" class="w-3 h-3"></v-icon>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Filtered empty state -->
        <div v-else-if="filteredTasks.length === 0 && tasks.length > 0"
          class="card bg-base-100 shadow-md border border-base-200">
          <div class="card-body p-6 flex flex-col items-center justify-center">
            <v-icon name="ri-inbox-line" class="w-12 h-12 text-base-content/20 mb-3"></v-icon>
            <h3 class="text-base font-medium mb-1.5">{{ $t('download.no_filter_results') }}</h3>
          </div>
        </div>

        <!-- tasks empty state -->
        <div v-else class="card bg-base-100 shadow-md border border-base-200">
          <div class="card-body p-6 flex flex-col items-center justify-center">
            <v-icon name="ri-inbox-line" class="w-12 h-12 text-base-content/20 mb-3"></v-icon>
            <h3 class="text-base font-medium mb-1.5">{{ $t('download.no_download_tasks') }}</h3>
            <p class="text-base-content/70 text-center mb-4">{{ $t('download.start_first_download_task') }}</p>
            <button class="btn btn-primary" @click="openDownloadModal">
              <v-icon name="ri-add-line" class="h-4 w-4 mr-1"></v-icon>
              {{ $t('download.new_task') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- new download modal -->
    <VideoDownloadModal :show="showDownloadModal" @update:show="showDownloadModal = $event"
      @download-started="onDownloadStarted" />
    <!-- task detail modal -->
    <div class="modal" :class="{ 'modal-open': selectedTaskId }">
      <div class="modal-box max-w-3xl w-11/12 max-h-[90vh] overflow-y-auto">
        <!-- modal header -->
        <div class="flex items-center justify-between mb-4 sticky top-0 bg-base-100 z-10 pb-2">
          <h3 class="text-base font-bold">{{ $t('download.task_detail') }}</h3>
          <button class="btn btn-sm btn-circle btn-ghost" @click="closeTaskDetail">
            <v-icon name="ri-close-line" class="w-4 h-4"></v-icon>
          </button>
        </div>

        <div v-if="selectedTaskId" class="space-y-6">
          <!-- task title and thumbnail -->
          <div class="flex items-start space-x-4">
            <div class="w-20 h-14 bg-base-200 rounded overflow-hidden flex-shrink-0">
              <template v-if="!selectedThumbnailLoadError && selectedTask.thumbnail">
                <img :src="selectedTask.thumbnail" class="w-full h-full object-cover" :alt="$t('download.thumbnail')"
                  @error="handleSelectedThumbnailError" />
              </template>
              <div v-else
                class="w-full h-full flex flex-col items-center justify-center text-base-content/30 bg-base-200">
                <v-icon name="ri-video-line" class="w-8 h-8 mb-1"></v-icon>
              </div>
            </div>
            <div class="flex-1">
              <div class="text-sm font-medium line-clamp-2">{{ selectedTask.title }}</div>
              <div class="badge" :class="statusBadgeClass(selectedTask.stage)">
                {{ statusText(selectedTask.stage) }}
              </div>
            </div>
          </div>

          <!-- tabs navigation -->
          <div class="tabs tabs-boxed mb-4 bg-base-200/50 p-1 rounded-lg">
            <a class="tab text-xs" :class="{ 'tab-active': activeTab === 'info' }" @click="activeTab = 'info'">
              <v-icon name="ri-information-line" class="w-3.5 h-3.5 mr-1.5"></v-icon>
              {{ $t('download.task_info') }}
            </a>
            <a class="tab text-xs" :class="{ 'tab-active': activeTab === 'download' }" @click="activeTab = 'download'">
              <v-icon name="ri-download-cloud-line" class="w-3.5 h-3.5 mr-1.5"></v-icon>
              {{ $t('download.download_info') }}
            </a>
            <a class="tab text-xs" :class="{ 'tab-active': activeTab === 'settings' }" @click="activeTab = 'settings'">
              <v-icon name="ri-settings-3-line" class="w-3.5 h-3.5 mr-1.5"></v-icon>
              {{ $t('download.download_settings') }}
            </a>
            <a class="tab text-xs" :class="{ 'tab-active': activeTab === 'output' }" @click="activeTab = 'output'">
              <v-icon name="ri-file-list-line" class="w-3.5 h-3.5 mr-1.5"></v-icon>
              {{ $t('download.output_files') }}
            </a>
          </div>

          <!-- content -->
          <!-- base info card -->
          <div v-if="activeTab === 'info' && selectedTask" class="space-y-4">
            <div class="card bg-base-200">
              <div class="card-body p-4 ">
                <div class="grid grid-cols-2 gap-3 text-xs">
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.source') }}</div>
                    <div class="font-medium">{{ selectedTask.source }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.uploader') }}</div>
                    <div class="font-medium">{{ selectedTask.uploader }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.extractor') }}</div>
                    <div class="font-medium">{{ selectedTask.extractor }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.format') }}</div>
                    <div class="font-medium">{{ selectedTask.format }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.duration') }}</div>
                    <div class="font-medium">{{ formatDuration(selectedTask.duration) }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.file_size') }}</div>
                    <div class="font-medium">{{ formatFileSize(selectedTask.fileSize) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- download info card -->
          <div v-if="activeTab === 'download' && selectedTask" class="space-y-4">
            <div class="card bg-base-200">
              <div class="card-body p-4">
                <div class="grid grid-cols-2 gap-3 text-xs">
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.speed') }}</div>
                    <div class="font-medium">{{ selectedTask.speed }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.estimated_time') }}</div>
                    <div
                      v-if="selectedTask.stage == 'downloading' || selectedTask.stage == 'translating' || selectedTask.stage == 'embedding'"
                      class="font-medium">{{ selectedTask.estimatedTime }}</div>
                    <div v-else class="font-medium">{{ selectedTask.stage }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.created_at') }}</div>
                    <div class="font-medium">{{ formatDateTime(selectedTask.createdAt) }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.updated_at') }}</div>
                    <div class="font-medium">{{ formatDateTime(selectedTask.updatedAt) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- download settings card -->
          <div v-if="activeTab === 'settings' && selectedTask" class="space-y-4">
            <div class="card bg-base-200">
              <div class="card-body p-4">
                <div class="grid grid-cols-2 gap-3 text-xs">
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.resolution') }}</div>
                    <div class="font-medium">{{ selectedTask.resolution }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.sub_langs') }}</div>
                    <div class="font-medium">{{ selectedTask.subLangsText }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.translate_to') }}</div>
                    <div class="font-medium">{{ selectedTask.translateTo }}</div>
                  </div>
                  <div>
                    <div class="text-xs text-base-content/70">{{ $t('download.subtitle_style') }}</div>
                    <div class="font-medium">{{ selectedTask.subtitleStyle }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- output files card -->
          <div v-if="activeTab === 'output' && selectedTask" class="space-y-4">
            <div class="card bg-base-200">
              <div class="card-body p-4">
                <div class="p-3 bg-base-300/50 rounded-lg flex items-center justify-between mb-3">
                  <div class="text-xs text-base-content/70">
                    <v-icon name="ri-folder-line" class="w-3.5 h-3.5 mr-1"></v-icon>
                    {{ selectedTask.outputDir }}
                  </div>
                  <button v-if="selectedTask.outputDir" class="btn btn-xs btn-outline"
                    @click="openDirectory(selectedTask.outputDir)">
                    <v-icon name="ri-folder-open-line" class="w-3.5 h-3.5 mr-1"></v-icon>
                    {{ $t('download.open_folder') }}
                  </button>
                </div>

                <!-- output files list -->
                <div v-if="selectedTask.allFiles && selectedTask.allFiles.length > 0">
                  <div v-for="(file, index) in selectedTask.allFiles" :key="index"
                    class="p-3 bg-base-300/50 rounded-lg flex items-center justify-between mb-2">
                    <div class="flex-1 min-w-0">
                      <div class="text-xs text-base-content/70 flex items-center">
                        <div class="w-4 h-4 rounded-full mr-1.5 flex items-center justify-center" :class="{
                          'bg-primary/10': getFileTypeLabel(file).includes('Video File'),
                          'bg-secondary/10': getFileTypeLabel(file).includes('Audio File'),
                          'bg-accent/10': getFileTypeLabel(file).includes('Subtitle File')
                        }">
                          <v-icon :name="getFileIcon(file)" class="w-3 h-3" :class="{
                            'text-primary': getFileTypeLabel(file).includes('Video File'),
                            'text-secondary': getFileTypeLabel(file).includes('Audio File'),
                            'text-accent': getFileTypeLabel(file).includes('Subtitle File')
                          }"></v-icon>
                        </div>
                        {{ getFileName(file) }}
                      </div>

                    </div>
                    <div class="flex-shrink-0 ml-2">
                      <button class="btn btn-xs btn-outline" @click="copyToClipboard(file)">
                        <v-icon name="ri-file-copy-line" class="w-3.5 h-3.5 mr-1"></v-icon>
                        {{ $t('download.copy_name') }}
                      </button>
                    </div>
                  </div>
                </div>
                <div v-else class="text-center text-base-content/50 py-2">
                  {{ $t('download.no_output_files') }}
                </div>
              </div>
            </div>

          </div>
        </div>

        <!-- operation buttons -->
        <div v-if="selectedTask" class="flex justify-end space-x-4 mt-4">
          <button v-if="selectedTask.outputDir" class="btn btn-sm btn-outline"
            @click="openDirectory(selectedTask.outputDir)">
            <v-icon name="ri-folder-open-line" class="w-3.5 h-3.5 mr-1"></v-icon>
            {{ $t('download.open_folder') }}
          </button>
          <button class="btn btn-sm btn-outline" @click="copyToClipboard(selectedTask.url)">
            <v-icon name="ri-file-copy-line" class="w-4 h-4 mr-1"></v-icon>
            {{ $t('download.copy_url') }}
          </button>
          <button v-if="selectedTask.stage === 'paused'" class="btn btn-sm btn-primary"
            @click="resumeTask(selectedTask.id)">
            <v-icon name="ri-play-line" class="w-4 h-4 mr-1"></v-icon>
            {{ $t('download.start_continue') }}
          </button>
          <button v-if="selectedTask.stage === 'downloading'" class="btn btn-sm btn-warning"
            @click="pauseTask(selectedTask.id)">
            <v-icon name="ri-pause-line" class="w-4 h-4 mr-1"></v-icon>
            {{ $t('download.pause') }}
          </button>
          <button class="btn btn-sm btn-primary" @click="refreshTasks()">
            <v-icon name="ri-refresh-line" class="w-4 h-4 mr-1"></v-icon>
            {{ $t('download.refresh') }}
          </button>
        </div>
      </div>
    </div>

    <!-- click background to close modal -->
    <div class="modal-backdrop" @click="closeTaskDetail"></div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListTasks, DeleteTask } from 'wailsjs/go/api/DowntasksAPI'
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useDtStore } from '@/handlers/downtasks'
import VideoDownloadModal from '@/components/modal/VideoDownloadModal.vue'
import { useLoggerStore } from '@/stores/logger'

// i18n
const { t } = useI18n()
// logger
const logger = useLoggerStore()

// base
const tasks = ref([])
const selectedTaskId = ref(null)
// task detail modal
const activeTab = ref('info')
const showDownloadModal = ref(false)
// filter
const filterStage = ref('all')
// pagination
const currentPage = ref(1)
const pageSize = ref(5)



// const stages
const stages = computed(() => [
  { id: 'downloading', label: t('download.downloading'), icon: 'ri-download-cloud-line', description: t('download.downloading_desc') },
  { id: 'translating', label: t('download.translating'), icon: 'ri-translate-2', description: t('download.translating_desc') },
  { id: 'embedding', label: t('download.embedding'), icon: 'ri-movie-line', description: t('download.embedding_desc') },
  { id: 'completed', label: t('download.completed'), icon: 'ri-checkbox-circle-line', description: t('download.completed_desc') }
])

// all stages
const allStages = computed(() => [
  { id: 'initializing', label: t('download.initializing'), icon: 'ri-loader-4-line', description: t('download.initializing_desc') },
  ...stages.value,
  { id: 'failed', label: t('download.failed'), icon: 'ri-error-warning-line', description: t('download.failed_desc') },
  { id: 'cancelled', label: t('download.cancelled'), icon: 'ri-stop-circle-line', description: t('download.cancelled_desc') }
])

// filter options
const filterOptions = computed(() => [
  { id: 'all', label: t('download.all') },
  ...allStages.value
])

// computed processing tasks
const processingTasks = computed(() => {
  return tasks.value.filter(task => task.stage === 'downloading' || task.stage === 'translating' || task.stage === 'embedding')
})

// computed filtered tasks
const filteredTasks = computed(() => {
  if (filterStage.value === 'all') return tasks.value
  return tasks.value.filter(task => task.stage === filterStage.value)
})

// computed paginated tasks
const paginatedTasks = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredTasks.value.slice(start, end)
})

// computed selected task
const selectedTask = computed(() => {
  return tasks.value.find(task => task.id === selectedTaskId.value)
})

// computed active task
const activeTask = computed(() => {
  // if no processing tasks and no manually selected task, return null
  if (processingTasks.value.length === 0 && !activeTaskId.value) return null

  // if manually selected task, find it from all tasks
  if (activeTaskId.value) {
    const found = tasks.value.find(task => task.id === activeTaskId.value)
    if (found) return found
  }

  // otherwise use the first processing task
  return processingTasks.value[0]
})

const toggleFilter = (stage) => {
  filterStage.value = stage
  document.activeElement.blur()
}

const currentStageInfo = computed(() => {
  return allStages.value.find(stage => stage.id === activeTask.value?.stage) || {}
})

// # tasks:tasks -> filterTasks -> paginatedTasks
// tasks: tasks
const refreshTasks = async () => {
  try {
    const response = await ListTasks()
    if (response.success) {
      const tasksData = JSON.parse(response.data)
      tasks.value = tasksData.map(task => {
        return {
          // base info
          id: task.id,
          // status
          stage: task.stage,
          stageInfo: task.stageInfo || '',
          error: task.error || '',

          // core metadata
          source: task.url ? new URL(task.url).hostname : t('download.unknown_source'), // calculate in  frontend
          extractor: task.extractor || '',
          title: task.title || t('download.unknown_video'),
          thumbnail: task.thumbnail ? (task.thumbnail.startsWith('http:') ? task.thumbnail.replace('http:', 'https:') : task.thumbnail) : '',
          url: task.url || '',
          formatId: task.formatId || '',
          resolution: task.resolution || '',
          uploader: task.uploader || '',
          duration: task.duration,
          fileSize: task.fileSize || 0,
          format: task.format || '',

          // progress
          percentage: task.percentage || 0,
          speed: task.speed || 0,
          estimatedTime: formatEstimatedTime(task.estimatedTime) || '',

          // timestamp
          createdAt: task.createdAt ? new Date(task.createdAt * 1000) : new Date(),
          updatedAt: task.updatedAt ? new Date(task.updatedAt * 1000) : new Date(),

          // file storage
          outputDir: task.outputDir || '',
          videoFiles: task.videoFiles || [],
          subtitleFiles: task.subtitleFiles || [],
          allDownloadedFiles: task.allDownloadedFiles || [],
          translatedSubs: task.translatedSubs || [],
          embeddedVideoFiles: task.embeddedVideoFiles || [],
          allFiles: task.allFiles || [],

          // requested params
          downloadSubs: task.downloadSubs,
          subLangs: task.subLangs || [],
          subLangsText: task.subLangs && task.subLangs.length > 0
            ? task.subLangs.map(lang => lang).join(', ')
            : t('download.none_content'),
          subFormat: task.subFormat || t('download.none_content'),
          translateTo: task.translateTo
            ? task.translateTo
            : t('download.none_content'),
          subtitleStyle: task.subtitleStyle || t('download.none_content'),

          // additional info
          thumbnailLoadError: false,
        }
      })
    } else {
      $dialog.error({
        title: t('common.error'),
        content: response.msg
      })
    }
  } catch (error) {
    $dialog.error({
      title: t('common.error'),
      content: error.message
    })
  }
}

const openDownloadModal = () => {
  showDownloadModal.value = true
}

const onDownloadStarted = async (taskId) => {
  // refresh tasks after download started
  await refreshTasks()

  // set new created task as active task
  activeTaskId.value = taskId

  // if no specified task found, try to set the first processing task as active task
  if (!tasks.value.find(task => task.id === taskId)) {
    const processingTask = tasks.value.find(task => task.stage === 'downloading' || task.stage === 'translating' || task.stage === 'embedding')
    if (processingTask) {
      activeTaskId.value = processingTask.id
    }
  }
}

const statusText = (stage) => {
  const statusMap = {
    'initializing': t('download.initializing'),
    'downloading': t('download.downloading'),
    'paused': t('download.paused'),
    'translating': t('download.translating'),
    'embedding': t('download.embedding'),
    'completed': t('download.completed'),
    'failed': t('download.failed'),
    'cancelled': t('download.cancelled')
  }

  return statusMap[stage] || t('download.unknown_status')
}


const statusBadgeClass = (stage) => {
  const colorMap = {
    'initializing': 'badge-info',
    'downloading': 'badge-primary',
    'paused': 'badge-warning',
    'translating': 'badge-primary',
    'embedding': 'badge-primary',
    'completed': 'badge-success',
    'failed': 'badge-error',
    'cancelled': 'badge-error'
  }

  return colorMap[stage] || 'badge-ghost'
}

const statusBarClass = (stage) => {
  const colorMap = {
    'initializing': 'bg-info',
    'downloading': 'bg-primary',
    'paused': 'bg-warning',
    'translating': 'bg-primary',
    'embedding': 'bg-primary',
    'completed': 'bg-success',
    'failed': 'bg-error',
    'cancelled': 'bg-error'
  }

  return colorMap[stage] || 'bg-base-content'
}

const isStageCompleted = (stage) => {
  if (stage === 'completed' || stage === 'failed' || stage === 'cancelled') {
    return true
  }
  return false
}

const pauseTask = async (taskId) => {
  try {
    // 暂时保留注释，等待后端实现
    // await PauseTask(taskId)
    console.log('暂停任务:', taskId)

    // 刷新任务列表
    refreshTasks()
  } catch (error) {
    console.error('暂停任务失败', error)
  }
}

const resumeTask = async (taskId) => {
  try {
    // 暂时保留注释，等待后端实现
    // await ResumeTask(taskId)
    console.log('恢复任务:', taskId)

    // 刷新任务列表
    refreshTasks()
  } catch (error) {
    console.error('恢复任务失败', error)
  }
}

const deleteTask = async (taskId) => {
  try {
    const response = await DeleteTask(taskId)
    if (!response.success) {
      $dialog.error({
        title: t('common.error'),
        content: response.msg
      })
      return
    }

    $message.success(t('common.delete_success'))

    // If the deleted task is the currently selected task, close the detail drawer
    if (selectedTask.value && selectedTask.value.id === taskId) {
      closeDetailDrawer()
    }

    // refresh tasks
    refreshTasks()
  } catch (error) {
    $dialog.error({
      title: t('common.error'),
      content: error.message
    })
  }
}

const deleteTaskWithConfirm = (taskId) => {
  if (confirm(t('common.delete_confirm'))) {
    deleteTask(taskId)
  }
}

const openDirectory = async (path) => {
  OpenDirectory(path)
}

const showTaskDetail = (task) => {
  // refresh task first
  selectedTaskId.value = task.id
  // default show info tab
  activeTab.value = 'info'
}

// close task detail
const closeTaskDetail = () => {
  selectedTaskId.value = null
}

const showError = (task) => {
  $dialog.error({
    title: t('common.error'),
    content: task.error,
  })
}

const formatDateTime = (timestamp) => {
  if (!timestamp) return '-'

  const date = new Date(timestamp)
  return date.toLocaleString()
}

const formatFileSize = (size) => {
  if (!size) return t('common.unknown_size')

  // ensure size is number type
  const numSize = Number(size)
  if (isNaN(numSize)) return t('common.unknown_size')

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let formattedSize = numSize

  while (formattedSize >= 1024 && i < units.length - 1) {
    formattedSize /= 1024
    i++
  }

  return `${formattedSize.toFixed(2)} ${units[i]}`
}

const formatDuration = (seconds) => {
  if (!seconds) return t('common.unknown_duration')

  // convert float number to integer seconds
  const totalSeconds = Math.floor(seconds)

  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const remainingSeconds = totalSeconds % 60

  // format as HH:MM:SS
  const formattedHours = hours > 0 ? `${hours}:` : ''
  const formattedMinutes = hours > 0 ? minutes.toString().padStart(2, '0') : minutes.toString()
  const formattedSeconds = remainingSeconds.toString().padStart(2, '0')

  return `${formattedHours}${formattedMinutes}:${formattedSeconds}`
}

const formatEstimatedTime = (eta) => {
  if (String(eta).includes('.')) {
    return String(eta).split('.')[0]
  }

  return eta
}

// get file icon
const getFileIcon = (filePath) => {
  if (!filePath) return 'ri-file-unknow-line'

  const ext = filePath.split('.').pop().toLowerCase()

  if (['mp4', 'webm', 'mkv', 'avi', 'mov'].includes(ext)) {
    return 'ri-video-line'
  } else if (['mp3', 'aac', 'wav', 'm4a'].includes(ext)) {
    return 'ri-file-music-line'
  } else if (['srt', 'vtt', 'ass'].includes(ext)) {
    return 'ri-file-text-line'
  } else if (['jpg', 'jpeg', 'png', 'webp'].includes(ext)) {
    return 'ri-image-line'
  } else {
    return 'ri-file-line'
  }
}

// get file type label
const getFileTypeLabel = (filePath) => {
  if (!filePath) return 'Unknown File'

  const ext = filePath.split('.').pop().toLowerCase()

  // check if file name contains language code (e.g. .en.srt, .zh-CN.vtt)
  const langMatch = filePath.match(/\.([a-z]{2}(-[A-Z]{2})?)\.([a-z]+)$/)

  if (['mp4', 'webm', 'mkv', 'avi', 'mov'].includes(ext)) {
    return 'Video File'
  } else if (['mp3', 'aac', 'wav', 'm4a'].includes(ext)) {
    return 'Audio File'
  } else if (['srt', 'vtt', 'ass', 'ssa'].includes(ext)) {
    // if language code is found, add it to the label
    if (langMatch && langMatch[1]) {
      return `Subtitle File (${langMatch[1].toUpperCase()})`
    }
    return 'Subtitle File'
  } else {
    return 'Other File'
  }
}

// get file name (remove path)
const getFileName = (filePath) => {
  if (!filePath) return ''

  // 统一处理路径分隔符（Windows 和 Unix）
  const normalizedPath = filePath.replace(/\\/g, '/')
  const fileName = normalizedPath.split('/').pop()

  try {
    // 尝试解码可能的 URL 编码
    return decodeURIComponent(fileName)
  } catch (e) {
    // 如果解码失败，返回原始文件名
    return fileName
  }
}

// copy to clipboard
const copyToClipboard = (toCopy) => {
  if (!toCopy) return

  navigator.clipboard.writeText(toCopy)
    .then(() => {
      $message.success(t('common.copy_success'))
    })
    .catch(err => {
      $message.error(t('common.copy_failed'), err)
    })
}

// 添加缩略图加载状态
const thumbnailLoadError = ref(false)
const selectedThumbnailLoadError = ref(false)

// handle thumbnail error
const handleThumbnailError = (e) => {
  // mark thumbnail load error
  logger.error('Thumbnail load error', activeTask.value?.thumbnail)

  thumbnailLoadError.value = true
}

const handleTaskThumbnailError = (task) => {
  // mark thumbnail load error
  logger.error('Thumbnail load error', task.value?.thumbnail)

  task.thumbnailLoadError = true
}

// handle selected thumbnail error
const handleSelectedThumbnailError = (e) => {
  // mark thumbnail load error
  logger.error('Thumbnail load error', selectedTask.value?.thumbnail)

  selectedThumbnailLoadError.value = true
}

// update task status (from WebSocket progress update)
const updateTaskFromProgress = (progress) => {
  // find corresponding task
  const taskIndex = tasks.value.findIndex(task => task.id === progress.id)
  if (taskIndex === -1) return

  // update task status
  const task = tasks.value[taskIndex]
  task.stage = progress.stage
  task.percentage = progress.percentage
  task.speed = progress.speed
  task.estimatedTime = progress.estimatedTime

  // update task list
  tasks.value[taskIndex] = { ...task }
}

const handleSignal = (signal) => {
  if (signal.refresh) {
    refreshTasks()
  }
}

// lifecycle hooks
onMounted(() => {
  // 注册信号回调
  const dtStore = useDtStore()
  dtStore.registerProgressCallback(updateTaskFromProgress)
  dtStore.registerSignalCallback(handleSignal)

  // get tasks list (only once)
  refreshTasks()

  // cleanup function
  onUnmounted(() => {
    dtStore.unregisterProgressCallback(updateTaskFromProgress)
    dtStore.unregisterSignalCallback(handleSignal)
  })
})

// add new reactive variable to store current active task ID
const activeTaskId = ref(null)

// switch current displayed active task
const switchActiveTask = (task) => {
  activeTaskId.value = task.id
  // reset task thumbnail load error
  task.thumbnailLoadError = false
}

watch(activeTaskId, (newId) => {
  if (newId) {
    refreshTasks()
    thumbnailLoadError.value = false
  }
})

watch(selectedTaskId, (newId) => {
  if (newId) {
    selectedThumbnailLoadError.value = false
  }
})

watch([showDownloadModal], ([newShowDownload], [oldShowDownload]) => {
  // 当下载模态框从打开状态变为关闭状态时刷新任务列表
  if (oldShowDownload && !newShowDownload){
    refreshTasks()
  }
})

// filter stages based on task info
const filteredStages = (task) => {
  if (!task) return []

  // default all stages from download to completion
  const result = [stages.value[0], stages.value[stages.value.length - 1]]

  if (task.type == 'custom') {
    // if has subtitle language, add subtitle processing stage
    if (task.translateTo && task.translateTo.toLowerCase() !== 'none') {
      result.splice(1, 0, stages.value.find(s => s.id === 'translating'))
    }

    // if has video embedding style (non-default), add video embedding stage
    if (task.subtitleStyle && task.subtitleStyle !== 'default') {
      // ensure correct order of insertion
      const subtitleIndex = result.findIndex(s => s.id === 'translating')
      if (subtitleIndex !== -1) {
        result.splice(subtitleIndex + 1, 0, stages.value.find(s => s.id === 'embedding'))
      } else {
        result.splice(1, 0, stages.value.find(s => s.id === 'embedding'))
      }
    }
  }


  return result
}
</script>

<style scoped>
.pipeline-node {
  width: 1.75rem;
  height: 1.75rem;
  @apply flex items-center justify-center rounded-full;
}

/* 添加平滑过渡效果 */
.pipeline-line {
  height: 2px;
  @apply bg-base-300 transition-all duration-300;
}

.pipeline-line.active {
  @apply bg-primary;
}

/* add refined hover effect for cards */
.card {
  @apply transition-all duration-200;
}

/* .card:hover {
  @apply shadow-md;
  transform: translateY(-1px);
} */

/* define text size variables */
:root {
  --text-2xs: 0.65rem;
}

.text-2xs {
  font-size: var(--text-2xs);
  line-height: 1.2;
}

/* add refined progress bar animation */
.progress-bar {
  @apply transition-all duration-300 ease-in-out;
}
</style>