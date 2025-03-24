import { createApp, h, ref } from 'vue'
import { i18nGlobal } from '@/utils/i18n.js'
import usePreferencesStore from 'stores/preferences.js'

// =============== Toast/Message 组件 ===============
const ToastComponent = {
  name: 'DaisyToast',
  props: {
    content: String,
    type: {
      type: String,
      default: 'info'
    },
    duration: {
      type: Number,
      default: 3000
    }
  },
  setup(props) {
    const prefStore = usePreferencesStore()
    
    // 根据类型获取 Toast 样式
    const getToastClass = (type) => {
      const baseClass = 'alert'
      const darkMode = prefStore.isDark
      
      switch (type) {
        case 'success':
          return `${baseClass} ${darkMode ? 'bg-green-900' : 'bg-green-100'} border-l-4 border-success`
        case 'error':
          return `${baseClass} ${darkMode ? 'bg-red-900' : 'bg-red-100'} border-l-4 border-error`
        case 'warning':
          return `${baseClass} ${darkMode ? 'bg-yellow-900' : 'bg-yellow-100'} border-l-4 border-warning`
        case 'info':
        default:
          return `${baseClass} ${darkMode ? 'bg-base-200' : 'bg-base-100'}`
      }
    }
    
    return () => h('div', {
      class: `toast toast-bottom toast-center z-50`,
    }, [
      // 使用自定义样式
      h('div', {
        class: getToastClass(props.type),
        style: {
          marginBottom: '38px',
          padding: '0.5rem 1rem',
          maxWidth: '300px',
          fontSize: '0.875rem',
          transform: 'scale(0.9)',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }
      }, [
        // 图标
        h('div', {
          style: {
            flexShrink: 0,
            transform: 'scale(0.8)'
          }
        }, [
          getIconForType(props.type)
        ]),
        // 内容
        h('span', { 
          style: {
            flex: '1',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis'
          }
        }, props.content)
      ])
    ])
  }
}

// 获取不同类型的 alert 样式
function getAlertClass(type) {
  switch (type) {
    case 'success': return 'alert-success'
    case 'error': return 'alert-error'
    case 'warning': return 'alert-warning'
    default: return 'alert-info'
  }
}

// 获取不同类型的图标
function getIconForType(type) {
  switch (type) {
    case 'success':
      return h('svg', {
        xmlns: 'http://www.w3.org/2000/svg',
        class: 'stroke-current shrink-0 h-6 w-6',
        fill: 'none',
        viewBox: '0 0 24 24'
      }, [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'stroke-width': '2',
          d: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
        })
      ])
    case 'error':
      return h('svg', {
        xmlns: 'http://www.w3.org/2000/svg',
        class: 'stroke-current shrink-0 h-6 w-6',
        fill: 'none',
        viewBox: '0 0 24 24'
      }, [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'stroke-width': '2',
          d: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z'
        })
      ])
    case 'warning':
      return h('svg', {
        xmlns: 'http://www.w3.org/2000/svg',
        class: 'stroke-current shrink-0 h-6 w-6',
        fill: 'none',
        viewBox: '0 0 24 24'
      }, [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'stroke-width': '2',
          d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z'
        })
      ])
    default: // info
      return h('svg', {
        xmlns: 'http://www.w3.org/2000/svg',
        class: 'stroke-current shrink-0 h-6 w-6',
        fill: 'none',
        viewBox: '0 0 24 24'
      }, [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          'stroke-width': '2',
          d: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
        })
      ])
  }
}

// 创建并显示 toast
function showToast(content, options = {}) {
  const { type = 'info', duration = 3000 } = options
  
  // 创建容器
  const container = document.createElement('div')
  document.body.appendChild(container)
  
  let destroyed = false
  const destroy = () => {
    if (destroyed) return
    destroyed = true
    app.unmount()
    if (document.body.contains(container)) {
      document.body.removeChild(container)
    }
  }
  
  // 创建 Vue 实例
  const app = createApp({
    render() {
      return h(ToastComponent, {
        content,
        type,
        duration
      })
    }
  })
  
  // 挂载
  app.mount(container)
  
  // 定时移除
  setTimeout(destroy, duration)
  
  return {
    close: destroy
  }
}

