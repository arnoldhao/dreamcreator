import { reactive } from 'vue'

const state = reactive({
  visible: false,
  title: 'Inspector',
  panel: null, // active panel key
  props: {},
  actions: [], // [{ key, icon, title }]
})

const actions = {
  open(panel, title = 'Inspector', props = {}) {
    state.panel = panel
    state.title = title
    state.props = props
    state.visible = true
  },
  close() {
    state.visible = false
  },
  toggle(panel, title = 'Inspector', props = {}) {
    if (state.visible && state.panel === panel) {
      state.visible = false
    } else {
      this.open(panel, title, props)
    }
  },
  setActions(list = []) {
    state.actions = Array.isArray(list) ? list : []
  },
  setVisible(v) { state.visible = !!v },
}

export default function useInspectorStore() {
  return {
    get visible() { return state.visible },
    set visible(v) { state.visible = !!v },
    get title() { return state.title },
    get panel() { return state.panel },
    get props() { return state.props },
    get actions() { return state.actions },

    open: actions.open,
    close: actions.close,
    toggle: actions.toggle,
    setActions: actions.setActions,
    setVisible: actions.setVisible,
  }
}
