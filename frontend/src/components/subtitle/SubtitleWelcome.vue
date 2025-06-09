<template>
  <div class="welcome-container">
    <!-- 欢迎页面内容 -->
    <div class="welcome-content">
      <!-- 主图标和标题 -->
      <div class="welcome-header">
        <div class="welcome-icon">
          <svg class="w-12 h-12 md:w-16 md:h-16 text-primary/60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" 
              d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2m0 0V1a1 1 0 011 1v2m0 0v14a2 2 0 01-2 2H8a2 2 0 01-2-2V6m8 0H8m0 0V4m0 2v14m8-14v14M9 12h6m-6 4h6"></path>
          </svg>
        </div>
        <h1 class="welcome-title">{{ $t('subtitle.welcome.title') }}</h1>
        <p class="welcome-subtitle">{{ $t('subtitle.welcome.description') }}</p>
      </div>

      <!-- 功能特性 -->
      <div class="features-grid">
        <div class="feature-card">
          <div class="feature-icon">
            <svg class="w-5 h-5 md:w-6 md:h-6 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
            </svg>
          </div>
          <h3 class="feature-title">{{ $t('subtitle.welcome.multi_lang_support') }}</h3>
          <p class="feature-description">{{ $t('subtitle.welcome.multi_lang_support_desc') }}</p>
        </div>

        <div class="feature-card">
          <div class="feature-icon">
            <svg class="w-5 h-5 md:w-6 md:h-6 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
            </svg>
          </div>
          <h3 class="feature-title">{{ $t('subtitle.welcome.auto_save') }}</h3>
          <p class="feature-description">{{ $t('subtitle.welcome.auto_save_desc') }}</p>
        </div>

        <div class="feature-card">
          <div class="feature-icon">
            <svg class="w-5 h-5 md:w-6 md:h-6 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10"></path>
            </svg>
          </div>
          <h3 class="feature-title">{{ $t('subtitle.welcome.multi_format_export') }}</h3>
          <p class="feature-description">{{ $t('subtitle.welcome.multi_format_export_desc') }}</p>
        </div>
      </div>

      <!-- 快速操作按钮 -->
      <div class="quick-actions">
        <button @click="$emit('open-file')" class="btn-macos-primary btn-macos-sm">
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-5l-2-2H5a2 2 0 00-2 2z"></path>
          </svg>
          {{ $t('subtitle.common.open_file') }}
        </button>
        
        <button @click="$emit('show-history')" class="btn-macos-secondary btn-macos-sm">
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
          {{ $t('subtitle.common.history') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SubtitleWelcome',
  emits: ['open-file', 'show-history']
}
</script>

<style scoped>
.welcome-container {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 0;
  padding: 1rem;
  background: linear-gradient(135deg, rgba(var(--color-base-200), 0.3) 0%, rgba(var(--color-base-300), 0.1) 100%);
  overflow-y: auto;
}

.welcome-content {
  max-width: 600px;
  width: 100%;
  text-align: center;
  animation: slideInUp 0.6s ease-out;
  padding: 1rem 0;
}

.welcome-header {
  margin-bottom: 2rem;
}

.welcome-icon {
  margin-bottom: 1rem;
  display: flex;
  justify-content: center;
}

.welcome-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: rgb(var(--color-base-content));
  margin-bottom: 0.5rem;
  line-height: 1.2;
}

.welcome-subtitle {
  font-size: 1rem;
  color: rgb(var(--color-base-content) / 0.7);
  margin-bottom: 0;
  line-height: 1.4;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.feature-card {
  background: rgb(var(--color-base-100));
  border: 1px solid rgb(var(--color-base-300));
  border-radius: 8px;
  padding: 1rem 0.75rem;
  transition: all 0.3s ease;
  cursor: default;
}

.feature-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-color: rgb(var(--color-primary) / 0.3);
}

.feature-icon {
  display: flex;
  justify-content: center;
  margin-bottom: 0.75rem;
}

.feature-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: rgb(var(--color-base-content));
  margin-bottom: 0.5rem;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.feature-description {
  font-size: 0.8rem;
  color: rgb(var(--color-base-content) / 0.6);
  line-height: 1.3;
}

.quick-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.welcome-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: rgb(var(--color-base-100) / 0.5);
  border: 1px solid rgb(var(--color-base-300) / 0.5);
  border-radius: 6px;
  backdrop-filter: blur(10px);
}

.tip-text {
  font-size: 0.8rem;
  color: rgb(var(--color-base-content) / 0.6);
  line-height: 1.3;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .welcome-container {
    padding: 0.75rem;
  }
  
  .welcome-content {
    padding: 0.5rem 0;
  }
  
  .welcome-title {
    font-size: 1.25rem;
  }
  
  .welcome-subtitle {
    font-size: 0.9rem;
  }
  
  .features-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }
  
  .feature-card {
    padding: 0.75rem;
  }
  
  .quick-actions {
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
  }
  
  .welcome-header {
    margin-bottom: 1.5rem;
  }
}

@media (max-width: 480px) {
  .welcome-container {
    padding: 0.5rem;
  }
  
  .welcome-title {
    font-size: 1.1rem;
  }
  
  .feature-title {
    font-size: 0.85rem;
  }
  
  .feature-description {
    font-size: 0.75rem;
  }
}

@media (max-height: 600px) {
  .welcome-header {
    margin-bottom: 1rem;
  }
  
  .features-grid {
    margin-bottom: 1rem;
  }
  
  .quick-actions {
    margin-bottom: 1rem;
  }
  
  .feature-card {
    padding: 0.5rem;
  }
}
</style>