// =============== Notification 组件 ===============
const NotificationComponent = {
  name: 'DaisyNotification',
  props: {
    title: String,
    content: String,
    type: {
      type: String,
      default: 'info'
    },
    duration: {
      type: Number,
      default: 4500
    },
    closable: {
      type: Boolean,
      default: true
    },
    onClose: Function,
    meta: [String, Object]
  },
  setup(props, { slots }) {
    const visible = ref(true)
    const prefStore = usePreferencesStore()
    
    const close = () => {
      visible.value = false
      if (props.onClose) {
        props.onClose()
      }
    }
    
    // 复制内容到剪贴板
    const copyContent = () => {
      if (props.content) {
        navigator.clipboard.writeText(props.content)
          .then(() => {
            // 可以添加一个小提示，表示复制成功
            const copyTip = document.createElement('div')
            copyTip.textContent = '已复制'
            copyTip.style.position = 'fixed'
            copyTip.style.bottom = '20px'
            copyTip.style.right = '20px'
            copyTip.style.padding = '5px 10px'
            copyTip.style.backgroundColor = 'rgba(0, 0, 0, 0.7)'
            copyTip.style.color = 'white'
            copyTip.style.borderRadius = '4px'
            copyTip.style.zIndex = '9999'
            document.body.appendChild(copyTip)
            
            setTimeout(() => {
              document.body.removeChild(copyTip)
            }, 1500)
          })
          .catch(err => console.error('复制失败:', err))
      }
    }
    
    if (props.duration > 0) {
      setTimeout(() => {
        close()
      }, props.duration)
    }
    
    // 根据类型获取卡片样式
    const getCardClass = (type) => {
      const baseClass = 'card w-96 shadow-xl'
      const darkMode = prefStore.isDark
      
      switch (type) {
        case 'success':
          return `${baseClass} ${darkMode ? 'bg-green-900' : 'bg-green-100'} border-l-4 border-success`
        case 'error':
          return `${baseClass} ${darkMode ? 'bg-red-900' : 'bg-red-100'} border-l-4 border-error`
        case 'warning':
          return `${baseClass} ${darkMode ? 'bg-yellow-900' : 'bg-yellow-100'} border-l-4 border-warning`
        case 'info':
        default:
          return `${baseClass} ${darkMode ? 'bg-base-200' : 'bg-base-100'}`
      }
    }
    
    // 处理内容中的换行符
    const renderContent = (content) => {
      if (!content) return null
      
      // 如果内容包含换行符，将其分割成多个段落
      if (typeof content === 'string' && content.includes('\n')) {
        return content.split('\n').map((paragraph, index) => 
          h('p', { 
            class: 'text-sm mt-1 whitespace-pre-wrap break-words select-text', 
            key: index 
          }, paragraph || ' ')
        )
      }
      
      // 否则作为单个段落渲染
      return h('p', { 
        class: 'text-sm mt-2 whitespace-pre-wrap break-words select-text' 
      }, content)
    }
    
    return () => visible.value ? h('div', {
      class: 'toast toast-end toast-bottom z-50 mb-4',
    }, [
      // 使用 DaisyUI 的 card 组件，根据类型设置样式
      h('div', {
        class: getCardClass(props.type),
      }, [
        h('div', { class: 'card-body py-4 px-6' }, [
          h('div', { class: 'flex justify-between items-center' }, [
            h('h2', { 
              class: `card-title text-base ${props.type === 'error' ? 'text-error' : props.type === 'warning' ? 'text-warning' : props.type === 'success' ? 'text-success' : ''}` 
            }, props.title || getDefaultTitle(props.type)),
            h('div', { class: 'flex items-center gap-2' }, [
              // 添加复制按钮
              props.content ? h('button', {
                class: 'btn btn-ghost btn-xs tooltip',
                'data-tip': i18nGlobal.t('message.copy_content'),
                onClick: copyContent
              }, [
                h('svg', {
                  xmlns: 'http://www.w3.org/2000/svg',
                  fill: 'none',
                  viewBox: '0 0 24 24',
                  'stroke-width': '1.5',
                  stroke: 'currentColor',
                  class: 'w-4 h-4'
                }, [
                  h('path', {
                    'stroke-linecap': 'round',
                    'stroke-linejoin': 'round',
                    d: 'M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75'
                  })
                ])
              ]) : null,
              // 关闭按钮
              props.closable ? h('button', { 
                class: 'btn btn-ghost btn-xs btn-circle',
                onClick: close
              }, '✕') : null
            ])
          ]),
          renderContent(props.content),
          props.meta ? h('div', { class: 'text-xs opacity-70 mt-2 select-text' }, props.meta) : null,
          slots.action ? h('div', { class: 'card-actions justify-end mt-3' }, slots.action()) : null
        ])
      ])
    ]) : null
  }
}

