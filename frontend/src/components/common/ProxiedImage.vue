<template>
    <div class="proxied-image-wrapper" :style="{ width: width, height: height }">
      <!-- 加载中状态 -->
      <div v-if="isLoading" class="loading-placeholder flex items-center justify-center w-full h-full bg-base-200 text-base-content/30">
        <v-icon name="ri-loader-4-line" class="animate-spin w-5 h-5"></v-icon>
      </div>
      <!-- 加载错误或无有效 URL 状态 -->
      <div v-else-if="hasError || !imageDataUrl" class="error-placeholder flex items-center justify-center w-full h-full bg-base-200 text-base-content/30">
        <v-icon :name="errorIcon" class="w-5 h-5"></v-icon>
      </div>
      <!-- 图片加载成功 -->
      <img
        v-else
        :src="imageDataUrl"
        :alt="alt"
        class="image-content w-full h-full object-cover"
        @error="handleImageError"
      />
    </div>
  </template>
  
  <script setup>
  import { ref, watch, onMounted } from 'vue'
  // 确保从正确的路径导入 GetImage
  import { GetImage } from 'wailsjs/go/api/UtilsAPI'
  import { useLoggerStore } from '@/stores/logger'
  
  const props = defineProps({
    // 原始图片 URL
    src: {
      type: String,
      default: ''
    },
    // 图片 alt 文本
    alt: {
      type: String,
      default: 'Image'
    },
    // 容器宽度 (CSS value, e.g., '50px', '3rem')
    width: {
      type: String,
      default: '100%'
    },
    // 容器高度 (CSS value, e.g., '50px', '3rem')
    height: {
      type: String,
      default: '100%'
    },
    // 错误时显示的图标
    errorIcon: {
      type: String,
      default: 'ri-image-line' // 默认图标
    }
  })
  
  const logger = useLoggerStore()
  
  const imageDataUrl = ref(null) // 存储最终的 Data URL
  const isLoading = ref(false)   // 加载状态
  const hasError = ref(false)    // 错误状态
  
  // 核心逻辑：获取图片 Data URL
  const fetchImageData = async (url) => {
    // 1. 检查输入 URL
    if (!url || typeof url !== 'string') {
      imageDataUrl.value = null
      isLoading.value = false
      hasError.value = true // 标记为错误，因为没有有效 URL
      logger.warn('ProxiedImage: Invalid src provided:', url)
      return
    }
  
    // 2. 如果已经是 Data URL，直接使用
    if (url.startsWith('data:image')) {
      imageDataUrl.value = url
      isLoading.value = false
      hasError.value = false
      return
    }
  
    // 3. 如果是需要代理的 URL
    isLoading.value = true
    hasError.value = false
    imageDataUrl.value = null // 清空旧数据
  
    try {
      // 确保 URL 是 https (如果需要)
      const correctedUrl = url.startsWith('http:') ? url.replace('http:', 'https:') : url;
      // 调用后端代理
      const response = await GetImage(correctedUrl)
      if (response.success) {
        const result = JSON.parse(response.data)
        if (result.base64Data && result.contentType) {
            imageDataUrl.value = `data:${result.contentType};base64,${result.base64Data}`
            hasError.value = false
        } else {
            // 后端返回的数据格式无效
             throw new Error('Invalid data received from proxy, base64Data or contentType is missing')
        }
      } else {
        // 后端返回错误
        throw new Error(response.msg)
      }
    } catch (error) {
      logger.error('ProxiedImage: Failed to fetch image via proxy for url:', url, 'Error:', error)
      hasError.value = true
      imageDataUrl.value = null // 确保错误时不显示旧图片
    } finally {
      isLoading.value = false
    }
  }
  
  // 处理原生 img 标签的 @error 事件 (理论上 Data URL 不会触发，但作为保险)
  const handleImageError = (event) => {
    logger.error('ProxiedImage: Native img error event for Data URL (should not happen):', props.src, event)
    hasError.value = true
    imageDataUrl.value = null
  }
  
  // 监听 src prop 的变化
  watch(() => props.src, (newSrc, oldSrc) => {
    if (newSrc !== oldSrc) {
      fetchImageData(newSrc)
    }
  }, { immediate: true }) // immediate: true 确保组件挂载时立即执行一次
  
  // 也可以在 onMounted 中调用，但 watch immediate 效果相同
  // onMounted(() => {
  //   fetchImageData(props.src)
  // })
  
  </script>
  
  <style scoped>
  .proxied-image-wrapper {
    display: inline-block; /* 或者 block，根据需要调整 */
    overflow: hidden; /* 隐藏可能的加载/错误状态溢出 */
    position: relative; /* 用于绝对定位内部元素（如果需要） */
  }
  
  .image-content {
    display: block; /* 避免 img 底部空隙 */
  }
  </style>