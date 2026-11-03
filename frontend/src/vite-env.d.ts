/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** R3-03: opt-in Streamdown pilot; absent/false keeps the legacy renderer. */
  readonly VITE_STREAMDOWN_ENABLED?: string
}

declare const __STREAMDOWN_ENABLED__: boolean