// 获取默认标题
function getDefaultTitle(type) {
  switch (type) {
    case 'success': return i18nGlobal.t('common.success')
    case 'error': return i18nGlobal.t('common.error')
    case 'warning': return i18nGlobal.t('common.warning')
    default: return i18nGlobal.t('common.info')
  }
}

// 创建并显示 notification
function showNotification(options = {}) {
    const { 
      title, 
      content, 
      type = 'info', 
      duration = 4500,
      closable = true,
      meta = '',
      action = null
    } = options
    
    // 创建容器
    const container = document.createElement('div')
    document.body.appendChild(container)
    
    let destroyed = false
    const destroy = () => {
      if (destroyed) return
      destroyed = true
      app.unmount()
      document.body.removeChild(container)
    }
    
    // 创建 Vue 实例
    const app = createApp({
      render() {
        return h(NotificationComponent, {
          title,
          content,
          type,
          duration,
          closable,
          meta,
          onClose: destroy
        }, {
          // 将 action 作为插槽传递
          action: action ? () => action(destroy) : undefined
        })
      }
    })
    
    // 挂载
    app.mount(container)
    
    return {
      close: destroy,
      destroy: destroy  // 添加 destroy 方法作为 close 的别名
    }
}

// =============== Dialog 组件 ===============
const DialogComponent = {
  name: 'DaisyDialog',
  props: {
    title: String,
    content: String,
    type: {
      type: String,
      default: 'info'
    },
    closable: {
      type: Boolean,
      default: true
    },
    maskClosable: {
      type: Boolean,
      default: true
    },
    positiveText: String,
    negativeText: String,
    onPositiveClick: Function,
    onNegativeClick: Function,
    onClose: Function
  },
  setup(props) {
    const prefStore = usePreferencesStore()
    
    // 复制内容到剪贴板
    const copyContent = () => {
      if (props.content) {
        navigator.clipboard.writeText(props.content)
          .then(() => {
            // 可以添加一个小提示，表示复制成功
            const copyTip = document.createElement('div')
            copyTip.textContent = i18nGlobal.t('message.copy_success')
            copyTip.style.position = 'fixed'
            copyTip.style.bottom = '20px'
            copyTip.style.right = '20px'
            copyTip.style.padding = '5px 10px'
            copyTip.style.backgroundColor = 'rgba(0, 0, 0, 0.7)'
            copyTip.style.color = 'white'
            copyTip.style.borderRadius = '4px'
            copyTip.style.zIndex = '9999'
            document.body.appendChild(copyTip)
            
            setTimeout(() => {
              document.body.removeChild(copyTip)
            }, 1500)
          })
          .catch(err => console.error(i18nGlobal.t('message.copy_failed')+':', err))
      }
    }
    
    // 根据类型获取对话框样式
    const getDialogClass = (type) => {
      const baseClass = 'modal-box relative'
      const darkMode = prefStore.isDark
      
      switch (type) {
        case 'success':
          return `${baseClass} ${darkMode ? 'bg-green-900' : 'bg-green-100'} border-t-4 border-success`
        case 'error':
          return `${baseClass} ${darkMode ? 'bg-red-900' : 'bg-red-100'} border-t-4 border-error`
        case 'warning':
          return `${baseClass} ${darkMode ? 'bg-yellow-900' : 'bg-yellow-100'} border-t-4 border-warning`
        case 'info':
        default:
          return `${baseClass} ${darkMode ? 'bg-base-200' : 'bg-base-100'}`
      }
    }
    
    // 处理内容中的换行符
    const renderContent = (content) => {
      if (!content) return null
      
      // 如果内容包含换行符，将其分割成多个段落
      if (typeof content === 'string' && content.includes('\n')) {
        return content.split('\n').map((paragraph, index) => 
          h('p', { 
            class: 'py-1 whitespace-pre-wrap break-words select-text', 
            key: index 
          }, paragraph || ' ')
        )
      }
      
      // 否则作为单个段落渲染
      return h('p', { 
        class: 'py-4 whitespace-pre-wrap break-words select-text' 
      }, content)
    }
    
    const handleMaskClick = (e) => {
      if (props.maskClosable && e.target.classList.contains('modal')) {
        if (props.onClose) props.onClose()
      }
    }
    
    const handlePositiveClick = () => {
      if (props.onPositiveClick) props.onPositiveClick()
    }
    
    const handleNegativeClick = () => {
      if (props.onNegativeClick) props.onNegativeClick()
    }
    
    const handleClose = () => {
      if (props.onClose) props.onClose()
    }
    
    // 使用 DaisyUI 的 modal 组件
    return () => h('div', {
      class: 'modal modal-open',
      onClick: handleMaskClick
    }, [
      h('div', { 
        class: getDialogClass(props.type)
      }, [
        h('div', { class: 'flex justify-between items-center mb-2' }, [
          h('h3', { 
            class: `font-bold text-lg ${props.type === 'error' ? 'text-error' : props.type === 'warning' ? 'text-warning' : props.type === 'success' ? 'text-success' : ''}` 
          }, props.title || getDefaultTitle(props.type)),
          h('div', { class: 'flex items-center gap-2' }, [
            // 添加复制按钮
            props.content ? h('button', {
              class: 'btn btn-ghost btn-xs tooltip',
              'data-tip': i18nGlobal.t('message.copy_content'),
              onClick: copyContent
            }, [
              h('svg', {
                xmlns: 'http://www.w3.org/2000/svg',
                fill: 'none',
                viewBox: '0 0 24 24',
                'stroke-width': '1.5',
                stroke: 'currentColor',
                class: 'w-4 h-4'
              }, [
                h('path', {
                  'stroke-linecap': 'round',
                  'stroke-linejoin': 'round',
                  d: 'M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75'
                })
              ])
            ]) : null,
            // 关闭按钮
            props.closable ? h('button', {
              class: 'btn btn-sm btn-circle',
              onClick: handleClose
            }, '✕') : null
          ])
        ]),
        renderContent(props.content),
        h('div', { class: 'modal-action' }, [
          props.negativeText ? h('button', {
            class: 'btn',
            onClick: handleNegativeClick
          }, props.negativeText) : null,
          props.positiveText ? h('button', {
            class: `btn ${getButtonClass(props.type)}`,
            onClick: handlePositiveClick
          }, props.positiveText) : null
        ])
      ])
    ])
  }
}

