export type AGUIEvent = {
  type: string
  runId?: string
  messageId?: string
  role?: string
  delta?: string
  surface?: A2UISurface
  error?: Record<string, unknown>
}

export type A2UISurface = {
  id: string
  title?: string
  components: A2UIComponent[]
}

export type A2UIComponent = {
  type: string
  id: string
  props?: Record<string, unknown>
  children?: A2UIComponent[]
}
