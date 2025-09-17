import { reactive } from 'vue'

const state = reactive({
  ribbonVisible: true,
  inspectorVisible: true,
  ribbonWidth: 160,
  inspectorWidth: 320,
})

const actions = {
  toggleRibbon() {
    state.ribbonVisible = !state.ribbonVisible
  },
  toggleInspector() {
    state.inspectorVisible = !state.inspectorVisible
  },
  showFull() {
    state.ribbonVisible = true
    state.inspectorVisible = true
  },
  showFoldedRibbon() {
    state.ribbonVisible = false
    state.inspectorVisible = true
  },
  showFoldedRibbonAndInspector() {
    state.ribbonVisible = false
    state.inspectorVisible = false
  }
}

export default function useLayoutStore() {
  return {
    get ribbonVisible() { return state.ribbonVisible },
    set ribbonVisible(v) { state.ribbonVisible = !!v },
    get inspectorVisible() { return state.inspectorVisible },
    set inspectorVisible(v) { state.inspectorVisible = !!v },
    get ribbonWidth() { return state.ribbonWidth },
    set ribbonWidth(v) { state.ribbonWidth = Number(v) || 48 },
    get inspectorWidth() { return state.inspectorWidth },
    set inspectorWidth(v) { state.inspectorWidth = Number(v) || 320 },

    toggleRibbon: actions.toggleRibbon,
    toggleInspector: actions.toggleInspector,
    showFull: actions.showFull,
    showFoldedRibbon: actions.showFoldedRibbon,
    showFoldedRibbonAndInspector: actions.showFoldedRibbonAndInspector,
  }
}