// 获取不同类型的按钮样式
function getButtonClass(type) {
  switch (type) {
    case 'success': return 'btn-success'
    case 'error': return 'btn-error'
    case 'warning': return 'btn-warning'
    default: return 'btn-primary'
  }
}

// 创建并显示 dialog
function showDialog(options = {}) {
  return new Promise((resolve) => {
    const {
      title,
      content,
      type = 'info',
      closable = true,
      maskClosable = true,
      positiveText,
      negativeText
    } = options
    
    // 创建容器
    const container = document.createElement('div')
    document.body.appendChild(container)
    
    let destroyed = false
    const destroy = () => {
      if (destroyed) return
      destroyed = true
      app.unmount()
      document.body.removeChild(container)
    }
    
    const onPositiveClick = () => {
      destroy()
      resolve(true)
    }
    
    const onNegativeClick = () => {
      destroy()
      resolve(false)
    }
    
    const onClose = () => {
      destroy()
      resolve(false)
    }
    
    // 创建 Vue 实例
    const app = createApp({
      render() {
        return h(DialogComponent, {
          title,
          content,
          type,
          closable,
          maskClosable,
          positiveText,
          negativeText,
          onPositiveClick,
          onNegativeClick,
          onClose
        })
      }
    })
    
    // 挂载
    app.mount(container)
  })
}

