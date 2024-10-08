// define translation status
export const TranslationStatus = {
    PENDING: 'pending',
    RUNNING: 'running',
    COMPLETED: 'completed',
    ERROR: 'error',
    CANCEL_PENDING: 'cancel_pending',
    CANCELED: 'canceled'
  }
  
  export const StreamStatus = {
    STREAMING: 'streaming',
    TRANSLATION_DONE: 'translation_done',
    ERROR: 'error',
    CANCEL_PENDING: 'cancel_pending',
    CANCELED: 'canceled',
    CANCEL_ERROR: 'cancel_error'
  }
  
  export class TranslationStateManager {
    constructor(initialState = {}) {
      this.state = {
        streamStatus: StreamStatus.STREAMING,
        translationStatus: TranslationStatus.PENDING,
        translationProgress: 0,
        actionDescription: undefined,
        isCompleted: false,
        ...initialState
      }
    }
  
    updateState(newState) {
      this.state = { ...this.state, ...newState }
      this.validateState()
      this.computedIsCompleted()
    }
  
    validateState() {
      // add state validation logic here
      // for example, ensure translationProgress is between 0-100
      if (this.state.translationProgress < 0) this.state.translationProgress = 0
      if (this.state.translationProgress > 100) this.state.translationProgress = 100
  
      // ensure the consistency of the state combination
      if (this.state.streamStatus === StreamStatus.TRANSLATION_DONE) {
        this.state.translationStatus = TranslationStatus.COMPLETED
        this.state.translationProgress = 100
      }
    }
  
    computedIsCompleted() {
      if (this.state.streamStatus === StreamStatus.TRANSLATION_DONE || this.state.streamStatus === StreamStatus.CANCELED || this.state.streamStatus === StreamStatus.ERROR) {
        this.state.isCompleted = true
      } else if (this.state.streamStatus = StreamStatus.STREAMING ||  this.state.streamStatus === StreamStatus.CANCEL_PENDING ){
        this.state.isCompleted = false
      }
    }

    setRunning() {
      this.updateState({
        streamStatus: StreamStatus.STREAMING,
        translationStatus: TranslationStatus.RUNNING,
        isCompleted: false
      })
    }
  
    setCompleted() {
      this.updateState({
        streamStatus: StreamStatus.TRANSLATION_DONE,
        translationStatus: TranslationStatus.COMPLETED,
        translationProgress: 100,
        isCompleted: true
      })
    }
  
    setError(description) {
      this.updateState({
        streamStatus: StreamStatus.ERROR,
        translationStatus: TranslationStatus.ERROR,
        actionDescription: description,
        isCompleted: true
      })
    }

    getState() {
        return { ...this.state }
    }
}