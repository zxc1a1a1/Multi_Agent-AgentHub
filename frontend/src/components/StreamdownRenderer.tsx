import { Component, Suspense, lazy, type ReactNode } from 'react'

/**
 * R3-03: Streamdown is an opt-in pilot. The default build keeps the legacy
 * renderer; only VITE_STREAMDOWN_ENABLED=true loads this chunk.
 */
export const STREAMDOWN_ENABLED = __STREAMDOWN_ENABLED__ || import.meta.env.VITE_STREAMDOWN_ENABLED === 'true'

const Streamdown = lazy(async () => {
  await import('streamdown/styles.css')
  const mod = await import('streamdown')
  return { default: mod.Streamdown }
})

type BoundaryProps = { fallback: ReactNode; children: ReactNode }
type BoundaryState = { failed: boolean }

class StreamdownBoundary extends Component<BoundaryProps, BoundaryState> {
  state: BoundaryState = { failed: false }

  static getDerivedStateFromError(): BoundaryState {
    return { failed: true }
  }

  render() {
    return this.state.failed ? this.props.fallback : this.props.children
  }
}

export default function StreamdownRenderer({ content, fallback }: { content: string; fallback: ReactNode }) {
  return (
    <StreamdownBoundary fallback={fallback}>
      <Suspense fallback={fallback}>
        <Streamdown>{content}</Streamdown>
      </Suspense>
    </StreamdownBoundary>
  )
}