// =============== 导出 API ===============
// 创建与 Naive UI 兼容的 message API
export function setupDaisyMessage() {
  const message = {
    success(content, option = {}) {
      return showToast(content, { ...option, type: 'success' })
    },
    error(content, option = {}) {
      return showToast(content, { ...option, type: 'error' })
    },
    warning(content, option = {}) {
      return showToast(content, { ...option, type: 'warning' })
    },
    info(content, option = {}) {
      return showToast(content, { ...option, type: 'info' })
    },
    loading(content, option = {}) {
      option.duration = option.duration != null ? option.duration : 30000
      return showToast(content, { ...option, type: 'info' })
    }
  }
  
  window.$message = message
  return message
}

// 创建与 Naive UI 兼容的 notification API
export function setupDaisyNotification() {
    const notification = {
      create(contentOrOption, option = {}) {
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option })
        } else {
          return showNotification(contentOrOption)
        }
      },
      show(contentOrOption, option = {}) {  // 添加 show 方法作为 create 的别名
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option })
        } else {
          return showNotification(contentOrOption)
        }
      },
      success(contentOrOption, option = {}) {
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option, type: 'success' })
        } else {
          return showNotification({ ...contentOrOption, type: 'success' })
        }
      },
      error(contentOrOption, option = {}) {
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option, type: 'error' })
        } else {
          return showNotification({ ...contentOrOption, type: 'error' })
        }
      },
      warning(contentOrOption, option = {}) {
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option, type: 'warning' })
        } else {
          return showNotification({ ...contentOrOption, type: 'warning' })
        }
      },
      info(contentOrOption, option = {}) {
        if (typeof contentOrOption === 'string') {
          return showNotification({ content: contentOrOption, ...option, type: 'info' })
        } else {
          return showNotification({ ...contentOrOption, type: 'info' })
        }
      }
    }
    
    window.$notification = notification
    return notification
}

// 创建与 Naive UI 兼容的 dialog API
export function setupDaisyDialog() {
  const dialog = {
    success(contentOrOption, option = {}) {
      if (typeof contentOrOption === 'string') {
        return showDialog({ content: contentOrOption, ...option, type: 'success' })
      } else {
        return showDialog({ ...contentOrOption, type: 'success' })
      }
    },
    error(contentOrOption, option = {}) {
      if (typeof contentOrOption === 'string') {
        return showDialog({ content: contentOrOption, ...option, type: 'error' })
      } else {
        return showDialog({ ...contentOrOption, type: 'error' })
      }
    },
    warning(contentOrOption, option = {}) {
      if (typeof contentOrOption === 'string') {
        return showDialog({ content: contentOrOption, ...option, type: 'warning' })
      } else {
        return showDialog({ ...contentOrOption, type: 'warning' })
      }
    },
    info(contentOrOption, option = {}) {
      if (typeof contentOrOption === 'string') {
        return showDialog({ content: contentOrOption, ...option, type: 'info' })
      } else {
        return showDialog({ ...contentOrOption, type: 'info' })
      }
    },
    confirm(contentOrOption, option = {}) {
      if (typeof contentOrOption === 'string') {
        return showDialog({
          content: contentOrOption,
          ...option,
          type: 'info',
          positiveText: option.positiveText || i18nGlobal.t('common.confirm'),
          negativeText: option.negativeText || i18nGlobal.t('common.cancel')
        })
      } else {
        return showDialog({
          ...contentOrOption,
          type: 'info',
          positiveText: contentOrOption.positiveText || i18nGlobal.t('common.confirm'),
          negativeText: contentOrOption.negativeText || i18nGlobal.t('common.cancel')
        })
      }
    }
  }
  
  window.$dialog = dialog
  return dialog
}

// 一次性设置所有 DaisyUI 组件
export function setupDaisyUI() {
  setupDaisyMessage()
  setupDaisyNotification()
  setupDaisyDialog()
}