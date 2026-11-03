import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchArtifactContent } from '../../services/api'

/** 256KB：超过即拒绝渲染，避免把超大文档塞进 iframe。 */
export const SANDBOX_HTML_MAX_BYTES = 256 * 1024

/** 拉取 artifact 的超时；超时后展示可重试的错误卡片。 */
export const SANDBOX_HTML_TIMEOUT_MS = 10_000

/** 子文档的内联 CSP；connect-src 'none' 阻断 fetch/XHR/WebSocket。 */
export const SANDBOX_HTML_CSP =
  "default-src 'none'; img-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'none'; frame-src 'none'; form-action 'none'; base-uri 'none'"

const MAX_MESSAGE_BYTES = 16 * 1024

/**
 * 把 CSP meta 插到 <head> 最前面，保证先于任何脚本生效。
 * 不依赖 artifact 自带的 CSP，也不会覆盖它（多个 CSP 取交集）。
 */
export function injectSandboxCSP(html: string): string {
  const meta = `<meta http-equiv="Content-Security-Policy" content="${SANDBOX_HTML_CSP}">`
  const head = /<head[^>]*>/i.exec(html)
  if (head) {
    const at = head.index + head[0].length
    return html.slice(0, at) + meta + html.slice(at)
  }
  const root = /<html[^>]*>/i.exec(html)
  if (root) {
    const at = root.index + root[0].length
    return html.slice(0, at) + `<head>${meta}</head>` + html.slice(at)
  }
  return `<!doctype html><html><head>${meta}</head><body>${html}</body></html>`
}

function isSafeMessageData(data: unknown): boolean {
  if (data === null || typeof data !== 'object') return false
  try {
    return JSON.stringify(data).length <= MAX_MESSAGE_BYTES
  } catch {
    return false
  }
}

type Props = {
  artifactId: string
  title?: string
  onMessage?: (data: unknown) => void
}

type State =
  | { status: 'loading' }
  | { status: 'ready'; html: string }
  | { status: 'error'; reason: string }

export default function SandboxedHtml({ artifactId, title, onMessage }: Props) {
  const [state, setState] = useState<State>({ status: 'loading' })
  const [attempt, setAttempt] = useState(0)
  const frameRef = useRef<HTMLIFrameElement | null>(null)

  useEffect(() => {
    let cancelled = false
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), SANDBOX_HTML_TIMEOUT_MS)
    setState({ status: 'loading' })
    fetchArtifactContent(artifactId, controller.signal)
      .then(async (blob) => {
        if (cancelled) return
        if (blob.size > SANDBOX_HTML_MAX_BYTES) {
          setState({ status: 'error', reason: 'artifact_too_large' })
          return
        }
        if (blob.type && !/text\/html|application\/xhtml\+xml/i.test(blob.type)) {
          setState({ status: 'error', reason: 'unsupported_content_type' })
          return
        }
        const html = await blob.text()
        if (cancelled) return
        if (html.length > SANDBOX_HTML_MAX_BYTES) {
          setState({ status: 'error', reason: 'artifact_too_large' })
          return
        }
        setState({ status: 'ready', html })
      })
      .catch((err: unknown) => {
        if (cancelled) return
        const name = err instanceof Error ? err.name : ''
        setState({ status: 'error', reason: name === 'AbortError' ? 'timeout' : 'fetch_failed' })
      })
      .finally(() => window.clearTimeout(timer))
    return () => {
      cancelled = true
      controller.abort()
      window.clearTimeout(timer)
    }
  }, [artifactId, attempt])

  useEffect(() => {
    const onWindowMessage = (event: MessageEvent) => {
      if (event.source !== frameRef.current?.contentWindow) return
      if (!isSafeMessageData(event.data)) return
      onMessage?.(event.data)
    }
    window.addEventListener('message', onWindowMessage)
    return () => window.removeEventListener('message', onWindowMessage)
  }, [onMessage])

  const retry = useCallback(() => setAttempt((value) => value + 1), [])

  if (state.status === 'loading') {
    return (
      <div className="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-600" role="status">
        Loading sandboxed HTML artifact…
      </div>
    )
  }

  if (state.status === 'error') {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" role="alert">
        <div className="font-medium">Sandboxed HTML unavailable</div>
        <div className="mt-1 text-xs">reason: {state.reason}</div>
        <button type="button" className="mt-2 rounded border border-amber-300 px-2 py-1 text-xs" onClick={retry}>
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
      <div className="flex items-center justify-between border-b border-gray-100 px-3 py-1.5 text-xs text-gray-500">
        <span className="font-medium">Sandboxed HTML</span>
        <span className="truncate">{title ?? artifactId}</span>
      </div>
      <iframe
        ref={frameRef}
        title={title ?? `Sandboxed artifact ${artifactId}`}
        sandbox="allow-scripts"
        referrerPolicy="no-referrer"
        srcDoc={injectSandboxCSP(state.html)}
        className="h-96 w-full border-0"
      />
    </div>
  )
}
