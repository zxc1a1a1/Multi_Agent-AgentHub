import { useState, useMemo } from 'react'
import { Eye, Code, Maximize2, Minimize2 } from 'lucide-react'
import type { WebPreviewBlock } from '../types'

interface Props {
  block: WebPreviewBlock
}

type TabMode = 'preview' | 'source'

export default function WebPreview({ block }: Props) {
  const [tab, setTab] = useState<TabMode>('preview')
  const [expanded, setExpanded] = useState(false)
  const html = block.html || ''
  const lineCount = html ? html.split(/\r?\n/).length : 0

  const sandboxPermissions = useMemo(() => {
    // Strict sandbox: no scripts, no same-origin, no popups, no top-level navigation.
    // No allow-scripts — prevents XSS execution in previewed HTML.
    // allow-same-origin is NOT included — prevents access to Gateway cookies/tokens.
    return ''
  }, [])

  const srcDoc = useMemo(() => {
    if (!html) return ''
    // Inject CSP meta to block script execution as defense-in-depth.
    const cspMeta =
      '<meta http-equiv="Content-Security-Policy" content="default-src \'none\'; style-src \'unsafe-inline\'">'
    const headIndex = html.search(/<head[^>]*>/i)
    if (headIndex !== -1) {
      return html.replace(/<head[^>]*>/i, (match) => match + cspMeta)
    }
    // No <head> tag — prepend before <html> or at the start.
    return cspMeta + html
  }, [html])

  const maxHeight = expanded ? 'max-h-[80vh]' : 'max-h-72'

  return (
    <div className={`w-full rounded-lg border border-teal-200 bg-teal-50 overflow-hidden mt-2 ${expanded ? 'fixed inset-4 z-50 bg-white shadow-2xl' : ''}`}>
      {/* Header bar */}
      <div className="px-3 py-2 border-b border-teal-200 bg-teal-100 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-teal-800">Web Preview</span>
          <div className="flex rounded-md overflow-hidden border border-teal-300">
            <button
              onClick={() => setTab('preview')}
              className={`px-2 py-0.5 text-xs flex items-center gap-1 ${
                tab === 'preview' ? 'bg-teal-500 text-white' : 'bg-white text-teal-700 hover:bg-teal-50'
              }`}
            >
              <Eye className="w-3 h-3" />
              Preview
            </button>
            <button
              onClick={() => setTab('source')}
              className={`px-2 py-0.5 text-xs flex items-center gap-1 ${
                tab === 'source' ? 'bg-teal-500 text-white' : 'bg-white text-teal-700 hover:bg-teal-50'
              }`}
            >
              <Code className="w-3 h-3" />
              Source
            </button>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-teal-700">{lineCount} lines</span>
          <button
            onClick={() => setExpanded(!expanded)}
            className="p-0.5 rounded hover:bg-teal-200 transition-colors text-teal-700"
            title={expanded ? 'Collapse' : 'Expand'}
          >
            {expanded ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
        </div>
      </div>

      {/* Content area */}
      <div className={`${maxHeight} overflow-auto`}>
        {tab === 'preview' ? (
          html ? (
            <iframe
              className="w-full h-full min-h-[200px] border-0"
              srcDoc={srcDoc}
              sandbox={sandboxPermissions}
              title={block.title || 'Web Preview'}
            />
          ) : (
            <div className="flex items-center justify-center h-32 text-teal-400 text-sm">
              No HTML to preview
            </div>
          )
        ) : (
          <pre className="m-0 p-3 whitespace-pre-wrap break-all text-xs text-teal-900 font-mono">
            <code>{html || '<!-- No HTML to preview -->'}</code>
          </pre>
        )}
      </div>
    </div>
  )
}
