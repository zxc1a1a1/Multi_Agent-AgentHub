import type { WebPreviewBlock } from '../types'

interface Props {
  block: WebPreviewBlock
}

export default function WebPreview({ block }: Props) {
  const html = block.html || ''
  const lineCount = html ? html.split(/\r?\n/).length : 0

  return (
    <div className="w-full rounded-lg border border-teal-200 bg-teal-50 overflow-hidden mt-2">
      <div className="px-3 py-2 border-b border-teal-200 bg-teal-100 flex items-center justify-between">
        <span className="text-xs font-medium text-teal-800">Web Preview (Safe Mode)</span>
        <span className="text-xs text-teal-700">{lineCount} lines</span>
      </div>
      <div className="max-h-72 overflow-auto p-3">
        <pre className="m-0 whitespace-pre-wrap break-all text-xs text-teal-900">
          <code>{html || '<!-- No HTML to preview -->'}</code>
        </pre>
      </div>
    </div>
  )
}
