import { useState, useMemo } from 'react'
import { Copy, Check, FileCode } from 'lucide-react'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import type { CodeBlock } from '../types'

interface Props {
  block: CodeBlock
}

export default function CodePreview({ block }: Props) {
  const [copied, setCopied] = useState(false)
  const code = typeof block?.code === 'string' ? block.code : ''
  const language = typeof block?.language === 'string' && block.language.trim() ? block.language : 'text'
  const filename = typeof block?.filename === 'string' && block.filename.trim() ? block.filename : 'untitled'
  const hasCode = code.trim().length > 0

  const highlighted = useMemo(() => {
    if (!hasCode) {
      return ''
    }
    if (hljs.getLanguage(language)) {
      return hljs.highlight(code, { language }).value
    }
    return hljs.highlightAuto(code).value
  }, [code, hasCode, language])

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // fallback for non-HTTPS contexts
      const ta = document.createElement('textarea')
      ta.value = code
      document.body.appendChild(ta)
      ta.select()
      const copiedByFallback = document.execCommand('copy')
      document.body.removeChild(ta)
      if (copiedByFallback) {
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }
    }
  }

  const lineCount = hasCode ? code.split('\n').length : 0

  return (
    <div className="rounded-lg border border-gray-700 overflow-hidden mt-3 shadow-sm">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-gray-800 text-gray-300 text-xs">
        <div className="flex items-center gap-2">
          <FileCode className="w-3.5 h-3.5" />
          <span className="font-medium">{filename}</span>
          <span className="text-gray-500">
            {language} &middot; {lineCount} lines
          </span>
        </div>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 px-2 py-1 rounded hover:bg-gray-700 transition-colors"
          title="Copy code"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-green-400" />
              <span className="text-green-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>
      {/* Code content */}
      <pre className="p-4 overflow-x-auto bg-gray-900 text-sm leading-relaxed m-0">
        {hasCode ? (
          <code
            className={`language-${language} hljs`}
            dangerouslySetInnerHTML={{ __html: highlighted }}
          />
        ) : (
          <code className={`language-${language} hljs text-gray-400`}>No code to preview</code>
        )}
      </pre>
    </div>
  )
}
