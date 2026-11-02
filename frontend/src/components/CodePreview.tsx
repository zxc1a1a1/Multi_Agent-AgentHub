import { useState, useMemo, useEffect } from 'react'
import { Copy, Check, FileCode } from 'lucide-react'
import type { BundledLanguage } from 'shiki'
import type { CodeBlock } from '../types'

interface Props {
  block: CodeBlock
}

interface HighlightToken {
  content: string
  color?: string
  fontStyle?: number
}

function normalizeLanguage(language: string): string {
  switch (language.trim().toLowerCase()) {
    case 'js':
      return 'javascript'
    case 'ts':
      return 'typescript'
    case 'py':
      return 'python'
    case 'sh':
    case 'shell':
      return 'bash'
    case '':
      return 'text'
    default:
      return language.trim().toLowerCase()
  }
}

function fontStyleFor(style?: number): 'italic' | 'normal' | undefined {
  if (!style) return undefined
  return style & 1 ? 'italic' : undefined
}

export default function CodePreview({ block }: Props) {
  const [copied, setCopied] = useState(false)
  const [tokens, setTokens] = useState<HighlightToken[][] | null>(null)
  const code = typeof block?.code === 'string' ? block.code : ''
  const language = typeof block?.language === 'string' && block.language.trim() ? block.language : 'text'
  const filename = typeof block?.filename === 'string' && block.filename.trim() ? block.filename : 'untitled'
  const hasCode = code.trim().length > 0

  // Shiki is loaded on demand so the initial bundle does not pay for grammars.
  // Rendering uses tokens (not HTML), so code can never inject markup.
  useEffect(() => {
    let cancelled = false
    setTokens(null)
    if (!hasCode) return
    const load = async () => {
      try {
        const shiki = await import('shiki')
        const lang = normalizeLanguage(language) as BundledLanguage
        try {
          const result = await shiki.codeToTokens(code, { lang, theme: 'github-dark' })
          if (!cancelled) setTokens(result.tokens)
        } catch {
          const fallback = await shiki.codeToTokens(code, { lang: 'text', theme: 'github-dark' })
          if (!cancelled) setTokens(fallback.tokens)
        }
      } catch {
        if (!cancelled) setTokens(null)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
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

  const lineCount = useMemo(() => (hasCode ? code.split('\n').length : 0), [code, hasCode])

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
      <pre className="p-4 overflow-x-auto bg-gray-900 text-sm leading-relaxed m-0 max-w-full">
        {hasCode ? (
          <code className={`language-${language} shiki break-words`}>
            {tokens ? (
              tokens.map((line, lineIndex) => (
                <span key={lineIndex} className="block min-h-[1em] whitespace-pre">
                  {line.length > 0
                    ? line.map((token, tokenIndex) => (
                        <span
                          key={tokenIndex}
                          style={{ color: token.color, fontStyle: fontStyleFor(token.fontStyle) }}
                        >
                          {token.content}
                        </span>
                      ))
                    : ' '}
                </span>
              ))
            ) : (
              <span className="whitespace-pre">{code}</span>
            )}
          </code>
        ) : (
          <code className={`language-${language} text-gray-400`}>No code to preview</code>
        )}
      </pre>
    </div>
  )
}
